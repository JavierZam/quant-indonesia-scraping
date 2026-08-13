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
	UpsertExecutive(ctx context.Context, exec *Executive) error
	ListExecutivesBySymbol(ctx context.Context, symbol string) ([]*Executive, error)
	ListAllSymbols(ctx context.Context) ([]string, error)
}

// StockUsecase defines the business logic interface for stocks.
type StockUsecase interface {
	GetBySymbol(ctx context.Context, symbol string) (*Stock, error)
	List(ctx context.Context, sector string, limit, offset int) ([]*Stock, error)
}

// StockDetail is a combined view for the stock detail panel.
type StockDetail struct {
	Stock      *Stock         `json:"stock"`
	Profile    *StockProfile  `json:"profile,omitempty"`
	Executives []*Executive   `json:"executives,omitempty"`
	News       []*NewsArticle `json:"news,omitempty"`
}
