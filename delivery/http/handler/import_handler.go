package handler

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/yfinance"
)

// ImportHandler handles CSV and JSON data imports for prices and broker summaries.
type ImportHandler struct {
	stockPriceRepo domain.StockPriceRepository
	stockRepo      domain.StockRepository
	yfFetcher      *yfinance.Fetcher
	logger         *slog.Logger
}

// NewImportHandler creates a new ImportHandler.
func NewImportHandler(
	stockPriceRepo domain.StockPriceRepository,
	stockRepo domain.StockRepository,
	yfFetcher *yfinance.Fetcher,
	logger *slog.Logger,
) *ImportHandler {
	return &ImportHandler{
		stockPriceRepo: stockPriceRepo,
		stockRepo:      stockRepo,
		yfFetcher:      yfFetcher,
		logger:         logger,
	}
}

// PriceImportJSON represents a single price record in JSON import requests.
type PriceImportJSON struct {
	Symbol     string   `json:"symbol" validate:"required"`
	Date       string   `json:"date" validate:"required"` // YYYY-MM-DD
	ClosePrice float64  `json:"close_price" validate:"required"`
	OpenPrice  *float64 `json:"open_price,omitempty"`
	HighPrice  *float64 `json:"high_price,omitempty"`
	LowPrice   *float64 `json:"low_price,omitempty"`
	Volume     *int64   `json:"volume,omitempty"`
}

// BrokerImportJSON represents a single broker summary record in JSON import.
type BrokerImportJSON struct {
	Symbol            string  `json:"symbol" validate:"required"`
	Date              string  `json:"date" validate:"required"` // YYYY-MM-DD
	NetForeignBuySell float64 `json:"net_foreign_buy_sell" validate:"required"`
	TopBuyer          string  `json:"top_buyer,omitempty"`
	TopSeller         string  `json:"top_seller,omitempty"`
}

// ImportPrices handles POST /api/v1/import/prices (accepts JSON array or CSV file upload)
func (h *ImportHandler) ImportPrices(c echo.Context) error {
	ctx := c.Request().Context()
	var prices []domain.StockPrice

	contentType := c.Request().Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		// Handle CSV file upload
		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeBadRequest, "missing 'file' form field"))
		}
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeBadRequest, "opening file: "+err.Error()))
		}
		defer src.Close()

		parsed, err := parsePricesCSV(src)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "parsing CSV: "+err.Error()))
		}
		prices = parsed
	} else {
		// Handle JSON payload
		var req []PriceImportJSON
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeBadRequest, "invalid JSON payload: "+err.Error()))
		}
		for _, item := range req {
			prices = append(prices, domain.StockPrice{
				Symbol:     strings.ToUpper(strings.TrimSpace(item.Symbol)),
				Date:       strings.TrimSpace(item.Date),
				ClosePrice: item.ClosePrice,
				OpenPrice:  item.OpenPrice,
				HighPrice:  item.HighPrice,
				LowPrice:   item.LowPrice,
				Volume:     item.Volume,
			})
		}
	}

	if len(prices) == 0 {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "no valid price records to import"))
	}

	if err := h.stockPriceRepo.UpsertBatch(ctx, prices); err != nil {
		h.logger.Error("failed to import stock prices", "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to save price records: "+err.Error()))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(echo.Map{
		"imported": len(prices),
		"message":  fmt.Sprintf("Successfully imported %d price records", len(prices)),
	}, nil))
}

// FetchYahooPrices handles POST /api/v1/import/fetch-prices/:symbol
func (h *ImportHandler) FetchYahooPrices(c echo.Context) error {
	symbol := c.Param("symbol")
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "symbol parameter is required"))
	}

	ctx := c.Request().Context()
	rangeStr := c.QueryParam("range")
	prices, err := h.yfFetcher.FetchPrices(ctx, symbol, rangeStr)
	if err != nil {
		h.logger.Error("yahoo finance fetch failed", "symbol", symbol, "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to fetch prices from Yahoo Finance: "+err.Error()))
	}

	if len(prices) == 0 {
		return c.JSON(http.StatusNotFound, NewErrorResponse(ErrCodeNotFound, "no price data found for symbol "+symbol))
	}

	if err := h.stockPriceRepo.UpsertBatch(ctx, prices); err != nil {
		h.logger.Error("failed to save fetched Yahoo prices", "symbol", symbol, "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to save fetched prices: "+err.Error()))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(echo.Map{
		"symbol":   symbol,
		"imported": len(prices),
		"prices":   prices,
	}, nil))
}

