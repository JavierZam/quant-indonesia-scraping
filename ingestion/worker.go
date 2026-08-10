package ingestion

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
	"github.com/javier-garcia/quant-indonesia-scraping/pkg/hasher"
)

// Job represents a single ingestion task to be processed by a worker.
type Job struct {
	FeedItem FeedItem
}

// Result captures the outcome of processing a single Job.
type Result struct {
	Article *domain.NewsArticle
	Err     error
	URL     string
}

// WorkerPool manages a pool of concurrent ingestion workers.
type WorkerPool struct {
	numWorkers int
	scraper    *Scraper
	dedupCache domain.DeduplicationCache
	logger     *slog.Logger

	jobs    chan Job
	results chan Result
}

// NewWorkerPool creates a new worker pool for ingesting articles.
func NewWorkerPool(
	numWorkers int,
	scraper *Scraper,
	dedupCache domain.DeduplicationCache,
	logger *slog.Logger,
) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		scraper:    scraper,
		dedupCache: dedupCache,
		logger:     logger,
		jobs:       make(chan Job, numWorkers*2),
		results:    make(chan Result, numWorkers*2),
	}
}

// Start launches the worker goroutines and returns when all workers are ready.
// Workers run until the context is cancelled or the jobs channel is closed.
func (wp *WorkerPool) Start(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < wp.numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			wp.worker(ctx, workerID)
		}(i)
	}

	// Close results channel when all workers finish
	go func() {
		wg.Wait()
		close(wp.results)
	}()

	wp.logger.Info("worker pool started", "workers", wp.numWorkers)
}

// Submit adds a job to the work queue. Blocks if the queue is full.
// Returns false if the context has been cancelled.
func (wp *WorkerPool) Submit(ctx context.Context, job Job) bool {
	select {
	case wp.jobs <- job:
		return true
	case <-ctx.Done():
		return false
	}
}

// Results returns the channel for reading processing results.
func (wp *WorkerPool) Results() <-chan Result {
	return wp.results
}

// Close signals that no more jobs will be submitted.
func (wp *WorkerPool) Close() {
	close(wp.jobs)
}

// worker is the main loop for a single worker goroutine.
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	wp.logger.Debug("worker started", "worker_id", id)

	for {
		select {
		case <-ctx.Done():
			wp.logger.Debug("worker stopping (context cancelled)", "worker_id", id)
			return
		case job, ok := <-wp.jobs:
			if !ok {
				wp.logger.Debug("worker stopping (jobs channel closed)", "worker_id", id)
				return
			}

			result := wp.processJob(ctx, job)

			select {
			case wp.results <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// processJob handles a single ingestion job: dedup check → scrape → build article.
func (wp *WorkerPool) processJob(ctx context.Context, job Job) Result {
	url := job.FeedItem.URL
	urlHash := hasher.MD5Hash(url)

	// Step 1: Check deduplication cache
	exists, err := wp.dedupCache.Exists(ctx, urlHash)
	if err != nil {
		wp.logger.Error("dedup check failed", "url", url, "error", err)
		return Result{URL: url, Err: err}
	}
	if exists {
		wp.logger.Debug("skipping duplicate url", "url", url)
		return Result{URL: url, Err: domain.ErrDuplicateURL}
	}

	// Step 2: Scrape article content
	content, err := wp.scraper.ScrapeArticle(ctx, url)
	if err != nil {
		wp.logger.Warn("scrape failed", "url", url, "error", err)
		// Don't mark as deduped on scrape failure — allow retry next cycle
		return Result{URL: url, Err: err}
	}

	// Step 3: Mark URL as processed in dedup cache
	if err := wp.dedupCache.Set(ctx, urlHash); err != nil {
		wp.logger.Error("failed to set dedup key", "url", url, "error", err)
		// Continue anyway — worst case we re-process once
	}

	// Step 4: Build the article domain entity
	now := time.Now()
	article := &domain.NewsArticle{
		ID:          uuid.New(),
		URLHash:     urlHash,
		Title:       job.FeedItem.Title,
		URL:         url,
		Source:      job.FeedItem.Source,
		Summary:     job.FeedItem.Summary,
		ContentRaw:  content,
		PublishedAt: job.FeedItem.PublishedAt,
		IngestedAt:  now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	wp.logger.Info("article ingested",
		"url", url,
		"title", article.Title,
		"source", article.Source,
		"content_length", len(content),
	)

	return Result{Article: article, URL: url}
}
