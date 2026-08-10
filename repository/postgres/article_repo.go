package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

// ArticleRepo implements domain.ArticleRepository using PostgreSQL.
type ArticleRepo struct {
	pool *pgxpool.Pool
}

// NewArticleRepo creates a new PostgreSQL-backed article repository.
func NewArticleRepo(pool *pgxpool.Pool) *ArticleRepo {
	return &ArticleRepo{pool: pool}
}

// GetByID retrieves an article by UUID, including its stock tags.
func (r *ArticleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.NewsArticle, error) {
	query := `SELECT id, url_hash, title, url, source, summary, content_raw,
					 sentiment_score, sentiment_label, published_at, ingested_at,
					 processed_at, created_at, updated_at
			  FROM news_articles WHERE id = $1`

	article, err := r.scanArticle(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying article %s: %w", id, err)
	}

	// Load tags
	tags, err := r.loadTags(ctx, id)
	if err != nil {
		return nil, err
	}
	article.Tags = tags

	return article, nil
}

// GetByURLHash retrieves an article by its URL hash.
func (r *ArticleRepo) GetByURLHash(ctx context.Context, urlHash string) (*domain.NewsArticle, error) {
	query := `SELECT id, url_hash, title, url, source, summary, content_raw,
					 sentiment_score, sentiment_label, published_at, ingested_at,
					 processed_at, created_at, updated_at
			  FROM news_articles WHERE url_hash = $1`

	article, err := r.scanArticle(r.pool.QueryRow(ctx, query, urlHash))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying article by hash %s: %w", urlHash, err)
	}

	return article, nil
}

// List retrieves articles matching the given filter with pagination.
func (r *ArticleRepo) List(ctx context.Context, filter domain.ArticleFilter) ([]*domain.NewsArticle, error) {
	var (
		conditions []string
		args       []interface{}
		argIdx     int
	)

	// Build dynamic WHERE clause
	if filter.Symbol != "" {
		argIdx++
		conditions = append(conditions,
			fmt.Sprintf("id IN (SELECT news_id FROM news_stock_tags WHERE symbol = $%d)", argIdx))
		args = append(args, filter.Symbol)
	}

	if filter.SentimentLabel != nil {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("sentiment_label = $%d", argIdx))
		args = append(args, string(*filter.SentimentLabel))
	}

	if filter.Source != "" {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, filter.Source)
	}

	if filter.FromDate != nil {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("published_at >= $%d", argIdx))
		args = append(args, *filter.FromDate)
	}

	if filter.ToDate != nil {
		argIdx++
		conditions = append(conditions, fmt.Sprintf("published_at <= $%d", argIdx))
		args = append(args, *filter.ToDate)
	}

	query := `SELECT id, url_hash, title, url, source, summary, content_raw,
					 sentiment_score, sentiment_label, published_at, ingested_at,
					 processed_at, created_at, updated_at
			  FROM news_articles`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY published_at DESC NULLS LAST"

	// Apply pagination
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	argIdx++
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)

	if filter.Offset > 0 {
		argIdx++
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing articles: %w", err)
	}
	defer rows.Close()

	var articles []*domain.NewsArticle
	for rows.Next() {
		a, err := r.scanArticleFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning article row: %w", err)
		}
		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating article rows: %w", err)
	}

	return articles, nil
}

// Create inserts a new article.
func (r *ArticleRepo) Create(ctx context.Context, article *domain.NewsArticle) error {
	if article.ID == uuid.Nil {
		article.ID = uuid.New()
	}

	query := `INSERT INTO news_articles 
			  (id, url_hash, title, url, source, summary, content_raw,
			   sentiment_score, sentiment_label, published_at, ingested_at,
			   processed_at, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		article.ID, article.URLHash, article.Title, article.URL,
		article.Source, article.Summary, article.ContentRaw,
		article.SentimentScore, article.SentimentLabel,
		article.PublishedAt, article.IngestedAt, article.ProcessedAt,
		article.CreatedAt, article.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("creating article: %w", err)
	}

	return nil
}

// Update modifies an existing article's mutable fields.
func (r *ArticleRepo) Update(ctx context.Context, article *domain.NewsArticle) error {
	query := `UPDATE news_articles SET
				  summary = $2,
				  sentiment_score = $3,
				  sentiment_label = $4,
				  processed_at = $5,
				  updated_at = NOW()
			  WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		article.ID, article.Summary, article.SentimentScore,
		article.SentimentLabel, article.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("updating article %s: %w", article.ID, err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// CreateTags batch-inserts news-stock tag associations.
// Uses ON CONFLICT to handle duplicate tags gracefully.
func (r *ArticleRepo) CreateTags(ctx context.Context, tags []domain.NewsStockTag) error {
	if len(tags) == 0 {
		return nil
	}

	// Build batch insert
	var (
		valueParts []string
		args       []interface{}
	)

	for i, tag := range tags {
		base := i * 3
		valueParts = append(valueParts,
			fmt.Sprintf("($%d, $%d, $%d)", base+1, base+2, base+3))
		args = append(args, tag.NewsID, tag.Symbol, tag.RelevanceScore)
	}

	query := fmt.Sprintf(
		`INSERT INTO news_stock_tags (news_id, symbol, relevance_score)
		 VALUES %s
		 ON CONFLICT (news_id, symbol) DO UPDATE SET
			 relevance_score = EXCLUDED.relevance_score`,
		strings.Join(valueParts, ", "),
	)

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("creating article tags: %w", err)
	}

	return nil
}

// scanArticle scans a single article row from QueryRow.
func (r *ArticleRepo) scanArticle(row pgx.Row) (*domain.NewsArticle, error) {
	var a domain.NewsArticle
	err := row.Scan(
		&a.ID, &a.URLHash, &a.Title, &a.URL, &a.Source, &a.Summary,
		&a.ContentRaw, &a.SentimentScore, &a.SentimentLabel,
		&a.PublishedAt, &a.IngestedAt, &a.ProcessedAt,
		&a.CreatedAt, &a.UpdatedAt,
	)
	return &a, err
}

// scanArticleFromRows scans an article from a Rows iterator.
func (r *ArticleRepo) scanArticleFromRows(rows pgx.Rows) (*domain.NewsArticle, error) {
	var a domain.NewsArticle
	err := rows.Scan(
		&a.ID, &a.URLHash, &a.Title, &a.URL, &a.Source, &a.Summary,
		&a.ContentRaw, &a.SentimentScore, &a.SentimentLabel,
		&a.PublishedAt, &a.IngestedAt, &a.ProcessedAt,
		&a.CreatedAt, &a.UpdatedAt,
	)
	return &a, err
}

// loadTags loads all stock tags for a given article.
func (r *ArticleRepo) loadTags(ctx context.Context, articleID uuid.UUID) ([]domain.NewsStockTag, error) {
	query := `SELECT news_id, symbol, relevance_score, created_at 
			  FROM news_stock_tags WHERE news_id = $1 ORDER BY relevance_score DESC`

	rows, err := r.pool.Query(ctx, query, articleID)
	if err != nil {
		return nil, fmt.Errorf("loading tags for article %s: %w", articleID, err)
	}
	defer rows.Close()

	var tags []domain.NewsStockTag
	for rows.Next() {
		var t domain.NewsStockTag
		if err := rows.Scan(&t.NewsID, &t.Symbol, &t.RelevanceScore, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning tag row: %w", err)
		}
		tags = append(tags, t)
	}

	return tags, rows.Err()
}
