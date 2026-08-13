package usecase

import (
	"context"
	"fmt"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

// SignalUsecase implements domain.SignalUsecase.
type SignalUsecase struct {
	signalRepo domain.SignalRepository
}

// NewSignalUsecase creates a new signal usecase instance.
func NewSignalUsecase(signalRepo domain.SignalRepository) *SignalUsecase {
	return &SignalUsecase{signalRepo: signalRepo}
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

	return signals, nil
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
