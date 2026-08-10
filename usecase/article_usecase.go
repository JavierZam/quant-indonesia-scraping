package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

// ArticleUsecase implements domain.ArticleUsecase.
type ArticleUsecase struct {
	articleRepo domain.ArticleRepository
	stockRepo   domain.StockRepository
	llmAnalyzer domain.LLMAnalyzer
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
			for _, sectorTag := range analysis.Tags {
				if sectorTag.Type == "sector" {
					stock.Sector = sectorTag.Value
					break
				}
			}

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
