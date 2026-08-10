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
- [ ] Create a Makefile with common commands:
  - `make run` — Run server
  - `make build` — Build binary
  - `make test` — Run tests
  - `make migrate-up` — Run migrations
  - `make migrate-down` — Rollback
  - `make docker-build` — Build Docker image
  - `make lint` — Run linter

### Input Validation
- [ ] Add request body validation (e.g., go-playground/validator)
- [ ] Validate URL formats in ingestion trigger
- [ ] Sanitize user inputs in query parameters

### Error Handling Improvements
- [ ] Add structured error codes to API responses
- [ ] Implement proper error wrapping throughout
- [ ] Add request context timeout for long-running ingestion

### Comprehensive Testing
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
- [ ] Add per-IP rate limiting middleware (Echo has built-in)
- [ ] Add per-endpoint rate limiting
- [ ] Return proper 429 responses with Retry-After header

### HTTPS & CORS
- [ ] Configure CORS properly for production (not wildcard)
- [ ] Ensure HTTPS-only in production (Cloud Run handles TLS)

## ⏰ Priority 4: Automation & Scheduling

### Scheduled Ingestion
- [ ] Add Cloud Scheduler job to trigger `POST /ingestion/trigger` on schedule
- [ ] Recommended: Every 30 minutes during market hours (09:00-16:00 WIB)
- [ ] Add a `/api/v1/feeds` endpoint to manage feed sources dynamically
- [ ] Store feed sources in database instead of hardcoding

### Feed Source Management
- [ ] Create `feed_sources` table in PostgreSQL
- [ ] Add CRUD API for managing feed sources
- [ ] Support enable/disable individual feeds
- [ ] Track last successful fetch per feed

## 📊 Priority 5: Enhanced Features

### Additional Data Sources
- [ ] IDX (Indonesia Stock Exchange) official API
- [ ] Yahoo Finance Indonesia API
- [ ] Google Finance data
- [ ] Bank Indonesia press releases
- [ ] OJK (Financial Services Authority) announcements
- [ ] Social media sentiment (Twitter/X Indonesia finance accounts)

### Enhanced LLM Analysis
- [ ] Add support for OpenAI GPT-4o as fallback provider
- [ ] Implement LLM response caching (same article = same analysis)
- [ ] Add confidence score to sentiment analysis
- [ ] Extract key financial metrics (revenue, profit, growth %)
- [ ] Detect market-moving events (IPO, M&A, earnings)
- [ ] Multi-language support (Bahasa Indonesia + English)

### Advanced API Features
- [ ] Aggregate sentiment endpoint: `GET /api/v1/sentiment/aggregate?symbol=BBCA&period=7d`
- [ ] Trending stocks endpoint: `GET /api/v1/trending`
- [ ] Sentiment history chart data: `GET /api/v1/sentiment/history?symbol=BBCA`
- [ ] Search endpoint with full-text search
- [ ] WebSocket for real-time updates
- [ ] GraphQL alternative endpoint

### Data Analytics
- [ ] Sentiment trend tracking over time
- [ ] Sector-level sentiment aggregation
- [ ] Correlation between sentiment and stock price movement
- [ ] Alert system when sentiment shifts dramatically

## 🚀 Priority 6: Deployment & DevOps

### CI/CD Pipeline (GitHub Actions)
- [ ] Create `.github/workflows/ci.yml`:
  - Run tests on every PR
  - Lint with golangci-lint
  - Build Docker image
  - Push to Artifact Registry on merge to main
  - Deploy to Cloud Run

### GCP Infrastructure
- [ ] Set up GCP project
- [ ] Create Cloud SQL PostgreSQL instance
- [ ] Create Memorystore for Valkey/Redis
- [ ] Configure Cloud Run service
- [ ] Set up Secret Manager for API keys
- [ ] Configure Cloud Scheduler for automated ingestion
- [ ] Set up Cloud Monitoring & Alerting
- [ ] Estimate monthly costs

### Monitoring & Observability
- [ ] Add OpenTelemetry tracing
- [ ] Export metrics (Prometheus format)
- [ ] Set up structured logging with correlation IDs
- [ ] Create dashboards in Cloud Monitoring or Grafana
- [ ] Set up alerts for: error rate > 5%, latency > 2s, ingestion failures

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