// ImportBrokerSummary handles POST /api/v1/import/broker-summary (accepts JSON array or CSV file upload)
func (h *ImportHandler) ImportBrokerSummary(c echo.Context) error {
	ctx := c.Request().Context()
	var summaries []domain.BrokerSummary

	contentType := c.Request().Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeBadRequest, "missing 'file' form field"))
		}
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeBadRequest, "opening file: "+err.Error()))
		}
		defer src.Close()

		parsed, err := parseBrokerCSV(src)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "parsing CSV: "+err.Error()))
		}
		summaries = parsed
	} else {
		var req []BrokerImportJSON
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeBadRequest, "invalid JSON payload: "+err.Error()))
		}
		for _, item := range req {
			summaries = append(summaries, domain.BrokerSummary{
				Symbol:            strings.ToUpper(strings.TrimSpace(item.Symbol)),
				Date:              strings.TrimSpace(item.Date),
				NetForeignBuySell: item.NetForeignBuySell,
				TopBuyer:          strings.TrimSpace(item.TopBuyer),
				TopSeller:         strings.TrimSpace(item.TopSeller),
			})
		}
	}

	if len(summaries) == 0 {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "no valid broker summary records to import"))
	}

	if err := h.stockPriceRepo.UpsertBrokerSummaries(ctx, summaries); err != nil {
		h.logger.Error("failed to import broker summaries", "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to save broker summary records: "+err.Error()))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(echo.Map{
		"imported": len(summaries),
		"message":  fmt.Sprintf("Successfully imported %d broker summary records", len(summaries)),
	}, nil))
}

// RefreshAllPrices handles POST /api/v1/import/refresh-prices
// Fetches latest prices from Yahoo Finance for all tracked stocks.
func (h *ImportHandler) RefreshAllPrices(c echo.Context) error {
	ctx := c.Request().Context()

	symbols, err := h.stockRepo.ListAllSymbols(ctx)
	if err != nil {
		h.logger.Error("failed to list symbols", "error", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "failed to list stocks: "+err.Error()))
	}

	if len(symbols) == 0 {
		return c.JSON(http.StatusOK, NewSuccessResponse(echo.Map{
			"message": "No tracked stocks to update",
		}, nil))
	}

	// Run in background, return immediately
	go func() {
		bgCtx := context.Background()
		var successCount, errorCount int

		allSymbols := append([]string{"^JKSE"}, symbols...)

		for i, symbol := range allSymbols {
			if i > 0 {
				time.Sleep(1 * time.Second)
			}

			prices, err := h.yfFetcher.FetchPrices(bgCtx, symbol, "1mo")
			if err != nil {
				h.logger.Warn("refresh: failed to fetch price", "symbol", symbol, "error", err)
				errorCount++
				continue
			}

			if len(prices) > 0 {
				if err := h.stockPriceRepo.UpsertBatch(bgCtx, prices); err != nil {
					h.logger.Error("refresh: failed to save prices", "symbol", symbol, "error", err)
					errorCount++
					continue
				}
			}
			successCount++
		}

		h.logger.Info("Manual price refresh completed", "success", successCount, "errors", errorCount)
	}()

	return c.JSON(http.StatusAccepted, NewSuccessResponse(echo.Map{
		"message": fmt.Sprintf("Price refresh started for %d stocks in background", len(symbols)+1),
		"stocks":  len(symbols) + 1,
	}, nil))
}

