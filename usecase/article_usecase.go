package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/broadcaster"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/sector"
)

// ArticleUsecase implements domain.ArticleUsecase.
type ArticleUsecase struct {
	articleRepo domain.ArticleRepository
	stockRepo   domain.StockRepository
	llmAnalyzer domain.LLMAnalyzer
	broadcaster *broadcaster.Broadcaster
	logger      *slog.Logger
}

// NewArticleUsecase creates a new article usecase with all required dependencies.
func NewArticleUsecase(
	articleRepo domain.ArticleRepository,
	stockRepo   domain.StockRepository,
	llmAnalyzer domain.LLMAnalyzer,
	logger      *slog.Logger,
) *ArticleUsecase {
	return &ArticleUsecase{
		articleRepo: articleRepo,
		stockRepo:   stockRepo,
		llmAnalyzer: llmAnalyzer,
		logger:      logger,
	}
}

// SetBroadcaster sets the SSE broadcaster for progress updates.
func (uc *ArticleUsecase) SetBroadcaster(b *broadcaster.Broadcaster) {
	uc.broadcaster = b
}

// GetByID retrieves a single article by ID.
func (uc *ArticleUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.NewsArticle, error) {
	article, err := uc.articleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting article %s: %w", id, err)
	}
	return article, nil
}

// List retrieves articles matching the given filter.
func (uc *ArticleUsecase) List(ctx context.Context, filter domain.ArticleFilter) ([]*domain.NewsArticle, error) {
	articles, err := uc.articleRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("listing articles: %w", err)
	}
	return articles, nil
}

// Ingest processes a raw article through the full pipeline:
// 1. Persist the raw article
// 2. Send content to LLM for analysis
// 3. Update the article with sentiment analysis results
// 4. Upsert discovered stock entities
// 5. Create news-stock tag associations
func (uc *ArticleUsecase) Ingest(ctx context.Context, article *domain.NewsArticle) error {
	if article == nil {
		return fmt.Errorf("%w: article is nil", domain.ErrInvalidInput)
	}

	uc.logger.Info("ingesting article",
		"url", article.URL,
		"title", article.Title,
	)

	// Step 1: Persist the raw article first (before LLM processing)
	if err := uc.articleRepo.Create(ctx, article); err != nil {
		return fmt.Errorf("persisting raw article: %w", err)
	}

	// Step 2: Run LLM analysis on the article content
	analysis, err := uc.llmAnalyzer.Analyze(ctx, article.Title, article.ContentRaw)
	if err != nil {
		uc.logger.Error("LLM analysis failed, article saved without analysis",
			"article_id", article.ID,
			"error", err,
		)
		// Don't fail the entire ingestion if LLM fails
		// The article is already saved and can be re-processed later
		return nil
	}

	// Step 3: Update article with analysis results
	now := time.Now()
	article.SentimentScore = &analysis.SentimentScore
	article.SentimentLabel = &analysis.SentimentLabel
	if analysis.Summary != "" {
		article.Summary = analysis.Summary
	}
	article.ProcessedAt = &now

	if err := uc.articleRepo.Update(ctx, article); err != nil {
		uc.logger.Error("failed to update article with analysis",
			"article_id", article.ID,
			"error", err,
		)
		return fmt.Errorf("updating article with analysis: %w", err)
	}

	// Step 4: Process extracted tags — upsert stocks and create associations
	var tags []domain.NewsStockTag
	for _, tag := range analysis.Tags {
		if tag.Type == "company" && tag.TickerSymbol != "" {
			// Upsert the stock entity
			stock := &domain.Stock{
				Symbol:      tag.TickerSymbol,
				CompanyName: tag.Value,
			}

			// Try to determine sector from tags
			rawSector := ""
			for _, sectorTag := range analysis.Tags {
				if sectorTag.Type == "sector" {
					rawSector = sectorTag.Value
					break
				}
			}
			stock.Sector = sector.NormalizeSector(tag.TickerSymbol, rawSector)

			if err := uc.stockRepo.Upsert(ctx, stock); err != nil {
				uc.logger.Warn("failed to upsert stock",
					"symbol", tag.TickerSymbol,
					"error", err,
				)
				// Continue processing other tags
				continue
			}

			// Build the tag association
			tags = append(tags, domain.NewsStockTag{
				NewsID:         article.ID,
				Symbol:         tag.TickerSymbol,
				RelevanceScore: tag.RelevanceScore,
			})
		}
	}

	// Upsert executive entities associated with primary company ticker
	primarySymbol := ""
	for _, tag := range analysis.Tags {
		if tag.Type == "company" && tag.TickerSymbol != "" {
			primarySymbol = tag.TickerSymbol
			break
		}
	}
	if primarySymbol != "" {
		for _, tag := range analysis.Tags {
			if tag.Type == "executive" && tag.Value != "" {
				if execErr := uc.stockRepo.UpsertExecutive(ctx, &domain.Executive{
					Symbol: primarySymbol,
					Name:   tag.Value,
					Title:  "Key Figure / Executive",
				}); execErr != nil {
					uc.logger.Warn("failed to upsert executive", "name", tag.Value, "symbol", primarySymbol, "error", execErr)
				}
			}
		}
	}

	// Step 5: Create tag associations
	if len(tags) > 0 {
		if err := uc.articleRepo.CreateTags(ctx, tags); err != nil {
			uc.logger.Error("failed to create article tags",
				"article_id", article.ID,
				"tags", len(tags),
				"error", err,
			)
			return fmt.Errorf("creating article tags: %w", err)
		}
	}

	uc.logger.Info("article fully processed",
		"article_id", article.ID,
		"sentiment", article.SentimentLabel,
		"score", article.SentimentScore,
		"tags", len(tags),
	)

	return nil
}

