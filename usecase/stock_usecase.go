package usecase

import (
	"context"
	"fmt"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

// StockUsecase implements domain.StockUsecase.
type StockUsecase struct {
	stockRepo domain.StockRepository
}

// NewStockUsecase creates a new stock usecase.
func NewStockUsecase(stockRepo domain.StockRepository) *StockUsecase {
	return &StockUsecase{stockRepo: stockRepo}
}

// GetBySymbol retrieves a stock by its ticker symbol.
func (uc *StockUsecase) GetBySymbol(ctx context.Context, symbol string) (*domain.Stock, error) {
	stock, err := uc.stockRepo.GetBySymbol(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("getting stock %s: %w", symbol, err)
	}
	return stock, nil
}

// List retrieves stocks with optional sector filtering and pagination.
func (uc *StockUsecase) List(ctx context.Context, sector string, limit, offset int) ([]*domain.Stock, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	stocks, err := uc.stockRepo.List(ctx, sector, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing stocks: %w", err)
	}
	return stocks, nil
}
