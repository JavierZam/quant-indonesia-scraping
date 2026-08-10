package httpclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultUserAgent = "QuantIntelBot/1.0"
	baseBackoff      = 500 * time.Millisecond
	maxBackoff       = 30 * time.Second
)

// Config holds HTTP client configuration.
type Config struct {
	Timeout      time.Duration
	MaxRetries   int
	RateLimitRPS int
	UserAgent    string
}

// Client wraps http.Client with retry logic and rate limiting.
type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	maxRetries int
	userAgent  string
	logger     *slog.Logger
}

// New creates a new HTTP client with the given configuration.
func New(cfg Config, logger *slog.Logger) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RateLimitRPS <= 0 {
		cfg.RateLimitRPS = 5
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		limiter:    rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitRPS),
		maxRetries: cfg.MaxRetries,
		userAgent:  cfg.UserAgent,
		logger:     logger,
	}
}

// Get performs a GET request with retry and rate limiting.
// The caller is responsible for closing the returned ReadCloser.
func (c *Client) Get(ctx context.Context, url string) ([]byte, int, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := c.calculateBackoff(attempt)
			c.logger.Debug("retrying request",
				"url", url,
				"attempt", attempt,
				"backoff", backoff,
			)

			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Wait for rate limiter
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, 0, fmt.Errorf("rate limiter: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("executing request: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Always close, even on ReadAll error

		if err != nil {
			lastErr = fmt.Errorf("reading response body: %w", err)
			continue
		}

		// Don't retry client errors (4xx), except 429 (Too Many Requests)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return nil, resp.StatusCode, fmt.Errorf("client error: HTTP %d for %s", resp.StatusCode, url)
		}

		// Retry on server errors (5xx) and 429
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("server error: HTTP %d for %s", resp.StatusCode, url)
			continue
		}

		return body, resp.StatusCode, nil
	}

	return nil, 0, fmt.Errorf("max retries (%d) exceeded: %w", c.maxRetries, lastErr)
}

// calculateBackoff computes exponential backoff with jitter.
func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := time.Duration(float64(baseBackoff) * math.Pow(2, float64(attempt-1)))
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	// Add jitter: 50-100% of computed backoff
	jitter := time.Duration(rand.Int64N(int64(backoff / 2)))
	return backoff/2 + jitter
}
