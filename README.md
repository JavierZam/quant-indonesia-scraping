# Quant Market Intelligence Pipeline 🇮🇩 📈

![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)
![Cloud Run](https://img.shields.io/badge/Cloud%20Run-Optimized-4285F4?style=flat&logo=googlecloud)

## 📌 Overview

A high-throughput automated **Market Intelligence & Quant News Pipeline** focused on the Indonesian stock market (IDX/BEI). This system ingests RSS feeds, deduplicates them, scrapes full article content, and analyzes it using AI (Gemini) for sentiment analysis and entity extraction. The processed data is persisted in PostgreSQL and served via a REST API with Valkey caching, designed for quantitative analysis and algorithmic trading signals.

## ✨ Features

- **Concurrent RSS Feed Ingestion**: High-performance worker pools for fetching and parsing RSS feeds.
- **Automated Ingestion Scheduler**: Built-in cron scheduler (`robfig/cron/v3`) for periodic background ingestion.
- **URL Deduplication**: Efficient deduplication using MD5 hashing stored in Valkey.
- **AI-Powered Sentiment Analysis**: Leverages Google Gemini API to analyze market sentiment (Bullish/Bearish/Neutral) with confidence scores (-1.0 to +1.0).
- **Entity Extraction**: Automatically identifies and extracts companies, executives, sectors, and IDX tickers mentioned in articles.
- **Quant Trading Signals & Analytics**: Calculates aggregated sentiment metrics and generates trading recommendations (BUY/SELL/HOLD).
- **Time-Series Chart Data**: Serves daily average sentiment scores for historical trend visualization.
- **RESTful API**: Fast and scalable API with pagination, filtering, rate-limiting, and caching built on Echo v4.
- **Observability**: Prometheus metrics export (`/metrics`) & deep health readiness probes (`/healthz`, `/readyz`).
- **Resilience**: Implements exponential backoff retry and token-bucket rate limiting for external API calls.
- **Graceful Shutdown**: Ensures all in-flight processes and database connections are closed safely.
- **Cloud-Native**: Stateless design, distroless Docker container, ready for deployment on GCP Cloud Run.

## 🏗 Architecture

```mermaid
flowchart LR
    A[RSS Feeds] -->|Worker Pool| B(Ingestion Pipeline)
    B -->|Check MD5| C{Valkey Dedup}
    C -->|New| D[Scraper]
    C -->|Exists| X((Skip))
    D --> E[Gemini LLM]
    E -->|Sentiment & Entities| F[(PostgreSQL)]
    F <--> G[Echo REST API]
    G <-->|Cache| H[(Valkey)]
    G --> I[Client/Quant Model]
```

## 🛠 Tech Stack

| Component | Technology |
|---|---|
| **Language** | Go 1.25 |
| **API Framework** | Echo v4 |
| **Database** | PostgreSQL 16 (pgx v5) |
| **Cache/Broker** | Valkey 8 |
| **AI/LLM** | Google Gemini API |
| **Metrics** | Prometheus |
| **Scheduler** | robfig/cron v3 |
| **Container** | Docker (distroless) |
| **Deployment** | GCP Cloud Run |
| **Architecture** | Clean Architecture |

## 📋 Prerequisites

- Go 1.23+
- Docker & Docker Compose
- Gemini API Key (from [Google AI Studio](https://aistudio.google.com/))
- PostgreSQL 16 (via Docker or managed service)
- Valkey 8 (via Docker or managed service)

## 🚀 Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/javier-garcia/quant-indonesia-scraping.git
   cd quant-indonesia-scraping
   ```

2. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

3. **Set your Gemini API key in `.env`:**
   ```env
   LLM_API_KEY=your_gemini_api_key_here
   ```

4. **Start the database and cache using Docker Compose:**
   ```bash
   make infra-up
   ```

5. **Run Database Migrations & Seed Data:**
   ```bash
   make migrate-up
   make seed
   ```

6. **Run the server:**
   ```bash
   make run
   ```

7. **Test Health & Metrics:**
   ```bash
   curl http://localhost:8080/healthz
   curl http://localhost:8080/readyz
   curl http://localhost:8080/metrics
   ```

## ⚙️ Configuration

Environment variables used to configure the application:

| Variable | Description | Default |
|---|---|---|
| `SERVER_PORT` | API Server port | `8080` |
| `DB_HOST` | PostgreSQL Host | `localhost` |
| `DB_PORT` | PostgreSQL Port | `5432` |
| `DB_USER` | PostgreSQL User | `quantuser` |
| `DB_PASSWORD` | PostgreSQL Password | `quantpass` |
| `DB_NAME` | PostgreSQL Database Name | `quantintel` |
| `VALKEY_ADDR` | Valkey Address | `localhost:6379` |
| `LLM_PROVIDER` | AI Provider (`gemini`) | `gemini` |
| `LLM_API_KEY`| Google Gemini API Key | *(Required)* |
| `LLM_MODEL` | Gemini model name | `gemini-2.0-flash` |
| `INGESTION_WORKERS` | Concurrent ingestion workers | `10` |
| `INGESTION_CRON_SCHEDULE` | Cron schedule for background ingestion | `*/30 * * * *` |
| `RATE_LIMIT_RPS` | Max requests per second per IP | `10` |
| `RATE_LIMIT_BURST` | Max request burst per IP | `20` |

## 📚 API Documentation

### 1. Health Checks & Metrics
- `GET /healthz` — Liveness probe (`200 OK`)
- `GET /readyz` — Readiness probe (Pings Postgres & Valkey, returns latency)
- `GET /metrics` — Prometheus metrics exporter

### 2. Trading Signals & Sentiment Analytics
`GET /api/v1/signals`
**Query Parameters:** `symbol`, `sector`, `period` (`24h`, `7d`, `30d`)
**Response:**
```json
{
  "success": true,
  "data": [
    {
      "symbol": "BBCA",
      "company_name": "Bank Central Asia",
      "sector": "Financial Services",
      "signal": "BUY",
      "average_score": 0.65,
      "article_count": 12,
      "bullish_articles": 9,
      "bearish_articles": 1,
      "neutral_articles": 2,
      "period": "7d",
      "generated_at": "2026-08-11T09:30:00Z"
    }
  ]
}
```

### 3. Historical Sentiment Trend
`GET /api/v1/signals/:symbol/history`
**Query Parameters:** `days` (default: 30)
**Response:**
```json
{
  "success": true,
  "data": [
    {
      "date": "2026-08-01",
      "average_score": 0.45,
      "article_count": 3
    },
    {
      "date": "2026-08-02",
      "average_score": 0.72,
      "article_count": 5
    }
  ]
}
```

### 4. Trigger Ingestion Pipeline
`POST /api/v1/ingestion/trigger`
**Request Body:**
```json
{
  "feeds": [
    {"name": "CNBC Indonesia", "url": "https://www.cnbcindonesia.com/market/rss"}
  ]
}
```

### 5. List Articles
`GET /api/v1/articles`
**Query Parameters:** `symbol`, `sentiment`, `source`, `from`, `to`, `limit`, `offset`

### 6. Get Article by ID
`GET /api/v1/articles/:id`

### 7. List Stocks
`GET /api/v1/stocks`
**Query Parameters:** `sector`, `limit`, `offset`

### 8. Get Stock by Symbol
`GET /api/v1/stocks/:symbol`

## 🗄 Database Schema

```mermaid
erDiagram
    ARTICLES {
        uuid id PK
        string title
        string content
        string url
        string source
        string sentiment
        float sentiment_score
        timestamp published_at
    }
    STOCKS {
        string symbol PK
        string name
        string sector
    }
    EXECUTIVES {
        uuid id PK
        string name
        string role
        string company
    }
    ARTICLE_ENTITIES {
        uuid article_id FK
        string entity_type
        string entity_id
    }
    ARTICLES ||--o{ ARTICLE_ENTITIES : "has"
    STOCKS ||--o{ ARTICLE_ENTITIES : "referenced as"
    EXECUTIVES ||--o{ ARTICLE_ENTITIES : "referenced as"
```

## 📁 Project Structure

```text
quant-indonesia-scraping/
├── cmd/server/          # Application entrypoint (main.go)
├── config/              # Configuration loading and env parsing
├── domain/              # Core business logic, interfaces, and entities
├── delivery/http/       # HTTP transport layer (Echo handlers, router, middleware)
├── ingestion/           # Pipeline logic: Feed fetching, workers, scrapers
├── llm/                 # AI Integration: Gemini API client and prompts
├── pkg/                 # Shared utilities (hasher, httpclient, logger)
├── repository/          # Data persistence layer
│   ├── postgres/        # PostgreSQL implementations
│   └── valkey/          # Valkey (Redis-compatible) cache/dedup implementations
├── usecase/             # Application specific business rules
├── migrations/          # SQL database migrations
├── docker-compose.yml   # Local infrastructure
├── Dockerfile           # Multi-stage Distroless Dockerfile
└── ...
```

## 💻 Development

- **Run Tests:**
  ```bash
  go test ./... -v
  ```
- **Build Binary:**
  ```bash
  go build -o server ./cmd/server
  ```
- **Linting:**
  It is recommended to use `golangci-lint`:
  ```bash
  golangci-lint run
  ```

## 🐳 Docker

- **Build Image:**
  ```bash
  docker build -t quant-intel .
  ```
- **Run Container:**
  ```bash
  docker run -p 8080:8080 --env-file .env quant-intel
  ```

## ☁️ Deployment to Cloud Run

The application is designed to be stateless and is optimized for GCP Cloud Run.

1. **Build & Push to Artifact Registry:**
   ```bash
   gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/quant-intel
   ```
2. **Deploy to Cloud Run:**
   Deploy the image, mapping environment variables securely using Secret Manager.
3. **Database & Cache:**
   - Provision a Cloud SQL instance for PostgreSQL 16.
   - Provision a Memorystore instance for Redis/Valkey.
   - Ensure Cloud Run has the appropriate VPC connectors to access these resources.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
