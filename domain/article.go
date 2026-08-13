package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SentimentLabel represents the sentiment classification of a news article.
type SentimentLabel string

const (
	SentimentBullish SentimentLabel = "Bullish"
	SentimentBearish SentimentLabel = "Bearish"
	SentimentNeutral SentimentLabel = "Neutral"
)

// NewsArticle represents an ingested and processed news article.
type NewsArticle struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	URLHash        string          `json:"url_hash" db:"url_hash"`
	Title          string          `json:"title" db:"title"`
	URL            string          `json:"url" db:"url"`
	Source         string          `json:"source,omitempty" db:"source"`
	Summary        string          `json:"summary,omitempty" db:"summary"`
	ContentRaw     string          `json:"-" db:"content_raw"`
	SentimentScore *float64        `json:"sentiment_score,omitempty" db:"sentiment_score"`
	SentimentLabel *SentimentLabel `json:"sentiment_label,omitempty" db:"sentiment_label"`
	PublishedAt    *time.Time      `json:"published_at,omitempty" db:"published_at"`
	IngestedAt     time.Time       `json:"ingested_at" db:"ingested_at"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	Tags           []NewsStockTag  `json:"tags,omitempty" db:"-"`
}

// NewsStockTag represents the many-to-many relationship between news and stocks.
type NewsStockTag struct {
	NewsID         uuid.UUID `json:"news_id" db:"news_id"`
	Symbol         string    `json:"symbol" db:"symbol"`
	RelevanceScore float64   `json:"relevance_score" db:"relevance_score"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ArticleFilter provides filtering options for querying articles.
type ArticleFilter struct {
	Symbol         string
	SentimentLabel *SentimentLabel
	Source         string
	FromDate       *time.Time
	ToDate         *time.Time
	Limit          int
	Offset         int
}

// ArticleRepository defines the interface for article persistence operations.
type ArticleRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*NewsArticle, error)
	GetByURLHash(ctx context.Context, urlHash string) (*NewsArticle, error)
	List(ctx context.Context, filter ArticleFilter) ([]*NewsArticle, error)
	ListUnprocessed(ctx context.Context, limit int) ([]*NewsArticle, error)
	Create(ctx context.Context, article *NewsArticle) error
	Update(ctx context.Context, article *NewsArticle) error
	CreateTags(ctx context.Context, tags []NewsStockTag) error
}

// ArticleUsecase defines the business logic interface for articles.
type ArticleUsecase interface {
	GetByID(ctx context.Context, id uuid.UUID) (*NewsArticle, error)
	List(ctx context.Context, filter ArticleFilter) ([]*NewsArticle, error)
	Ingest(ctx context.Context, article *NewsArticle) error
	ReprocessUnanalyzed(ctx context.Context) (int, []error)
}

// DeduplicationCache defines the interface for URL dedup checks using Valkey.
type DeduplicationCache interface {
	Exists(ctx context.Context, urlHash string) (bool, error)
	Set(ctx context.Context, urlHash string) error
}
