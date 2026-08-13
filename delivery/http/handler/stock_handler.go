package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/yfinance"
)

// StockHandler handles HTTP requests for stocks.
type StockHandler struct {
	usecase     domain.StockUsecase
	profileRepo domain.StockProfileRepository
	articleRepo domain.ArticleRepository
	stockRepo   domain.StockRepository
	yfFetcher   *yfinance.Fetcher
	logger      *slog.Logger
}

// NewStockHandler creates a new stock HTTP handler.
func NewStockHandler(
	uc domain.StockUsecase,
	profileRepo domain.StockProfileRepository,
	articleRepo domain.ArticleRepository,
	stockRepo domain.StockRepository,
	yfFetcher *yfinance.Fetcher,
	logger *slog.Logger,
) *StockHandler {
	return &StockHandler{
		usecase:     uc,
		profileRepo: profileRepo,
		articleRepo: articleRepo,
		stockRepo:   stockRepo,
		yfFetcher:   yfFetcher,
		logger:      logger,
	}
}

// GetBySymbol handles GET /api/v1/stocks/:symbol
func (h *StockHandler) GetBySymbol(c echo.Context) error {
	symbol := c.Param("symbol")
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "symbol is required"))
	}

	stock, err := h.usecase.GetBySymbol(c.Request().Context(), symbol)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.JSON(http.StatusNotFound, NewErrorResponse(ErrCodeNotFound, "stock not found"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "internal server error"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(stock, nil))
}

// List handles GET /api/v1/stocks
// Query params: sector, limit, offset
func (h *StockHandler) List(c echo.Context) error {
	sector := c.QueryParam("sector")
	limit := 20
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	stocks, err := h.usecase.List(c.Request().Context(), sector, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "internal server error"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(stocks, &Meta{
		Limit:  limit,
		Offset: offset,
		Count:  len(stocks),
	}))
}

// GetDetail handles GET /api/v1/stocks/:symbol/detail
func (h *StockHandler) GetDetail(c echo.Context) error {
	symbol := c.Param("symbol")
	ctx := c.Request().Context()

	stock, err := h.usecase.GetBySymbol(ctx, symbol)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.JSON(http.StatusNotFound, NewErrorResponse(ErrCodeNotFound, "stock not found"))
		}
		h.logger.Error("failed to get stock", "symbol", symbol, "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "internal server error"))
	}

	profile, err := h.profileRepo.GetBySymbol(ctx, symbol)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		h.logger.Error("failed to get profile", "symbol", symbol, "error", err)
	}

	if profile == nil || time.Since(profile.FetchedAt) > 24*time.Hour {
		yp, err := h.yfFetcher.FetchProfile(ctx, symbol)
		if err == nil && yp != nil {
			newProfile := &domain.StockProfile{
				Symbol:            stock.Symbol,
				Description:       yp.LongBusinessSummary,
				Industry:          yp.Industry,
				City:              yp.City,
				Country:           yp.Country,
				Website:           yp.Website,
				MarketCap:         &yp.MarketCap,
				SharesOutstanding: &yp.SharesOutstanding,
				FloatShares:       &yp.FloatShares,
				TrailingPE:        &yp.TrailingPE,
				PriceToBook:       &yp.PriceToBook,
				TrailingEps:       &yp.TrailingEps,
				DividendYield:     &yp.DividendYield,
				Week52High:        &yp.FiftyTwoWeekHigh,
				Week52Low:         &yp.FiftyTwoWeekLow,
				TotalRevenue:      &yp.TotalRevenue,
				NetIncome:         &yp.NetIncome,
				TotalDebt:         &yp.TotalDebt,
				TotalAssets:       &yp.TotalAssets,
				ReturnOnEquity:    &yp.ReturnOnEquity,
				DebtToEquity:      &yp.DebtToEquity,
			}
			if yp.FullTimeEmployees > 0 {
				newProfile.Employees = &yp.FullTimeEmployees
			}
			if err := h.profileRepo.Upsert(ctx, newProfile); err != nil {
				h.logger.Error("failed to upsert fetched profile", "symbol", symbol, "error", err)
			} else {
				profile = newProfile
			}
		} else {
			h.logger.Error("failed to fetch profile from yfinance", "symbol", symbol, "error", err)
		}
	}

	executives, err := h.stockRepo.ListExecutivesBySymbol(ctx, symbol)
	if err != nil {
		h.logger.Error("failed to list executives", "symbol", symbol, "error", err)
	}

	news, err := h.articleRepo.List(ctx, domain.ArticleFilter{
		Symbol: symbol,
		Limit:  10,
	})
	if err != nil {
		h.logger.Error("failed to list news", "symbol", symbol, "error", err)
	}

	detail := &domain.StockDetail{
		Stock:      stock,
		Profile:    profile,
		Executives: executives,
		News:       news,
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(detail, nil))
}

