package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/broadcaster"
)

// Pipeline orchestrates the full ingestion workflow:
// parse feeds → submit jobs → collect results.
type Pipeline struct {
	feedParser  *FeedParser
	workerPool  *WorkerPool
	logger      *slog.Logger
	broadcaster *broadcaster.Broadcaster
}

// NewPipeline creates a new ingestion pipeline.
func NewPipeline(
	feedParser *FeedParser,
	workerPool *WorkerPool,
	logger *slog.Logger,
) *Pipeline {
	return &Pipeline{
		feedParser: feedParser,
		workerPool: workerPool,
		logger:     logger,
	}
}

// SetBroadcaster attaches an SSE broadcaster for real-time UI progress updates.
func (p *Pipeline) SetBroadcaster(b *broadcaster.Broadcaster) {
	p.broadcaster = b
}

// Run executes the ingestion pipeline for the given feed sources.
// It returns all successfully ingested articles and any errors encountered.
func (p *Pipeline) Run(ctx context.Context, sources []FeedSource) ([]*domain.NewsArticle, []error) {
	p.logger.Info("ingestion pipeline starting", "sources", len(sources))

	// Instantiate a fresh WorkerPool for this run to avoid channel reuse races
	wp := NewWorkerPool(p.workerPool.numWorkers, p.workerPool.scraper, p.workerPool.dedupCache, p.logger)
	wp.Start(ctx)

	// Feed submission goroutine
	go func() {
		defer wp.Close()

		for i, source := range sources {
			select {
			case <-ctx.Done():
				p.logger.Warn("feed submission cancelled")
				return
			default:
			}

			if p.broadcaster != nil {
				p.broadcaster.Broadcast(broadcaster.IngestionEvent{
					Type:    "progress",
					Stage:   "SCRAPING",
					Current: i + 1,
					Total:   len(sources),
					Message: fmt.Sprintf("Scraping RSS Feed [%d/%d]: %s...", i+1, len(sources), source.Name),
				})
			}

			items, err := p.feedParser.ParseFeed(ctx, source)
			if err != nil {
				p.logger.Error("failed to parse feed",
					"source", source.Name,
					"error", err,
				)
				continue
			}

			// Limit to 20 latest items per feed source for fast, responsive ingestion
			if len(items) > 20 {
				items = items[:20]
			}

			for _, item := range items {
				if !wp.Submit(ctx, Job{FeedItem: item}) {
					p.logger.Warn("job submission cancelled", "url", item.URL)
					return
				}
			}
		}
	}()

	// Collect results
	var (
		articles []*domain.NewsArticle
		errs     []error
		mu       sync.Mutex
	)

	for result := range wp.Results() {
		mu.Lock()
		if result.Err != nil {
			errs = append(errs, result.Err)
		} else if result.Article != nil {
			articles = append(articles, result.Article)
		}
		mu.Unlock()
	}

	p.logger.Info("ingestion pipeline completed",
		"articles_ingested", len(articles),
		"errors", len(errs),
	)

	return articles, errs
}
