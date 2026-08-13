package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

// ExecutiveRepo implements domain.ExecutiveRepository using PostgreSQL.
type ExecutiveRepo struct {
	pool *pgxpool.Pool
}

// NewExecutiveRepo creates a new PostgreSQL-backed executive repository.
func NewExecutiveRepo(pool *pgxpool.Pool) *ExecutiveRepo {
	return &ExecutiveRepo{pool: pool}
}

// GetByID retrieves an executive by their UUID.
func (r *ExecutiveRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Executive, error) {
	query := `SELECT id, symbol, name, title, created_at, updated_at FROM executives WHERE id = $1`

	var e domain.Executive
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.Symbol, &e.Name, &e.Title, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying executive %s: %w", id, err)
	}

	return &e, nil
}

// ListBySymbol retrieves all executives associated with a stock symbol.
func (r *ExecutiveRepo) ListBySymbol(ctx context.Context, symbol string) ([]*domain.Executive, error) {
	query := `SELECT id, symbol, name, title, created_at, updated_at 
			  FROM executives WHERE symbol = $1 ORDER BY name`

	rows, err := r.pool.Query(ctx, query, symbol)
	if err != nil {
		return nil, fmt.Errorf("listing executives for %s: %w", symbol, err)
	}
	defer rows.Close()

	var execs []*domain.Executive
	for rows.Next() {
		var e domain.Executive
		if err := rows.Scan(&e.ID, &e.Symbol, &e.Name, &e.Title, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning executive row: %w", err)
		}
		execs = append(execs, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating executive rows: %w", err)
	}

	return execs, nil
}

// Upsert inserts or updates an executive record.
// Uses name + symbol as the natural key for conflict resolution.
func (r *ExecutiveRepo) Upsert(ctx context.Context, exec *domain.Executive) error {
	query := `INSERT INTO executives (id, symbol, name, title, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, NOW(), NOW())
			  ON CONFLICT (id) DO UPDATE SET
				  name = EXCLUDED.name,
				  title = EXCLUDED.title,
				  updated_at = NOW()`

	if exec.ID == uuid.Nil {
		exec.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query, exec.ID, exec.Symbol, exec.Name, exec.Title)
	if err != nil {
		return fmt.Errorf("upserting executive %s: %w", exec.Name, err)
	}

	return nil
}
