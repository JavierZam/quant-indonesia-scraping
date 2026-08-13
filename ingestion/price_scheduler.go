package ingestion

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/yfinance"
)

// PriceScheduler handles background automated price fetching from Yahoo Finance.
type PriceScheduler struct {
	cron           *cron.Cron
	stockRepo      domain.StockRepository
	stockPriceRepo domain.StockPriceRepository
	yfFetcher      *yfinance.Fetcher
	logger         *slog.Logger
}

// NewPriceScheduler creates a new price update scheduler.
func NewPriceScheduler(
	stockRepo domain.StockRepository,
	stockPriceRepo domain.StockPriceRepository,
	yfFetcher *yfinance.Fetcher,
	logger *slog.Logger,
) *PriceScheduler {
	return &PriceScheduler{
		cron:           cron.New(),
		stockRepo:      stockRepo,
		stockPriceRepo: stockPriceRepo,
		yfFetcher:      yfFetcher,
		logger:         logger,
	}
}

// ScheduleJob adds a cron schedule for automatic price fetching.
func (s *PriceScheduler) ScheduleJob(spec string) error {
	_, err := s.cron.AddFunc(spec, func() {
		s.logger.Info("Cron triggered automatic price update")
		ctx := context.Background()

		// Fetch all tracked symbols from DB
		symbols, err := s.stockRepo.ListAllSymbols(ctx)
		if err != nil {
			s.logger.Error("Failed to list symbols for price update", "error", err)
			return
		}

		if len(symbols) == 0 {
			s.logger.Info("No tracked stocks to update prices for")
			return
		}

		s.logger.Info("Starting price update for tracked stocks", "count", len(symbols))

		var successCount, errorCount int

		// Also fetch IHSG index
		allSymbols := append([]string{"^JKSE"}, symbols...)

		for i, symbol := range allSymbols {
			// Rate limit: 1s between requests to avoid Yahoo Finance blocking
			if i > 0 {
				time.Sleep(1 * time.Second)
			}

			prices, err := s.yfFetcher.FetchPrices(ctx, symbol, "1mo")
			if err != nil {
				s.logger.Warn("Failed to fetch price from Yahoo Finance",
					"symbol", symbol,
					"error", err,
				)
				errorCount++
				continue
			}

			if len(prices) == 0 {
				continue
			}

			if err := s.stockPriceRepo.UpsertBatch(ctx, prices); err != nil {
				s.logger.Error("Failed to save prices",
					"symbol", symbol,
					"error", err,
				)
				errorCount++
				continue
			}

			successCount++
			s.logger.Debug("Price updated",
				"symbol", symbol,
				"records", len(prices),
			)
		}

		s.logger.Info("Cron price update completed",
			"success", successCount,
			"errors", errorCount,
			"total", len(allSymbols),
		)
	})

	return err
}

// Start begins executing scheduled cron jobs.
func (s *PriceScheduler) Start() {
	s.logger.Info("Starting price update cron scheduler")
	s.cron.Start()
}

// Stop halts the cron scheduler.
func (s *PriceScheduler) Stop() context.Context {
	s.logger.Info("Stopping price update cron scheduler")
	return s.cron.Stop()
}
