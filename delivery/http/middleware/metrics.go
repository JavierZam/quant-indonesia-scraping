package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/JavierZam/quant-indonesia-scraping/pkg/metrics"
)

// MetricsMiddleware records HTTP request metrics for Prometheus.
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(c.Response().Status)
			method := c.Request().Method
			path := c.Path() // Use route pattern, not actual URL (avoids cardinality explosion)

			metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)

			return err
		}
	}
}
