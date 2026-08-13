package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

// ArticleHandler handles HTTP requests for news articles.
type ArticleHandler struct {
	usecase domain.ArticleUsecase
}

// NewArticleHandler creates a new article HTTP handler.
func NewArticleHandler(uc domain.ArticleUsecase) *ArticleHandler {
	return &ArticleHandler{usecase: uc}
}

// GetByID handles GET /api/v1/articles/:id
func (h *ArticleHandler) GetByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "invalid article ID format"))
	}

	article, err := h.usecase.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return c.JSON(http.StatusNotFound, NewErrorResponse(ErrCodeNotFound, "article not found"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "internal server error"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(article, nil))
}

// List handles GET /api/v1/articles
// Query params: symbol, sentiment, source, from, to, limit, offset
func (h *ArticleHandler) List(c echo.Context) error {
	filter := domain.ArticleFilter{
		Symbol: c.QueryParam("symbol"),
		Source: c.QueryParam("source"),
		Limit:  20,
		Offset: 0,
	}

	// Parse sentiment label
	if sentiment := c.QueryParam("sentiment"); sentiment != "" {
		label := domain.SentimentLabel(sentiment)
		switch label {
		case domain.SentimentBullish, domain.SentimentBearish, domain.SentimentNeutral:
			filter.SentimentLabel = &label
		default:
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "invalid sentiment: must be Bullish, Bearish, or Neutral"))
		}
	}

	// Parse date range
	if from := c.QueryParam("from"); from != "" {
		t, err := time.Parse(time.DateOnly, from)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "invalid 'from' date format, use YYYY-MM-DD"))
		}
		filter.FromDate = &t
	}

	if to := c.QueryParam("to"); to != "" {
		t, err := time.Parse(time.DateOnly, to)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "invalid 'to' date format, use YYYY-MM-DD"))
		}
		filter.ToDate = &t
	}

	// Parse pagination
	if limit := c.QueryParam("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offset := c.QueryParam("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	articles, err := h.usecase.List(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(ErrCodeInternal, "internal server error"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(articles, &Meta{
		Limit:  filter.Limit,
		Offset: filter.Offset,
		Count:  len(articles),
	}))
}
