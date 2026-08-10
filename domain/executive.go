package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Executive represents a company executive.
type Executive struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Symbol    string    `json:"symbol" db:"symbol"`
	Name      string    `json:"name" db:"name"`
	Title     string    `json:"title,omitempty" db:"title"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ExecutiveRepository defines the interface for executive persistence operations.
type ExecutiveRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Executive, error)
	ListBySymbol(ctx context.Context, symbol string) ([]*Executive, error)
	Upsert(ctx context.Context, exec *Executive) error
}
