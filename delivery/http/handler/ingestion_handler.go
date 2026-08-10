package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/javier-garcia/quant-indonesia-scraping/domain"
	"github.com/javier-garcia/quant-indonesia-scraping/ingestion"
)

// IngestionHandler handles HTTP requests to trigger the ingestion pipeline.
type IngestionHandler struct {
	pipeline       *ingestion.Pipeline
	articleUsecase domain.ArticleUsecase
	logger         *slog.Logger
}

// NewIngestionHandler creates a new ingestion HTTP handler.
func NewIngestionHandler(
	pipeline *ingestion.Pipeline,
	articleUsecase domain.ArticleUsecase,
	logger *slog.Logger,
) *IngestionHandler {
	return &IngestionHandler{
		pipeline:       pipeline,
		articleUsecase: articleUsecase,
		logger:         logger,
	}
}

// TriggerRequest is the request body for triggering ingestion.
type TriggerRequest struct {
	Feeds []FeedInput `json:"feeds" validate:"required,min=1"`
}

// FeedInput represents a single feed source in the trigger request.
type FeedInput struct {
	Name string `json:"name" validate:"required"`
	URL  string `json:"url" validate:"required,url"`
}

// TriggerResponse is the response after triggering ingestion.
type TriggerResponse struct {
	ArticlesIngested int      `json:"articles_ingested"`
	Errors           int      `json:"errors"`
	ErrorMessages    []string `json:"error_messages,omitempty"`
}

// Trigger handles POST /api/v1/ingestion/trigger
// Accepts a list of RSS feed sources and runs the ingestion pipeline.
func (h *IngestionHandler) Trigger(c echo.Context) error {
	var req TriggerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
	}

	if len(req.Feeds) == 0 {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "at least one feed source is required",
		})
	}

	// Convert to ingestion feed sources
	sources := make([]ingestion.FeedSource, len(req.Feeds))
	for i, f := range req.Feeds {
		sources[i] = ingestion.FeedSource{
			Name: f.Name,
			URL:  f.URL,
		}
	}

	h.logger.Info("ingestion triggered via API", "feeds", len(sources))

	// Run the pipeline
	ctx := c.Request().Context()
	articles, errs := h.pipeline.Run(ctx, sources)

	// Process ingested articles through the usecase (LLM analysis + persistence)
	var processErrors []string
	for _, article := range articles {
		if err := h.articleUsecase.Ingest(ctx, article); err != nil {
			h.logger.Error("failed to process article",
				"url", article.URL,
				"error", err,
			)
			processErrors = append(processErrors, err.Error())
		}
	}

	// Collect error messages from pipeline
	for _, e := range errs {
		if e != nil {
			processErrors = append(processErrors, e.Error())
		}
	}

	resp := TriggerResponse{
		ArticlesIngested: len(articles),
		Errors:           len(processErrors),
		ErrorMessages:    processErrors,
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    resp,
	})
}
