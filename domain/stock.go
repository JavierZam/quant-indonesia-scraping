package domain

import (
	"context"
	"time"
)

// Stock represents a publicly traded company.
type Stock struct {
	Symbol      string    `json:"symbol" db:"symbol"`
	CompanyName string    `json:"company_name" db:"company_name"`
	Sector      string    `json:"sector,omitempty" db:"sector"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// StockRepository defines the interface for stock persistence operations.
type StockRepository interface {
	GetBySymbol(ctx context.Context, symbol string) (*Stock, error)
	List(ctx context.Context, sector string, limit, offset int) ([]*Stock, error)
	Upsert(ctx context.Context, stock *Stock) error
}

// StockUsecase defines the business logic interface for stocks.
type StockUsecase interface {
	GetBySymbol(ctx context.Context, symbol string) (*Stock, error)
	List(ctx context.Context, sector string, limit, offset int) ([]*Stock, error)
}
