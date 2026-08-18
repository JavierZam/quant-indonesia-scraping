package usecase

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/quant"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/technical"
)

// SignalUsecase implements domain.SignalUsecase.
type SignalUsecase struct {
	signalRepo     domain.SignalRepository
	stockPriceRepo domain.StockPriceRepository
}

// NewSignalUsecase creates a new signal usecase instance.
func NewSignalUsecase(signalRepo domain.SignalRepository, stockPriceRepo domain.StockPriceRepository) *SignalUsecase {
	return &SignalUsecase{
		signalRepo:     signalRepo,
		stockPriceRepo: stockPriceRepo,
	}
}

// GetSignals retrieves stock signals matching the filter.
func (uc *SignalUsecase) GetSignals(ctx context.Context, filter domain.SignalFilter) ([]*domain.StockSignal, error) {
	if filter.Period == "" {
		filter.Period = "7d"
	}

	signals, err := uc.signalRepo.GetStockSignals(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("getting stock signals: %w", err)
	}

	if signals == nil {
		signals = make([]*domain.StockSignal, 0)
	}

	// Enrich each signal concurrently with technical indicators and composite scoring
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // max 20 concurrent goroutines

	for _, sig := range signals {
		wg.Add(1)
		go func(s *domain.StockSignal) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			uc.enrichWithTechnicals(ctx, s)
		}(sig)
	}
	wg.Wait()

	// Sort signals strictly by CompositeScore DESCENDING (highest score first)
	sort.Slice(signals, func(i, j int) bool {
		return signals[i].CompositeScore > signals[j].CompositeScore
	})

	return signals, nil
}

func (uc *SignalUsecase) enrichWithTechnicals(ctx context.Context, sig *domain.StockSignal) {
	// Fetch price history (60 days for MA50 + buffer)
	prices, err := uc.stockPriceRepo.ListBySymbol(ctx, sig.Symbol, 90)
	if err != nil || len(prices) < 15 {
		// Even without price history, set composite score to sentiment score
		sig.CompositeScore = sig.AverageScore
		return
	}

	// Extract close prices (oldest first for technical indicators)
	closePrices := make([]float64, len(prices))
	for i, p := range prices {
		closePrices[len(prices)-1-i] = p.ClosePrice
	}

	// Compute technical indicators
	tech := technical.Analyze(closePrices)
	if tech == nil {
		sig.CompositeScore = sig.AverageScore
		return
	}

	// Populate signal fields
	sig.MA20 = tech.MA20
	sig.MA50 = tech.MA50
	sig.RSI14 = tech.RSI14
	lastPrice := tech.LastPrice
	sig.LastPrice = &lastPrice
	priceChange := tech.PriceChange
	sig.PriceChangePct = &priceChange
	sig.TechnicalSignal = domain.SignalType(tech.Signal)

	// Compute composite score
	var priceVsMA20, priceVsMA50 *float64
	if tech.MA20 != nil && *tech.MA20 > 0 {
		v := tech.LastPrice / *tech.MA20
		priceVsMA20 = &v
	}
	if tech.MA50 != nil && *tech.MA50 > 0 {
		v := tech.LastPrice / *tech.MA50
		priceVsMA50 = &v
	}

	compositeInput := quant.CompositeInput{
		SentimentScore: sig.AverageScore,
		RSI14:          tech.RSI14,
		PriceVsMA20:    priceVsMA20,
		PriceVsMA50:    priceVsMA50,
		MACDHist:       tech.MACDHist,
		TechSignal:     tech.Signal,
		PriceChangePct: tech.PriceChange,
	}

	result := quant.ComputeComposite(compositeInput)
	sig.CompositeScore = result.CompositeScore
	sig.Confidence = result.Confidence

	// Override signal with composite signal
	sig.Signal = domain.SignalType(result.Signal)
}

// GetHistory retrieves historical daily sentiment points for a stock symbol.
func (uc *SignalUsecase) GetHistory(ctx context.Context, symbol string, days int) ([]*domain.SentimentTrendPoint, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidInput)
	}
	if days <= 0 {
		days = 30
	}

	history, err := uc.signalRepo.GetStockSentimentHistory(ctx, symbol, days)
	if err != nil {
		return nil, fmt.Errorf("getting sentiment history for %s: %w", symbol, err)
	}

	return history, nil
}
