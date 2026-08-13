package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

// SignalHandler handles HTTP requests for signals and sentiment analytics.
type SignalHandler struct {
	usecase domain.SignalUsecase
}

// NewSignalHandler creates a new signal HTTP handler.
func NewSignalHandler(uc domain.SignalUsecase) *SignalHandler {
	return &SignalHandler{usecase: uc}
}

// GetSignals handles GET /api/v1/signals
// Query params: symbol, sector, period ("24h", "7d", "30d")
func (h *SignalHandler) GetSignals(c echo.Context) error {
	filter := domain.SignalFilter{
		Symbol: c.QueryParam("symbol"),
		Sector: c.QueryParam("sector"),
		Period: c.QueryParam("period"),
	}

	signals, err := h.usecase.GetSignals(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "internal server error"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(signals, nil))
}

// GetHistory handles GET /api/v1/signals/:symbol/history
// Query params: days (default 30)
func (h *SignalHandler) GetHistory(c echo.Context) error {
	symbol := c.Param("symbol")
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "symbol is required"))
	}

	days := 30
	if d := c.QueryParam("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	history, err := h.usecase.GetHistory(c.Request().Context(), symbol, days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "internal server error"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(history, nil))
}