// GetNews handles GET /api/v1/stocks/:symbol/news
func (h *StockHandler) GetNews(c echo.Context) error {
	symbol := c.Param("symbol")
	ctx := c.Request().Context()

	limit := 20
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	news, err := h.articleRepo.List(ctx, domain.ArticleFilter{
		Symbol: symbol,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list news", "symbol", symbol, "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to get news"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(news, &Meta{
		Limit:  limit,
		Offset: offset,
		Count:  len(news),
	}))
}

// FetchProfile handles POST /api/v1/stocks/:symbol/profile
func (h *StockHandler) FetchProfile(c echo.Context) error {
	symbol := c.Param("symbol")
	ctx := c.Request().Context()

	yp, err := h.yfFetcher.FetchProfile(ctx, symbol)
	if err != nil {
		h.logger.Error("failed to fetch profile", "symbol", symbol, "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to fetch profile"))
	}

	newProfile := &domain.StockProfile{
		Symbol:            symbol,
		Description:       yp.LongBusinessSummary,
		Industry:          yp.Industry,
		City:              yp.City,
		Country:           yp.Country,
		Website:           yp.Website,
		MarketCap:         &yp.MarketCap,
		SharesOutstanding: &yp.SharesOutstanding,
		FloatShares:       &yp.FloatShares,
		TrailingPE:        &yp.TrailingPE,
		PriceToBook:       &yp.PriceToBook,
		TrailingEps:       &yp.TrailingEps,
		DividendYield:     &yp.DividendYield,
		Week52High:        &yp.FiftyTwoWeekHigh,
		Week52Low:         &yp.FiftyTwoWeekLow,
		TotalRevenue:      &yp.TotalRevenue,
		NetIncome:         &yp.NetIncome,
		TotalDebt:         &yp.TotalDebt,
		TotalAssets:       &yp.TotalAssets,
		ReturnOnEquity:    &yp.ReturnOnEquity,
		DebtToEquity:      &yp.DebtToEquity,
	}
	if yp.FullTimeEmployees > 0 {
		newProfile.Employees = &yp.FullTimeEmployees
	}

	if err := h.profileRepo.Upsert(ctx, newProfile); err != nil {
		h.logger.Error("failed to upsert profile", "symbol", symbol, "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to save profile"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(newProfile, nil))
}

// GetIHSG handles GET /api/v1/market/ihsg
func (h *StockHandler) GetIHSG(c echo.Context) error {
	ctx := c.Request().Context()

	prices, err := h.yfFetcher.FetchPrices(ctx, "^JKSE", "1mo")
	if err != nil {
		h.logger.Error("failed to fetch IHSG data", "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to fetch IHSG data: "+err.Error()))
	}

	if len(prices) == 0 {
		return c.JSON(http.StatusNotFound, NewErrorResponse(ErrCodeNotFound, "no IHSG data available"))
	}

	// Get latest price
	latest := prices[len(prices)-1]

	// Calculate daily change
	var prevClose float64
	var changeAbs, changePct float64
	if len(prices) >= 2 {
		prevClose = prices[len(prices)-2].ClosePrice
		changeAbs = latest.ClosePrice - prevClose
		if prevClose > 0 {
			changePct = (changeAbs / prevClose) * 100
		}
	}

	// Build sparkline data (last 30 data points)
	sparkline := make([]float64, 0, len(prices))
	sparkDates := make([]string, 0, len(prices))
	for _, p := range prices {
		sparkline = append(sparkline, p.ClosePrice)
		sparkDates = append(sparkDates, p.Date)
	}

	result := echo.Map{
		"symbol":      "IHSG",
		"name":        "Jakarta Composite Index",
		"price":       latest.ClosePrice,
		"change":      changeAbs,
		"change_pct":  changePct,
		"prev_close":  prevClose,
		"date":        latest.Date,
		"sparkline":   sparkline,
		"spark_dates": sparkDates,
	}

	// Add OHLV if available
	if latest.OpenPrice != nil {
		result["open"] = *latest.OpenPrice
	}
	if latest.HighPrice != nil {
		result["high"] = *latest.HighPrice
	}
	if latest.LowPrice != nil {
		result["low"] = *latest.LowPrice
	}
	if latest.Volume != nil {
		result["volume"] = *latest.Volume
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(result, nil))
}
