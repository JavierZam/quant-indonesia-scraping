package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	valkeylib "github.com/valkey-io/valkey-go"

	"github.com/javier-garcia/quant-indonesia-scraping/delivery/http/handler"
	"github.com/javier-garcia/quant-indonesia-scraping/delivery/http/middleware"
)

// RouterDeps holds all dependencies needed to configure the HTTP router.
type RouterDeps struct {
	ArticleHandler   *handler.ArticleHandler
	StockHandler     *handler.StockHandler
	IngestionHandler *handler.IngestionHandler
	ValkeyClient     valkeylib.Client
	Logger           *slog.Logger
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

	// Health check endpoint (required for Cloud Run)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	})

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

	// Ingestion routes (no caching — these are write operations)
	ingestionGroup := v1.Group("/ingestion")
	ingestionGroup.POST("/trigger", deps.IngestionHandler.Trigger)

	return e
}