// parsePricesCSV parses a CSV input with header: symbol,date,close_price,[open_price,high_price,low_price,volume]
func parsePricesCSV(r io.Reader) ([]domain.StockPrice, error) {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[strings.ToLower(strings.TrimSpace(col))] = i
	}

	symbolIdx, okSymbol := colIdx["symbol"]
	dateIdx, okDate := colIdx["date"]
	closeIdx, okClose := colIdx["close_price"]

	if !okSymbol || !okDate || !okClose {
		// Fallback check for alternate column names: close
		if closeIdx2, ok := colIdx["close"]; ok {
			closeIdx = closeIdx2
			okClose = true
		}
	}

	if !okSymbol || !okDate || !okClose {
		return nil, fmt.Errorf("CSV must contain 'symbol', 'date', and 'close_price' (or 'close') headers")
	}

	openIdx, hasOpen := colIdx["open_price"]
	if !hasOpen {
		openIdx, hasOpen = colIdx["open"]
	}
	highIdx, hasHigh := colIdx["high_price"]
	if !hasHigh {
		highIdx, hasHigh = colIdx["high"]
	}
	lowIdx, hasLow := colIdx["low_price"]
	if !hasLow {
		lowIdx, hasLow = colIdx["low"]
	}
	volIdx, hasVol := colIdx["volume"]

	var prices []domain.StockPrice
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		symbol := strings.ToUpper(strings.TrimSpace(record[symbolIdx]))
		dateStr := strings.TrimSpace(record[dateIdx])
		closeVal, err := strconv.ParseFloat(strings.TrimSpace(record[closeIdx]), 64)
		if err != nil || symbol == "" || dateStr == "" {
			continue
		}

		p := domain.StockPrice{
			Symbol:     symbol,
			Date:       dateStr,
			ClosePrice: closeVal,
		}

		if hasOpen && openIdx < len(record) {
			if val, err := strconv.ParseFloat(strings.TrimSpace(record[openIdx]), 64); err == nil {
				p.OpenPrice = &val
			}
		}
		if hasHigh && highIdx < len(record) {
			if val, err := strconv.ParseFloat(strings.TrimSpace(record[highIdx]), 64); err == nil {
				p.HighPrice = &val
			}
		}
		if hasLow && lowIdx < len(record) {
			if val, err := strconv.ParseFloat(strings.TrimSpace(record[lowIdx]), 64); err == nil {
				p.LowPrice = &val
			}
		}
		if hasVol && volIdx < len(record) {
			if val, err := strconv.ParseInt(strings.TrimSpace(record[volIdx]), 10, 64); err == nil {
				p.Volume = &val
			}
		}

		prices = append(prices, p)
	}

	return prices, nil
}

// parseBrokerCSV parses a CSV input with header: symbol,date,net_foreign_buy_sell,[top_buyer,top_seller]
func parseBrokerCSV(r io.Reader) ([]domain.BrokerSummary, error) {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[strings.ToLower(strings.TrimSpace(col))] = i
	}

	symbolIdx, okSymbol := colIdx["symbol"]
	dateIdx, okDate := colIdx["date"]
	netIdx, okNet := colIdx["net_foreign_buy_sell"]

	if !okSymbol || !okDate || !okNet {
		return nil, fmt.Errorf("CSV must contain 'symbol', 'date', and 'net_foreign_buy_sell' headers")
	}

	buyerIdx, hasBuyer := colIdx["top_buyer"]
	sellerIdx, hasSeller := colIdx["top_seller"]

	var summaries []domain.BrokerSummary
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		symbol := strings.ToUpper(strings.TrimSpace(record[symbolIdx]))
		dateStr := strings.TrimSpace(record[dateIdx])
		netVal, err := strconv.ParseFloat(strings.TrimSpace(record[netIdx]), 64)
		if err != nil || symbol == "" || dateStr == "" {
			continue
		}

		bs := domain.BrokerSummary{
			Symbol:            symbol,
			Date:              dateStr,
			NetForeignBuySell: netVal,
		}

		if hasBuyer && buyerIdx < len(record) {
			bs.TopBuyer = strings.TrimSpace(record[buyerIdx])
		}
		if hasSeller && sellerIdx < len(record) {
			bs.TopSeller = strings.TrimSpace(record[sellerIdx])
		}

		summaries = append(summaries, bs)
	}

	return summaries, nil
}

// Suppress unused import error
var _ = bytes.MinRead
