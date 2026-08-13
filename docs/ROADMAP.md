# Quant Market Intelligence Pipeline - Roadmap

This document lists EVERYTHING that needs to be done — both technical and non-technical — to take this project from its current state to a fully production-ready system.

The project is a Quant Market Intelligence Pipeline for the Indonesian stock market (IDX/BEI). It ingests RSS feeds, scrapes articles, analyzes with Gemini AI, and serves via REST API.

## 🔑 Priority 1: External Services Setup (Before First Run)
Things the developer needs to set up before the system can actually work:

### Gemini API Key
- Go to https://aistudio.google.com/app/apikey
- Create a new API key
- Set `LLM_API_KEY` in `.env`
- Free tier: 15 RPM, 1M tokens/day (sufficient for development)
- Consider Gemini 2.0 Flash for cost efficiency

### Indonesian Financial News RSS Feed Sources
List real, working RSS feeds for Indonesian financial news:
- https://www.cnbcindonesia.com/market/rss — CNBC Indonesia Market
- https://www.bisnis.com/rss — Bisnis Indonesia
- https://finance.detik.com/rss — Detik Finance
- https://www.kontan.co.id/rss — Kontan
- https://www.idxchannel.com/rss — IDX Channel
- https://investor.id/rss — Investor Daily
- https://www.bareksa.com/rss — Bareksa
- Note: Some feeds may require verification, the developer should test each one

### Docker Setup
- Install Docker Desktop
- Run `docker compose up -d` for Postgres + Valkey
- Verify with `docker compose ps`

## 🔧 Priority 2: Critical Technical Tasks

### Database Migration Tooling
- [ ] Install golang-migrate CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- [ ] Add Makefile with migration commands
- [ ] Consider adding `migrate` command to the server binary itself

### Makefile
- [x] Create a Makefile with common commands:
  - `make run` — Run server
  - `make build` — Build binary
  - `make test` — Run tests
  - `make migrate-up` — Run migrations
  - `make migrate-down` — Rollback
  - `make docker-build` — Build Docker image
  - `make lint` — Run linter

### Input Validation
- [x] Add request body validation & query param validation (sentiment label, dates, limits)
- [x] Validate URL formats in ingestion trigger
- [x] Sanitize user inputs in query parameters

### Error Handling Improvements
- [x] Add structured error codes (`APIError` envelope: `VALIDATION_ERROR`, `NOT_FOUND`, `INTERNAL_ERROR`, `BAD_REQUEST`, `RATE_LIMITED`)
- [x] Implement proper error wrapping throughout
- [x] Add request context timeout for long-running ingestion

### Comprehensive Testing
- [x] Unit tests for URL hasher (`pkg/hasher`)
- [ ] Unit tests for all handlers (mock usecases)
- [ ] Unit tests for usecases (mock repos)
- [ ] Integration tests with testcontainers (Postgres + Valkey)
- [ ] Test the LLM prompt with sample articles
- [ ] Load testing with k6 or vegeta
- [ ] Aim for >80% code coverage

## 🔐 Priority 3: Security & Production Hardening

### Authentication & Authorization
- [ ] Add API key authentication middleware
- [ ] Protect ingestion endpoint (admin only)
- [ ] Rate limit per API key
- [ ] Consider JWT for future multi-tenant support

### Secret Management
- [ ] Integrate GCP Secret Manager SDK for production
- [ ] Never commit .env files
- [ ] Rotate API keys periodically

### Rate Limiting on API
- [x] Add per-IP rate limiting middleware (token bucket with `golang.org/x/time/rate`)
- [x] Add per-endpoint rate limiting
- [x] Return proper 429 responses with `Retry-After` header & auto-cleanup stale IPs

### HTTPS & CORS
- [x] Configure CORS middleware in Echo
- [ ] Ensure HTTPS-only in production (Cloud Run handles TLS)

## ⏰ Priority 4: Automation & Scheduling

### Scheduled Ingestion
- [x] Add in-process cron scheduler (`robfig/cron/v3`) for auto-ingestion (configurable schedule, default every 30m)
- [x] Support manual trigger via `POST /api/v1/ingestion/trigger`
- [ ] Add a `/api/v1/feeds` endpoint to manage feed sources dynamically
- [ ] Store feed sources in database instead of hardcoding

## 📊 Priority 5: Enhanced Features

### Advanced API Features
- [x] Aggregate sentiment & buy/sell/hold signals endpoint: `GET /api/v1/signals?symbol=BBCA&period=7d`
- [x] Sentiment history chart data endpoint: `GET /api/v1/signals/:symbol/history?days=30`
- [ ] Trending stocks endpoint: `GET /api/v1/trending`
- [ ] Search endpoint with full-text search
- [ ] WebSocket for real-time updates

### Data Analytics
- [x] Sentiment trend tracking over time (daily aggregated average scores)
- [x] Sector-level sentiment aggregation & filtering
- [x] Rule-based trading signal generation (BUY, SELL, HOLD)

## 🚀 Priority 6: Deployment & DevOps

### Monitoring & Observability
- [x] Export metrics in Prometheus format (`GET /metrics`)
- [x] HTTP request latency histograms & request counters
- [x] Structured JSON logging (`log/slog`) with context
- [x] Deep health probes (`GET /healthz` liveness, `GET /readyz` readiness checking DB & Valkey latencies)
- [ ] Set up Grafana dashboards & alerts

## 📝 Priority 7: Documentation
- [ ] API documentation with Swagger/OpenAPI spec
- [ ] Generate `docs/swagger.yaml` using swaggo
- [ ] Add Swagger UI endpoint
- [ ] Architecture Decision Records (ADRs)
- [ ] Contributing guide (CONTRIBUTING.md)
- [ ] Changelog (CHANGELOG.md)

## 💰 Cost Estimates (GCP Production)
Provide rough monthly cost estimates:
| Service | Spec | Est. Cost/month |
|---|---|---|
| Cloud Run | 1 vCPU, 512MB, always-on | ~$15-30 |
| Cloud SQL PostgreSQL | db-f1-micro, 10GB | ~$10-15 |
| Memorystore (Valkey) | M1, 1GB | ~$35-50 |
| Gemini API | Free tier (15 RPM) | $0 |
| Cloud Scheduler | 3 jobs | ~$0.10 |
| **Total** | | **~$60-100/month** |

Note: Can start with free tier for development.
