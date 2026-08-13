package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	valkeylib "github.com/valkey-io/valkey-go"
)

const (
	cacheKeyPrefix = "api:cache:"
)

// CacheConfig configures the caching middleware.
type CacheConfig struct {
	TTL    time.Duration
	Client valkeylib.Client
	Logger *slog.Logger
}

// CacheWithConfig returns a middleware that caches GET responses in Valkey.
// Only successful (2xx) JSON responses are cached.
func CacheWithConfig(cfg CacheConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Bypass cache if client is not configured
			if cfg.Client == nil {
				return next(c)
			}

			// Only cache GET requests
			if c.Request().Method != http.MethodGet {
				return next(c)
			}

			// Build cache key from path + query string
			key := buildCacheKey(c)

			// Try to get from cache
			cached, err := getFromCache(c.Request().Context(), cfg.Client, key)
			if err == nil && cached != "" {
				cfg.Logger.Debug("cache hit", "key", key)
				c.Response().Header().Set("X-Cache", "HIT")
				c.Response().Header().Set("Content-Type", "application/json")
				return c.String(http.StatusOK, cached)
			}

			// Cache miss — capture response
			cfg.Logger.Debug("cache miss", "key", key)
			c.Response().Header().Set("X-Cache", "MISS")

			// Use a response recorder to capture the output
			rec := &responseRecorder{
				ResponseWriter: c.Response().Writer,
				body:           &strings.Builder{},
			}
			c.Response().Writer = rec

			// Call next handler
			if err := next(c); err != nil {
				return err
			}

			// Only cache successful responses
			if rec.statusCode >= 200 && rec.statusCode < 300 {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					if setErr := setInCache(ctx, cfg.Client, key, rec.body.String(), cfg.TTL); setErr != nil {
						cfg.Logger.Warn("failed to set cache", "key", key, "error", setErr)
					}
				}()
			}

			return nil
		}
	}
}

// responseRecorder captures the response body for caching.
type responseRecorder struct {
	http.ResponseWriter
	body       *strings.Builder
	statusCode int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// buildCacheKey creates a deterministic cache key from the request.
func buildCacheKey(c echo.Context) string {
	path := c.Request().URL.Path
	query := c.Request().URL.RawQuery
	if query != "" {
		return cacheKeyPrefix + path + "?" + query
	}
	return cacheKeyPrefix + path
}

// getFromCache retrieves a cached value from Valkey with a strict 100ms timeout.
func getFromCache(ctx context.Context, client valkeylib.Client, key string) (string, error) {
	cacheCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	cmd := client.B().Get().Key(key).Build()
	result, err := client.Do(cacheCtx, cmd).ToString()
	if err != nil {
		return "", err
	}
	return result, nil
}

// setInCache stores a value in Valkey with the given TTL.
func setInCache(ctx context.Context, client valkeylib.Client, key, value string, ttl time.Duration) error {
	cmd := client.B().Set().Key(key).Value(value).Ex(ttl).Build()
	if err := client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("setting cache key %s: %w", key, err)
	}
	return nil
}
