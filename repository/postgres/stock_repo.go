package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

// StockRepo implements domain.StockRepository using PostgreSQL.
type StockRepo struct {
	pool *pgxpool.Pool
}

// NewStockRepo creates a new PostgreSQL-backed stock repository.
func NewStockRepo(pool *pgxpool.Pool) *StockRepo {
	return &StockRepo{pool: pool}
}

// GetBySymbol retrieves a stock by its ticker symbol.
func (r *StockRepo) GetBySymbol(ctx context.Context, symbol string) (*domain.Stock, error) {
	query := `SELECT symbol, company_name, sector, created_at, updated_at FROM stocks WHERE symbol = $1`

	var s domain.Stock
	err := r.pool.QueryRow(ctx, query, symbol).Scan(
		&s.Symbol, &s.CompanyName, &s.Sector, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying stock %s: %w", symbol, err)
	}

	return &s, nil
}

// List retrieves stocks optionally filtered by sector with pagination.
func (r *StockRepo) List(ctx context.Context, sector string, limit, offset int) ([]*domain.Stock, error) {
	var (
		query string
		args  []interface{}
	)

	if sector != "" {
		query = `SELECT symbol, company_name, sector, created_at, updated_at 
				 FROM stocks WHERE sector = $1 ORDER BY symbol LIMIT $2 OFFSET $3`
		args = []interface{}{sector, limit, offset}
	} else {
		query = `SELECT symbol, company_name, sector, created_at, updated_at 
				 FROM stocks ORDER BY symbol LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing stocks: %w", err)
	}
	defer rows.Close()

	var stocks []*domain.Stock
	for rows.Next() {
		var s domain.Stock
		if err := rows.Scan(&s.Symbol, &s.CompanyName, &s.Sector, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning stock row: %w", err)
		}
		stocks = append(stocks, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating stock rows: %w", err)
	}

	return stocks, nil
}

// Upsert inserts or updates a stock record.
func (r *StockRepo) Upsert(ctx context.Context, stock *domain.Stock) error {
	query := `INSERT INTO stocks (symbol, company_name, sector, created_at, updated_at)
			  VALUES ($1, $2, $3, NOW(), NOW())
			  ON CONFLICT (symbol) DO UPDATE SET
				  company_name = EXCLUDED.company_name,
				  sector = EXCLUDED.sector,
				  updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query, stock.Symbol, stock.CompanyName, stock.Sector)
	if err != nil {
		return fmt.Errorf("upserting stock %s: %w", stock.Symbol, err)
	}

	return nil
}
