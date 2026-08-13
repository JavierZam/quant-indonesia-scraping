package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
	"github.com/JavierZam/quant-indonesia-scraping/ingestion"
	"github.com/JavierZam/quant-indonesia-scraping/pkg/broadcaster"
)

// IngestionHandler handles HTTP requests to trigger the ingestion pipeline.
type IngestionHandler struct {
	pipeline       *ingestion.Pipeline
	articleUsecase domain.ArticleUsecase
	broadcaster    *broadcaster.Broadcaster
	logger         *slog.Logger

	mu            sync.Mutex
	currentCancel context.CancelFunc
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

// SetBroadcaster sets the SSE broadcaster.
func (h *IngestionHandler) SetBroadcaster(b *broadcaster.Broadcaster) {
	h.broadcaster = b
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
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeBadRequest, "invalid request body: "+err.Error()))
	}

	if len(req.Feeds) == 0 {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(ErrCodeValidation, "at least one feed source is required"))
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

	h.mu.Lock()
	if h.currentCancel != nil {
		h.mu.Unlock()
		return c.JSON(http.StatusConflict, NewErrorResponse(ErrCodeConflict, "Proses ingestion/analisis AI sedang berjalan di server. Harap tunggu hingga selesai."))
	}

	// Create cancellable context for background execution
	procCtx, cancel := context.WithCancel(context.Background())
	h.currentCancel = cancel
	h.mu.Unlock()

	go func() {
		defer func() {
			h.mu.Lock()
			h.currentCancel = nil
			h.mu.Unlock()
		}()

		if h.broadcaster != nil {
			h.broadcaster.Broadcast(broadcaster.IngestionEvent{
				Type:    "start",
				Stage:   "SCRAPING",
				Total:   len(sources),
				Message: fmt.Sprintf("Mulai Scraping RSS dari %d sumber berita...", len(sources)),
			})
		}

		// Run the pipeline
		articles, errs := h.pipeline.Run(procCtx, sources)

		// Process ingested articles through the usecase (LLM analysis + persistence)
		var processErrors []string
		for i, article := range articles {
			select {
			case <-procCtx.Done():
				h.logger.Warn("ingestion processing cancelled")
				if h.broadcaster != nil {
					h.broadcaster.Broadcast(broadcaster.IngestionEvent{
						Type:    "cancelled",
						Stage:   "SCRAPING",
						Message: "🛑 Ingestion dihentikan oleh pengguna.",
					})
				}
				return
			default:
			}

			if h.broadcaster != nil {
				h.broadcaster.Broadcast(broadcaster.IngestionEvent{
					Type:    "progress",
					Stage:   "AI_ANALYSIS",
					Current: i + 1,
					Total:   len(articles),
					Message: fmt.Sprintf("Analisis AI [%d/%d]: \"%s\"", i+1, len(articles), article.Title),
				})
			}

			if err := h.articleUsecase.Ingest(procCtx, article); err != nil {
				if errors.Is(err, domain.ErrAlreadyExists) || strings.Contains(err.Error(), "already exists") {
					h.logger.Debug("skipping duplicate article already in database", "url", article.URL)
				} else {
					h.logger.Error("failed to process article",
						"url", article.URL,
						"error", err,
					)
					processErrors = append(processErrors, err.Error())
				}
			}
			
			// Add 2-second delay to respect Gemini rate limits
			time.Sleep(2 * time.Second)
		}

		// Collect error messages from pipeline
		for _, e := range errs {
			if e != nil {
				processErrors = append(processErrors, e.Error())
			}
		}

		if h.broadcaster != nil {
			h.broadcaster.Broadcast(broadcaster.IngestionEvent{
				Type:    "done",
				Stage:   "SCRAPING",
				Total:   len(articles),
				Message: fmt.Sprintf("Proses Ingestion Selesai! %d artikel baru berhasil diserap & dianalisis.", len(articles)),
			})
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message": "Ingestion started in background",
		"feeds":   len(sources),
	})
}

// Reprocess handles POST /api/v1/ingestion/reprocess
// Re-analyzes articles that were saved without LLM analysis.
func (h *IngestionHandler) Reprocess(c echo.Context) error {
	h.logger.Info("reprocessing unanalyzed articles via API")

	h.mu.Lock()
	if h.currentCancel != nil {
		h.mu.Unlock()
		return c.JSON(http.StatusConflict, NewErrorResponse(ErrCodeConflict, "Proses ingestion/analisis AI sedang berjalan di server. Harap tunggu hingga selesai."))
	}

	// Create cancellable context for background execution
	procCtx, cancel := context.WithCancel(context.Background())
	h.currentCancel = cancel
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.currentCancel = nil
		h.mu.Unlock()
	}()

	processed, errs := h.articleUsecase.ReprocessUnanalyzed(procCtx)

	var errMsgs []string
	for _, e := range errs {
		if e != nil {
			errMsgs = append(errMsgs, e.Error())
		}
	}

	resp := TriggerResponse{
		ArticlesIngested: processed,
		Errors:           len(errMsgs),
		ErrorMessages:    errMsgs,
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(resp, nil))
}

// Cancel handles POST /api/v1/ingestion/cancel
// Aborts any ongoing ingestion or reprocess task.
func (h *IngestionHandler) Cancel(c echo.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.currentCancel != nil {
		h.currentCancel()
		h.currentCancel = nil
		h.logger.Info("ingestion process cancelled by user request")
		if h.broadcaster != nil {
			h.broadcaster.Broadcast(broadcaster.IngestionEvent{
				Type:    "done",
				Stage:   "CANCEL",
				Message: "🛑 Proses dihentikan oleh pengguna.",
			})
		}
		return c.JSON(http.StatusOK, NewSuccessResponse(map[string]string{"message": "Proses berhasil dibatalkan"}, nil))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(map[string]string{"message": "Tidak ada proses yang sedang berjalan"}, nil))
}
