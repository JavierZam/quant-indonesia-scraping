package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

// StockPriceRepo implements domain.StockPriceRepository for PostgreSQL.
type StockPriceRepo struct {
	pool *pgxpool.Pool
}

// NewStockPriceRepo creates a new stock price repository.
func NewStockPriceRepo(pool *pgxpool.Pool) *StockPriceRepo {
	return &StockPriceRepo{pool: pool}
}

// UpsertBatch inserts or updates stock prices.
func (r *StockPriceRepo) UpsertBatch(ctx context.Context, prices []domain.StockPrice) error {
	if len(prices) == 0 {
		return nil
	}

	query := `INSERT INTO stock_prices (symbol, date, close_price, open_price, high_price, low_price, volume)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)
			  ON CONFLICT (symbol, date) DO UPDATE SET
				  close_price = EXCLUDED.close_price,
				  open_price = COALESCE(EXCLUDED.open_price, stock_prices.open_price),
				  high_price = COALESCE(EXCLUDED.high_price, stock_prices.high_price),
				  low_price = COALESCE(EXCLUDED.low_price, stock_prices.low_price),
				  volume = COALESCE(EXCLUDED.volume, stock_prices.volume)`

	batch := &pgx.Batch{}
	for _, p := range prices {
		batch.Queue(query, p.Symbol, p.Date, p.ClosePrice, p.OpenPrice, p.HighPrice, p.LowPrice, p.Volume)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(prices); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("executing stock price upsert batch at index %d: %w", i, err)
		}
	}

	return nil
}

// ListBySymbol retrieves price history for a given symbol.
func (r *StockPriceRepo) ListBySymbol(ctx context.Context, symbol string, days int) ([]domain.StockPrice, error) {
	if days <= 0 {
		days = 30
	}

	query := `SELECT symbol, to_char(date, 'YYYY-MM-DD') as date, close_price, open_price, high_price, low_price, volume, created_at
			  FROM stock_prices
			  WHERE symbol = $1
			  ORDER BY date DESC
			  LIMIT $2`

	rows, err := r.pool.Query(ctx, query, symbol, days)
	if err != nil {
		return nil, fmt.Errorf("querying stock prices for %s: %w", symbol, err)
	}
	defer rows.Close()

	var prices []domain.StockPrice
	for rows.Next() {
		var p domain.StockPrice
		if err := rows.Scan(&p.Symbol, &p.Date, &p.ClosePrice, &p.OpenPrice, &p.HighPrice, &p.LowPrice, &p.Volume, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning stock price row: %w", err)
		}
		prices = append(prices, p)
	}

	return prices, rows.Err()
}

// UpsertBrokerSummaries inserts or updates broker summaries.
func (r *StockPriceRepo) UpsertBrokerSummaries(ctx context.Context, summaries []domain.BrokerSummary) error {
	if len(summaries) == 0 {
		return nil
	}

	query := `INSERT INTO broker_summaries (symbol, date, net_foreign_buy_sell, top_buyer, top_seller)
			  VALUES ($1, $2, $3, $4, $5)
			  ON CONFLICT (symbol, date) DO UPDATE SET
				  net_foreign_buy_sell = EXCLUDED.net_foreign_buy_sell,
				  top_buyer = COALESCE(EXCLUDED.top_buyer, broker_summaries.top_buyer),
				  top_seller = COALESCE(EXCLUDED.top_seller, broker_summaries.top_seller)`

	batch := &pgx.Batch{}
	for _, s := range summaries {
		batch.Queue(query, s.Symbol, s.Date, s.NetForeignBuySell, s.TopBuyer, s.TopSeller)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(summaries); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("executing broker summary upsert batch at index %d: %w", i, err)
		}
	}

	return nil
}

// ListBrokerSummaries retrieves broker summaries for a given symbol.
func (r *StockPriceRepo) ListBrokerSummaries(ctx context.Context, symbol string, days int) ([]domain.BrokerSummary, error) {
	if days <= 0 {
		days = 30
	}

	query := `SELECT id, symbol, to_char(date, 'YYYY-MM-DD') as date, net_foreign_buy_sell, top_buyer, top_seller, created_at
			  FROM broker_summaries
			  WHERE symbol = $1
			  ORDER BY date DESC
			  LIMIT $2`

	rows, err := r.pool.Query(ctx, query, symbol, days)
	if err != nil {
		return nil, fmt.Errorf("querying broker summaries for %s: %w", symbol, err)
	}
	defer rows.Close()

	var summaries []domain.BrokerSummary
	for rows.Next() {
		var s domain.BrokerSummary
		if err := rows.Scan(&s.ID, &s.Symbol, &s.Date, &s.NetForeignBuySell, &s.TopBuyer, &s.TopSeller, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning broker summary row: %w", err)
		}
		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}
