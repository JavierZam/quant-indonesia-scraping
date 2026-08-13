package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/broadcaster"
)

// DefaultFeedSources defines standard Indonesian financial news RSS feeds.
var DefaultFeedSources = []FeedSource{
	{Name: "CNBC Indonesia Market", URL: "https://www.cnbcindonesia.com/market/rss"},
	{Name: "CNBC Indonesia News", URL: "https://www.cnbcindonesia.com/news/rss"},
	{Name: "Detik Finance", URL: "https://finance.detik.com/rss"},
	{Name: "IDX Channel", URL: "https://www.idxchannel.com/rss"},
	{Name: "Bisnis.com Market", URL: "https://market.bisnis.com/rss"},
	{Name: "Investor Daily", URL: "https://investor.id/rss"},
}

// Scheduler handles background automated ingestion runs.
type Scheduler struct {
	cron           *cron.Cron
	pipeline       *Pipeline
	articleUsecase domain.ArticleUsecase
	logger         *slog.Logger
	sources        []FeedSource
}

// NewScheduler creates a new ingestion scheduler.
func NewScheduler(
	pipeline *Pipeline,
	articleUsecase domain.ArticleUsecase,
	logger *slog.Logger,
	sources []FeedSource,
) *Scheduler {
	if len(sources) == 0 {
		sources = DefaultFeedSources
	}

	return &Scheduler{
		cron:           cron.New(),
		pipeline:       pipeline,
		articleUsecase: articleUsecase,
		logger:         logger,
		sources:        sources,
	}
}

// ScheduleJob adds a cron schedule (e.g., "*/30 * * * *") for automatic ingestion.
func (s *Scheduler) ScheduleJob(spec string) error {
	_, err := s.cron.AddFunc(spec, func() {
		s.logger.Info("Cron triggered automatic ingestion run", "sources", len(s.sources))
		ctx := context.Background()

		articles, errs := s.pipeline.Run(ctx, s.sources)
		for _, err := range errs {
			if err != nil {
				s.logger.Error("Cron ingestion pipeline error", "error", err)
			}
		}

		for _, article := range articles {
			if err := s.articleUsecase.Ingest(ctx, article); err != nil {
				s.logger.Error("Cron ingestion usecase error", "url", article.URL, "error", err)
			}
		}

		s.logger.Info("Cron automatic ingestion completed", "ingested", len(articles), "errors", len(errs))

		if s.pipeline.broadcaster != nil {
			s.pipeline.broadcaster.Broadcast(broadcaster.IngestionEvent{
				Type:    "done",
				Stage:   "SCRAPING",
				Total:   len(articles),
				Message: fmt.Sprintf("Cron Otomatis Selesai: %d berita diserap.", len(articles)),
			})
		}
	})

	return err
}

// Start begins executing scheduled cron jobs in the background.
func (s *Scheduler) Start() {
	s.logger.Info("Starting ingestion cron scheduler")
	s.cron.Start()
}

// Stop halts the cron scheduler gracefully.
func (s *Scheduler) Stop() context.Context {
	s.logger.Info("Stopping ingestion cron scheduler")
	return s.cron.Stop()
}
