package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

type StockProfileRepo struct {
	pool *pgxpool.Pool
}

func NewStockProfileRepo(pool *pgxpool.Pool) *StockProfileRepo {
	return &StockProfileRepo{pool: pool}
}

func (r *StockProfileRepo) GetBySymbol(ctx context.Context, symbol string) (*domain.StockProfile, error) {
	query := `SELECT 
		symbol, description, industry, founded_year, city, country, employees, website, 
		market_cap, shares_outstanding, float_shares, trailing_pe, price_to_book, trailing_eps, dividend_yield, 
		week_52_high, week_52_low, total_revenue, net_income, total_debt, total_assets, return_on_equity, debt_to_equity, 
		fetched_at, created_at, updated_at
		FROM stock_profiles WHERE symbol = $1`

	var p domain.StockProfile
	err := r.pool.QueryRow(ctx, query, symbol).Scan(
		&p.Symbol, &p.Description, &p.Industry, &p.FoundedYear, &p.City, &p.Country, &p.Employees, &p.Website,
		&p.MarketCap, &p.SharesOutstanding, &p.FloatShares, &p.TrailingPE, &p.PriceToBook, &p.TrailingEps, &p.DividendYield,
		&p.Week52High, &p.Week52Low, &p.TotalRevenue, &p.NetIncome, &p.TotalDebt, &p.TotalAssets, &p.ReturnOnEquity, &p.DebtToEquity,
		&p.FetchedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying stock profile %s: %w", symbol, err)
	}

	return &p, nil
}

func (r *StockProfileRepo) Upsert(ctx context.Context, p *domain.StockProfile) error {
	query := `INSERT INTO stock_profiles (
		symbol, description, industry, founded_year, city, country, employees, website, 
		market_cap, shares_outstanding, float_shares, trailing_pe, price_to_book, trailing_eps, dividend_yield, 
		week_52_high, week_52_low, total_revenue, net_income, total_debt, total_assets, return_on_equity, debt_to_equity, 
		fetched_at, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, 
		$9, $10, $11, $12, $13, $14, $15, 
		$16, $17, $18, $19, $20, $21, $22, $23, 
		NOW(), NOW(), NOW()
	) ON CONFLICT (symbol) DO UPDATE SET 
		description = EXCLUDED.description,
		industry = EXCLUDED.industry,
		founded_year = EXCLUDED.founded_year,
		city = EXCLUDED.city,
		country = EXCLUDED.country,
		employees = EXCLUDED.employees,
		website = EXCLUDED.website,
		market_cap = EXCLUDED.market_cap,
		shares_outstanding = EXCLUDED.shares_outstanding,
		float_shares = EXCLUDED.float_shares,
		trailing_pe = EXCLUDED.trailing_pe,
		price_to_book = EXCLUDED.price_to_book,
		trailing_eps = EXCLUDED.trailing_eps,
		dividend_yield = EXCLUDED.dividend_yield,
		week_52_high = EXCLUDED.week_52_high,
		week_52_low = EXCLUDED.week_52_low,
		total_revenue = EXCLUDED.total_revenue,
		net_income = EXCLUDED.net_income,
		total_debt = EXCLUDED.total_debt,
		total_assets = EXCLUDED.total_assets,
		return_on_equity = EXCLUDED.return_on_equity,
		debt_to_equity = EXCLUDED.debt_to_equity,
		fetched_at = NOW(),
		updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query,
		p.Symbol, p.Description, p.Industry, p.FoundedYear, p.City, p.Country, p.Employees, p.Website,
		p.MarketCap, p.SharesOutstanding, p.FloatShares, p.TrailingPE, p.PriceToBook, p.TrailingEps, p.DividendYield,
		p.Week52High, p.Week52Low, p.TotalRevenue, p.NetIncome, p.TotalDebt, p.TotalAssets, p.ReturnOnEquity, p.DebtToEquity,
	)
	if err != nil {
		return fmt.Errorf("upserting stock profile %s: %w", p.Symbol, err)
	}

	return nil
}
