package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// StockPrice represents historical daily price data for a stock.
type StockPrice struct {
	Symbol     string    `json:"symbol" db:"symbol"`
	Date       string    `json:"date" db:"date"` // YYYY-MM-DD
	ClosePrice float64   `json:"close_price" db:"close_price"`
	OpenPrice  *float64  `json:"open_price,omitempty" db:"open_price"`
	HighPrice  *float64  `json:"high_price,omitempty" db:"high_price"`
	LowPrice   *float64  `json:"low_price,omitempty" db:"low_price"`
	Volume     *int64    `json:"volume,omitempty" db:"volume"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// BrokerSummary represents daily net foreign flow and top broker activity.
type BrokerSummary struct {
	ID                uuid.UUID `json:"id" db:"id"`
	Symbol            string    `json:"symbol" db:"symbol"`
	Date              string    `json:"date" db:"date"` // YYYY-MM-DD
	NetForeignBuySell float64   `json:"net_foreign_buy_sell" db:"net_foreign_buy_sell"`
	TopBuyer          string    `json:"top_buyer,omitempty" db:"top_buyer"`
	TopSeller         string    `json:"top_seller,omitempty" db:"top_seller"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// CombinedTimeSeriesPoint represents sentiment + price + foreign flow for a specific date.
type CombinedTimeSeriesPoint struct {
	Date              string   `json:"date"`
	AverageScore      float64  `json:"average_score"`
	ArticleCount      int      `json:"article_count"`
	ClosePrice        *float64 `json:"close_price,omitempty"`
	NetForeignBuySell *float64 `json:"net_foreign_buy_sell,omitempty"`
}

// StockPriceRepository defines operations for price & broker persistence.
type StockPriceRepository interface {
	UpsertBatch(ctx context.Context, prices []StockPrice) error
	ListBySymbol(ctx context.Context, symbol string, days int) ([]StockPrice, error)
	UpsertBrokerSummaries(ctx context.Context, summaries []BrokerSummary) error
	ListBrokerSummaries(ctx context.Context, symbol string, days int) ([]BrokerSummary, error)
}
