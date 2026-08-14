package http

import (
	"log/slog"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	valkeylib "github.com/valkey-io/valkey-go"

	"github.com/JavierZam/quant-indonesia-scraping/delivery/http/handler"
	"github.com/JavierZam/quant-indonesia-scraping/delivery/http/middleware"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/broadcaster"
)

// RouterDeps holds all dependencies needed to configure the HTTP router.
type RouterDeps struct {
	ArticleHandler   *handler.ArticleHandler
	StockHandler     *handler.StockHandler
	IngestionHandler *handler.IngestionHandler
	HealthHandler    *handler.HealthHandler
	SignalHandler    *handler.SignalHandler
	ImportHandler    *handler.ImportHandler
	Broadcaster      *broadcaster.Broadcaster
	ValkeyClient     valkeylib.Client
	Logger           *slog.Logger
	RateLimitRPS     int
	RateLimitBurst   int
}

// NewRouter creates and configures the Echo HTTP router with all routes registered.
func NewRouter(deps RouterDeps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(echomw.Logger())
	e.Use(echomw.CORS())
	e.Use(echomw.GzipWithConfig(echomw.GzipConfig{
		Level: 5,
	}))

	// Prometheus metrics middleware
	e.Use(middleware.MetricsMiddleware())

	// Rate limiter middleware
	if deps.RateLimitRPS > 0 {
		e.Use(middleware.RateLimiterMiddleware(middleware.RateLimiterConfig{
			RPS:   deps.RateLimitRPS,
			Burst: deps.RateLimitBurst,
		}))
	}

	// Health check endpoints
	e.GET("/health", deps.HealthHandler.Liveness)  // backward compat
	e.GET("/healthz", deps.HealthHandler.Liveness) // liveness probe
	e.GET("/readyz", deps.HealthHandler.Readiness)  // readiness probe

	// Prometheus metrics endpoint
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// API v1 group
	v1 := e.Group("/api/v1")

	// Caching middleware for read-heavy endpoints (30s TTL)
	cacheMiddleware := middleware.CacheWithConfig(middleware.CacheConfig{
		TTL:    30 * time.Second,
		Client: deps.ValkeyClient,
		Logger: deps.Logger,
	})

	// Article routes
	articles := v1.Group("/articles")
	articles.GET("", deps.ArticleHandler.List, cacheMiddleware)
	articles.GET("/:id", deps.ArticleHandler.GetByID, cacheMiddleware)

	// Stock routes
	stocks := v1.Group("/stocks")
	stocks.GET("", deps.StockHandler.List, cacheMiddleware)
	stocks.GET("/:symbol", deps.StockHandler.GetBySymbol, cacheMiddleware)
	stocks.GET("/:symbol/detail", deps.StockHandler.GetDetail, cacheMiddleware)
	stocks.GET("/:symbol/news", deps.StockHandler.GetNews, cacheMiddleware)
	stocks.POST("/:symbol/profile", deps.StockHandler.FetchProfile)

	// Signal routes
	signals := v1.Group("/signals")
	signals.GET("", deps.SignalHandler.GetSignals, cacheMiddleware)
	signals.GET("/:symbol/history", deps.SignalHandler.GetHistory, cacheMiddleware)

	// Market data routes
	market := v1.Group("/market")
	market.GET("/ihsg", deps.StockHandler.GetIHSG, cacheMiddleware)

	// Ingestion routes (no caching — these are write operations)
	ingestionGroup := v1.Group("/ingestion")
	ingestionGroup.POST("/trigger", deps.IngestionHandler.Trigger)
	ingestionGroup.POST("/reprocess", deps.IngestionHandler.Reprocess)
	ingestionGroup.POST("/cancel", deps.IngestionHandler.Cancel)
	if deps.Broadcaster != nil {
		ingestionGroup.GET("/stream", deps.Broadcaster.Handler)
	}

	// Import routes (prices & broker data)
	if deps.ImportHandler != nil {
		importGroup := v1.Group("/import")
		importGroup.POST("/prices", deps.ImportHandler.ImportPrices)
		importGroup.POST("/fetch-prices/:symbol", deps.ImportHandler.FetchYahooPrices)
		importGroup.POST("/broker-summary", deps.ImportHandler.ImportBrokerSummary)
		importGroup.POST("/refresh-prices", deps.ImportHandler.RefreshAllPrices)
	}

	// Serve Static Web Dashboard with no-cache headers for instant updates
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if path == "/" || strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
				c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Response().Header().Set("Pragma", "no-cache")
				c.Response().Header().Set("Expires", "0")
			}
			return next(c)
		}
	})
	e.Static("/", "web")

	return e
}
