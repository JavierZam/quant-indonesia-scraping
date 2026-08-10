package ingestion

import (
	"context"
	"log/slog"
	"sync"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

// Pipeline orchestrates the full ingestion workflow:
// parse feeds → submit jobs → collect results.
type Pipeline struct {
	feedParser *FeedParser
	workerPool *WorkerPool
	logger     *slog.Logger
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

// Run executes the ingestion pipeline for the given feed sources.
// It returns all successfully ingested articles and any errors encountered.
func (p *Pipeline) Run(ctx context.Context, sources []FeedSource) ([]*domain.NewsArticle, []error) {
	p.logger.Info("ingestion pipeline starting", "sources", len(sources))

	// Start the worker pool
	p.workerPool.Start(ctx)

	// Feed submission goroutine
	go func() {
		defer p.workerPool.Close()

		for _, source := range sources {
			select {
			case <-ctx.Done():
				p.logger.Warn("feed submission cancelled")
				return
			default:
			}

			items, err := p.feedParser.ParseFeed(ctx, source)
			if err != nil {
				p.logger.Error("failed to parse feed",
					"source", source.Name,
					"error", err,
				)
				continue
			}

			for _, item := range items {
				if !p.workerPool.Submit(ctx, Job{FeedItem: item}) {
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

	for result := range p.workerPool.Results() {
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
