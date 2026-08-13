package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds rate limiter configuration.
type RateLimiterConfig struct {
	RPS   int // Requests per second per IP
	Burst int // Maximum burst size
}

// ipLimiter pairs a token-bucket limiter with the timestamp of the last
// request from that IP, so stale entries can be reaped.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// staleThreshold is the duration after which an IP entry is considered stale
// and eligible for cleanup.
const staleThreshold = 3 * time.Minute

// RateLimiterMiddleware creates a per-IP rate limiting middleware.
//
// Each unique client IP gets its own token bucket configured with the
// supplied RPS and burst values. When a client exceeds its limit the
// middleware responds with 429 Too Many Requests, a Retry-After header,
// and a JSON error body.
//
// A background goroutine runs every minute to remove entries for IPs that
// have not been seen for 3 minutes, preventing unbounded memory growth.
func RateLimiterMiddleware(cfg RateLimiterConfig) echo.MiddlewareFunc {
	var visitors sync.Map // map[string]*ipLimiter

	// Start background goroutine to clean up stale IP entries.
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			visitors.Range(func(key, value any) bool {
				v, ok := value.(*ipLimiter)
				if !ok {
					return true
				}
				if now.Sub(v.lastSeen) > staleThreshold {
					visitors.Delete(key)
				}
				return true
			})
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			// Retrieve or create a limiter for this IP.
			val, loaded := visitors.LoadOrStore(ip, &ipLimiter{
				limiter:  rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst),
				lastSeen: time.Now(),
			})

			v := val.(*ipLimiter)

			// Always update lastSeen so the cleanup goroutine knows the
			// IP is still active.
			if loaded {
				v.lastSeen = time.Now()
			}

			if !v.limiter.Allow() {
				// Compute a Retry-After value based on the token
				// replenishment rate (1 / RPS seconds).
				retryAfter := "1"
				if cfg.RPS > 0 {
					retryAfter = time.Duration(
						time.Second.Nanoseconds() / int64(cfg.RPS),
					).Truncate(time.Second).String()
					// Ensure at least "1s" so the header is meaningful.
					if retryAfter == "0s" {
						retryAfter = "1"
					}
				}
				c.Response().Header().Set("Retry-After", retryAfter)

				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"success": false,
					"error":   "rate limit exceeded, try again later",
				})
			}

			return next(c)
		}
	}
}