// ReprocessUnanalyzed finds articles that were saved without LLM analysis and processes them.
func (uc *ArticleUsecase) ReprocessUnanalyzed(ctx context.Context) (int, []error) {
	// Limit to 10 articles per batch so processing finishes fast (~25 seconds)
	articles, err := uc.articleRepo.ListUnprocessed(ctx, 10)
	if err != nil {
		return 0, []error{fmt.Errorf("listing unprocessed articles: %w", err)}
	}

	if len(articles) == 0 {
		uc.logger.Info("no unprocessed articles found")
		if uc.broadcaster != nil {
			uc.broadcaster.Broadcast(broadcaster.IngestionEvent{
				Type:    "done",
				Stage:   "REPROCESS",
				Total:   0,
				Message: "No unanalyzed articles found.",
			})
		}
		return 0, nil
	}

	uc.logger.Info("reprocessing unanalyzed articles", "count", len(articles))

	if uc.broadcaster != nil {
		uc.broadcaster.Broadcast(broadcaster.IngestionEvent{
			Type:    "start",
			Stage:   "REPROCESS",
			Total:   len(articles),
			Message: fmt.Sprintf("Mulai memproses batch %d artikel dengan Gemini AI...", len(articles)),
		})
	}

	var (
		successCount int
		errs         []error
	)

	for i, article := range articles {
		if uc.broadcaster != nil {
			uc.broadcaster.Broadcast(broadcaster.IngestionEvent{
				Type:    "progress",
				Stage:   "AI_ANALYSIS",
				Current: i + 1,
				Total:   len(articles),
				Message: fmt.Sprintf("Gemini AI Menganalisis [%d/%d]: \"%s\"", i+1, len(articles), article.Title),
			})
		}

		// Rate limit: wait 2.0s between requests to stay safe under 20 RPM free tier
		if i > 0 {
			select {
			case <-time.After(2000 * time.Millisecond):
			case <-ctx.Done():
				uc.logger.Warn("reprocess cancelled", "processed", successCount)
				return successCount, errs
			}
		}

		// Run LLM analysis
		analysis, err := uc.llmAnalyzer.Analyze(ctx, article.Title, article.ContentRaw)
		if err != nil {
			uc.logger.Error("LLM reprocess failed",
				"article_id", article.ID,
				"title", article.Title,
				"error", err,
			)
			if uc.broadcaster != nil {
				uc.broadcaster.Broadcast(broadcaster.IngestionEvent{
					Type:    "error",
					Stage:   "AI_ANALYSIS",
					Current: i + 1,
					Total:   len(articles),
					Message: fmt.Sprintf("❌ AI Gagal [%d/%d]: %s (%v)", i+1, len(articles), article.Title, err),
				})
			}
			errs = append(errs, fmt.Errorf("article %s: %w", article.ID, err))
			continue
		}

		// Update article with analysis results
		now := time.Now()
		article.SentimentScore = &analysis.SentimentScore
		article.SentimentLabel = &analysis.SentimentLabel
		if analysis.Summary != "" {
			article.Summary = analysis.Summary
		}
		article.ProcessedAt = &now

		if err := uc.articleRepo.Update(ctx, article); err != nil {
			uc.logger.Error("failed to update reprocessed article",
				"article_id", article.ID,
				"error", err,
			)
			errs = append(errs, fmt.Errorf("updating article %s: %w", article.ID, err))
			continue
		}

		// Process tags
		var tags []domain.NewsStockTag
		for _, tag := range analysis.Tags {
			if tag.Type == "company" && tag.TickerSymbol != "" {
				stock := &domain.Stock{
					Symbol:      tag.TickerSymbol,
					CompanyName: tag.Value,
				}
				rawSector := ""
				for _, sectorTag := range analysis.Tags {
					if sectorTag.Type == "sector" {
						rawSector = sectorTag.Value
						break
					}
				}
				stock.Sector = sector.NormalizeSector(tag.TickerSymbol, rawSector)
				if err := uc.stockRepo.Upsert(ctx, stock); err != nil {
					uc.logger.Warn("failed to upsert stock during reprocess",
						"symbol", tag.TickerSymbol,
						"error", err,
					)
					continue
				}
				tags = append(tags, domain.NewsStockTag{
					NewsID:         article.ID,
					Symbol:         tag.TickerSymbol,
					RelevanceScore: tag.RelevanceScore,
				})
			}
		}

		// Upsert executive entities during reprocess
		primarySymbol := ""
		for _, tag := range analysis.Tags {
			if tag.Type == "company" && tag.TickerSymbol != "" {
				primarySymbol = tag.TickerSymbol
				break
			}
		}
		if primarySymbol != "" {
			for _, tag := range analysis.Tags {
				if tag.Type == "executive" && tag.Value != "" {
					_ = uc.stockRepo.UpsertExecutive(ctx, &domain.Executive{
						Symbol: primarySymbol,
						Name:   tag.Value,
						Title:  "Key Figure / Executive",
					})
				}
			}
		}

		if len(tags) > 0 {
			if err := uc.articleRepo.CreateTags(ctx, tags); err != nil {
				uc.logger.Error("failed to create tags during reprocess",
					"article_id", article.ID,
					"error", err,
				)
			}
		}

		successCount++
		uc.logger.Info("article reprocessed successfully",
			"article_id", article.ID,
			"sentiment", article.SentimentLabel,
			"score", article.SentimentScore,
			"tags", len(tags),
		)

		if uc.broadcaster != nil {
			uc.broadcaster.Broadcast(broadcaster.IngestionEvent{
				Type:      "log",
				Stage:     "AI_ANALYSIS",
				Current:   successCount,
				Total:     len(articles),
				Message:   fmt.Sprintf("✅ Selesai [%d/%d]: %s (%s | Score: %+.2f)", successCount, len(articles), article.Title, *article.SentimentLabel, *article.SentimentScore),
				Sentiment: string(*article.SentimentLabel),
				Score:     article.SentimentScore,
			})
		}
	}

	if uc.broadcaster != nil {
		uc.broadcaster.Broadcast(broadcaster.IngestionEvent{
			Type:    "done",
			Stage:   "REPROCESS",
			Current: successCount,
			Total:   len(articles),
			Message: fmt.Sprintf("🎉 Reprocess Selesai! %d/%d artikel berhasil dianalisis AI.", successCount, len(articles)),
		})
	}

	return successCount, errs
}

