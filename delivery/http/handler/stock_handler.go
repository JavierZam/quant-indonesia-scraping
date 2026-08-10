package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

// StockHandler handles HTTP requests for stocks.
type StockHandler struct {
	usecase domain.StockUsecase
}

// NewStockHandler creates a new stock HTTP handler.
func NewStockHandler(uc domain.StockUsecase) *StockHandler {
	return &StockHandler{usecase: uc}
}

// GetBySymbol handles GET /api/v1/stocks/:symbol
func (h *StockHandler) GetBySymbol(c echo.Context) error {
	symbol := c.Param("symbol")
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "symbol is required",
		})
	}

	stock, err := h.usecase.GetBySymbol(c.Request().Context(), symbol)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.JSON(http.StatusNotFound, APIResponse{
				Success: false,
				Error:   "stock not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "internal server error",
		})
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stock,
	})
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
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "internal server error",
		})
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stocks,
		Meta: &Meta{
			Limit:  limit,
			Offset: offset,
			Count:  len(stocks),
		},
	})
}
