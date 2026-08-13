package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	valkeylib "github.com/valkey-io/valkey-go"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	pgPool       *pgxpool.Pool
	valkeyClient valkeylib.Client
}

// NewHealthHandler creates a new HealthHandler with the given PostgreSQL pool
// and Valkey client for dependency health checks.
func NewHealthHandler(pgPool *pgxpool.Pool, valkeyClient valkeylib.Client) *HealthHandler {
	return &HealthHandler{
		pgPool:       pgPool,
		valkeyClient: valkeyClient,
	}
}

// checkResult represents the health status of a single dependency.
type checkResult struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
}

// livenessResponse is the JSON body returned by the Liveness endpoint.
type livenessResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// readinessResponse is the JSON body returned by the Readiness endpoint.
type readinessResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Checks    map[string]checkResult `json:"checks"`
}

// Liveness handles GET /healthz — basic liveness check, always returns 200.
func (h *HealthHandler) Liveness(c echo.Context) error {
	return c.JSON(http.StatusOK, livenessResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Readiness handles GET /readyz — deep check, pings DB and Valkey.
// Returns 200 if all healthy, 503 if any dependency is down.
func (h *HealthHandler) Readiness(c echo.Context) error {
	checks := make(map[string]checkResult, 2)
	healthy := true

	// Ping PostgreSQL with a 2-second timeout.
	pgCheck := h.pingPostgres()
	checks["postgres"] = pgCheck
	if pgCheck.Status != "ok" {
		healthy = false
	}

	// Ping Valkey with a 2-second timeout.
	valkeyCheck := h.pingValkey()
	checks["valkey"] = valkeyCheck
	if valkeyCheck.Status != "ok" && valkeyCheck.Status != "disabled" {
		healthy = false
	}

	overallStatus := "ok"
	httpStatus := http.StatusOK
	if !healthy {
		overallStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	return c.JSON(httpStatus, readinessResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	})
}

// pingPostgres pings the PostgreSQL pool and returns the check result.
func (h *HealthHandler) pingPostgres() checkResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := h.pgPool.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return checkResult{Status: "error", LatencyMs: latency}
	}

	return checkResult{Status: "ok", LatencyMs: latency}
}

// pingValkey pings the Valkey client and returns the check result.
func (h *HealthHandler) pingValkey() checkResult {
	if h.valkeyClient == nil {
		return checkResult{Status: "disabled", LatencyMs: 0}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	cmd := h.valkeyClient.B().Ping().Build()
	err := h.valkeyClient.Do(ctx, cmd).Error()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return checkResult{Status: "error", LatencyMs: latency}
	}

	return checkResult{Status: "ok", LatencyMs: latency}
}
