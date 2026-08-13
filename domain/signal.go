package domain

import (
	"context"
	"time"
)

// SignalType represents the trading signal recommendation.
type SignalType string

const (
	SignalBuy  SignalType = "BUY"
	SignalSell SignalType = "SELL"
	SignalHold SignalType = "HOLD"
)

// StockSignal holds aggregated sentiment metrics and trading signal for a stock.
type StockSignal struct {
	Symbol          string     `json:"symbol"`
	CompanyName     string     `json:"company_name"`
	Sector          string     `json:"sector,omitempty"`
	Signal          SignalType `json:"signal"`
	AverageScore    float64    `json:"average_score"`
	ArticleCount    int        `json:"article_count"`
	BullishArticles int        `json:"bullish_articles"`
	BearishArticles int        `json:"bearish_articles"`
	NeutralArticles int        `json:"neutral_articles"`
	Period          string     `json:"period"`
	GeneratedAt     time.Time  `json:"generated_at"`

	// Technical indicators
	MA20            *float64   `json:"ma20,omitempty"`
	MA50            *float64   `json:"ma50,omitempty"`
	RSI14           *float64   `json:"rsi14,omitempty"`
	LastPrice       *float64   `json:"last_price,omitempty"`
	PriceChangePct  *float64   `json:"price_change_pct,omitempty"`
	TechnicalSignal SignalType `json:"technical_signal,omitempty"`

	// Composite quant score
	CompositeScore  float64    `json:"composite_score"`
	Confidence      float64    `json:"confidence"`
}

// SentimentTrendPoint represents a daily data point for historical sentiment charts and price correlation.
type SentimentTrendPoint struct {
	Date              string   `json:"date"` // YYYY-MM-DD
	AverageScore      float64  `json:"average_score"`
	ArticleCount      int      `json:"article_count"`
	ClosePrice        *float64 `json:"close_price,omitempty"`
	NetForeignBuySell *float64 `json:"net_foreign_buy_sell,omitempty"`
}

// SignalFilter provides filtering options for querying stock signals.
type SignalFilter struct {
	Symbol string
	Sector string
	Period string // "24h", "7d", "30d" (default: "7d")
}

// SignalRepository defines database queries for analytics and signals.
type SignalRepository interface {
	GetStockSignals(ctx context.Context, filter SignalFilter) ([]*StockSignal, error)
	GetStockSentimentHistory(ctx context.Context, symbol string, days int) ([]*SentimentTrendPoint, error)
}

// SignalUsecase defines business logic for stock signals and sentiment analytics.
type SignalUsecase interface {
	GetSignals(ctx context.Context, filter SignalFilter) ([]*StockSignal, error)
	GetHistory(ctx context.Context, symbol string, days int) ([]*SentimentTrendPoint, error)
}
