# Mutual Fund Basket Backtester

A Go backend service for building custom mutual fund portfolios and backtesting them against historical NAV data sourced from AMFI India.

## Features

- **NAV ingestion** — fetches daily NAV data from AMFI India and stores it in Postgres
- **Basket management** — create named portfolios of funds with weighted allocations
- **Backtesting** — computes CAGR, XIRR, and max drawdown over any historical period
- **Caching** — Redis caches backtest results to avoid redundant computation
- **AI summaries** — Claude API generates plain-language summaries of backtest results

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Container | Docker / Docker Compose |
| AI | Claude API |

## Project Structure

```
.
├── cmd/server/         # Entry point
├── config/             # Env-based config loader
├── docker/             # Dockerfile and docker-compose.yml
├── internal/
│   ├── api/            # HTTP router and handlers
│   ├── cache/          # Redis client
│   ├── claude/         # AI summary integration
│   ├── compute/        # CAGR, XIRR, drawdown calculations
│   ├── db/             # Postgres client and queries
│   ├── ingestion/      # AMFI NAV fetcher and scheduler
│   └── models/         # Shared data types
└── migrations/         # SQL schema files
```

## Setup

### Prerequisites

- Go 1.21+
- Docker and Docker Compose

### 1. Clone and configure

```bash
git clone https://github.com/AdityaSingh-25/Mutual-Fund-Basket-Backtester
cd Mutual-Fund-Basket-Backtester
cp .env.example .env   # then fill in values
```

**.env variables:**

```
DB_URL=postgres://user:password@localhost:5432/mf_backtester?sslmode=disable
REDIS_URL=localhost:6379
CLAUDE_API_KEY=your_api_key_here
PORT=8080
```

### 2. Start infrastructure

```bash
docker compose -f docker/docker-compose.yml up -d
```

This starts Postgres on `:5432` and Redis on `:6379`.

### 3. Apply migrations

```bash
psql $DB_URL -f migrations/001_create_funds.sql
psql $DB_URL -f migrations/002_create_nav.sql
psql $DB_URL -f migrations/003_create_baskets.sql
```

### 4. Run the server

```bash
go run cmd/server/main.go
```

## Database Schema

- **funds** — mutual fund metadata (scheme code, name, fund house)
- **nav** — daily NAV records per fund
- **baskets** — named user-defined portfolios
- **basket_items** — funds within a basket with weight allocations

## Running Tests

```bash
go test ./...
```

The compute package (CAGR, XIRR, max drawdown) has full unit test coverage. API and DB layers require running infrastructure.
