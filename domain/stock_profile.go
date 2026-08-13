package domain

import (
	"context"
	"time"
)

type StockProfile struct {
	Symbol            string    `json:"symbol" db:"symbol"`
	Description       string    `json:"description,omitempty" db:"description"`
	Industry          string    `json:"industry,omitempty" db:"industry"`
	FoundedYear       *int      `json:"founded_year,omitempty" db:"founded_year"`
	City              string    `json:"city,omitempty" db:"city"`
	Country           string    `json:"country,omitempty" db:"country"`
	Employees         *int      `json:"employees,omitempty" db:"employees"`
	Website           string    `json:"website,omitempty" db:"website"`
	MarketCap         *int64    `json:"market_cap,omitempty" db:"market_cap"`
	SharesOutstanding *int64    `json:"shares_outstanding,omitempty" db:"shares_outstanding"`
	FloatShares       *int64    `json:"float_shares,omitempty" db:"float_shares"`
	TrailingPE        *float64  `json:"trailing_pe,omitempty" db:"trailing_pe"`
	PriceToBook       *float64  `json:"price_to_book,omitempty" db:"price_to_book"`
	TrailingEps       *float64  `json:"trailing_eps,omitempty" db:"trailing_eps"`
	DividendYield     *float64  `json:"dividend_yield,omitempty" db:"dividend_yield"`
	Week52High        *float64  `json:"week_52_high,omitempty" db:"week_52_high"`
	Week52Low         *float64  `json:"week_52_low,omitempty" db:"week_52_low"`
	TotalRevenue      *int64    `json:"total_revenue,omitempty" db:"total_revenue"`
	NetIncome         *int64    `json:"net_income,omitempty" db:"net_income"`
	TotalDebt         *int64    `json:"total_debt,omitempty" db:"total_debt"`
	TotalAssets       *int64    `json:"total_assets,omitempty" db:"total_assets"`
	ReturnOnEquity    *float64  `json:"return_on_equity,omitempty" db:"return_on_equity"`
	DebtToEquity      *float64  `json:"debt_to_equity,omitempty" db:"debt_to_equity"`
	FetchedAt         time.Time `json:"fetched_at" db:"fetched_at"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type StockProfileRepository interface {
	GetBySymbol(ctx context.Context, symbol string) (*StockProfile, error)
	Upsert(ctx context.Context, profile *StockProfile) error
}
