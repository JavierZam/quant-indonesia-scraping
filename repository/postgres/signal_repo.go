package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

// SignalRepo implements domain.SignalRepository using PostgreSQL.
type SignalRepo struct {
	pool *pgxpool.Pool
}

// NewSignalRepo creates a new PostgreSQL-backed signal repository.
func NewSignalRepo(pool *pgxpool.Pool) *SignalRepo {
	return &SignalRepo{pool: pool}
}

// GetStockSignals computes aggregated sentiment metrics and generates trading signals.
func (r *SignalRepo) GetStockSignals(ctx context.Context, filter domain.SignalFilter) ([]*domain.StockSignal, error) {
	periodDuration := 7 * 24 * time.Hour // Default 7d
	switch filter.Period {
	case "24h":
		periodDuration = 24 * time.Hour
	case "30d":
		periodDuration = 30 * 24 * time.Hour
	}
	cutoff := time.Now().Add(-periodDuration)

	var (
		conditions []string
		args       []interface{}
	)

	args = append(args, cutoff)
	conditions = append(conditions, fmt.Sprintf("a.published_at >= $%d", len(args)))

	if filter.Symbol != "" {
		args = append(args, filter.Symbol)
		conditions = append(conditions, fmt.Sprintf("s.symbol = $%d", len(args)))
	}

	if filter.Sector != "" {
		args = append(args, filter.Sector)
		conditions = append(conditions, fmt.Sprintf("s.sector = $%d", len(args)))
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			s.symbol,
			s.company_name,
			COALESCE(s.sector, '') AS sector,
			COALESCE(AVG(a.sentiment_score), 0.0) AS avg_score,
			COUNT(a.id) AS article_count,
			COUNT(CASE WHEN a.sentiment_label = 'Bullish' THEN 1 END) AS bullish_count,
			COUNT(CASE WHEN a.sentiment_label = 'Bearish' THEN 1 END) AS bearish_count,
			COUNT(CASE WHEN a.sentiment_label = 'Neutral' THEN 1 END) AS neutral_count
		FROM stocks s
		JOIN news_stock_tags nst ON s.symbol = nst.symbol
		JOIN news_articles a ON nst.news_id = a.id
		WHERE %s
		GROUP BY s.symbol, s.company_name, s.sector
		ORDER BY avg_score DESC`, whereClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying stock signals: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	var signals []*domain.StockSignal

	for rows.Next() {
		var s domain.StockSignal
		if err := rows.Scan(
			&s.Symbol,
			&s.CompanyName,
			&s.Sector,
			&s.AverageScore,
			&s.ArticleCount,
			&s.BullishArticles,
			&s.BearishArticles,
			&s.NeutralArticles,
		); err != nil {
			return nil, fmt.Errorf("scanning stock signal row: %w", err)
		}

		s.Period = filter.Period
		if s.Period == "" {
			s.Period = "7d"
		}
		s.GeneratedAt = now

		// Calculate recommendation signal
		if s.AverageScore >= 0.20 && s.BullishArticles > s.BearishArticles {
			s.Signal = domain.SignalBuy
		} else if s.AverageScore <= -0.20 && s.BearishArticles > s.BullishArticles {
			s.Signal = domain.SignalSell
		} else {
			s.Signal = domain.SignalHold
		}

		signals = append(signals, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating stock signal rows: %w", err)
	}

	return signals, nil
}

// GetStockSentimentHistory returns daily time-series sentiment data + price & broker flow for a stock.
func (r *SignalRepo) GetStockSentimentHistory(ctx context.Context, symbol string, days int) ([]*domain.SentimentTrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT 
			d.date_str,
			COALESCE(s.avg_score, 0.0) AS avg_score,
			COALESCE(s.article_count, 0) AS article_count,
			sp.close_price,
			bs.net_foreign_buy_sell
		FROM (
			SELECT DISTINCT TO_CHAR(published_at, 'YYYY-MM-DD') AS date_str
			FROM news_articles
			WHERE published_at >= $2
			UNION
			SELECT DISTINCT TO_CHAR(date, 'YYYY-MM-DD') AS date_str
			FROM stock_prices
			WHERE symbol = $1 AND date >= $2
		) d
		LEFT JOIN (
			SELECT 
				TO_CHAR(a.published_at, 'YYYY-MM-DD') AS date_str,
				AVG(a.sentiment_score) AS avg_score,
				COUNT(a.id) AS article_count
			FROM news_articles a
			JOIN news_stock_tags nst ON a.id = nst.news_id
			WHERE nst.symbol = $1 AND a.published_at >= $2
			GROUP BY date_str
		) s ON d.date_str = s.date_str
		LEFT JOIN stock_prices sp ON sp.symbol = $1 AND TO_CHAR(sp.date, 'YYYY-MM-DD') = d.date_str
		LEFT JOIN broker_summaries bs ON bs.symbol = $1 AND TO_CHAR(bs.date, 'YYYY-MM-DD') = d.date_str
		ORDER BY d.date_str ASC`

	rows, err := r.pool.Query(ctx, query, symbol, cutoff)
	if err != nil {
		return nil, fmt.Errorf("querying stock sentiment history for %s: %w", symbol, err)
	}
	defer rows.Close()

	var points []*domain.SentimentTrendPoint
	for rows.Next() {
		var p domain.SentimentTrendPoint
		if err := rows.Scan(&p.Date, &p.AverageScore, &p.ArticleCount, &p.ClosePrice, &p.NetForeignBuySell); err != nil {
			return nil, fmt.Errorf("scanning sentiment trend row: %w", err)
		}
		points = append(points, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sentiment trend rows: %w", err)
	}

	return points, nil
}
