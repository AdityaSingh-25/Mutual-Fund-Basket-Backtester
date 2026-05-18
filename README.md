# Mutual Fund Basket Backtester

A Go backend service for building custom mutual fund portfolios and backtesting them against historical NAV data sourced from AMFI India.

## Features

- **NAV ingestion** — fetches daily NAV data from AMFI India and stores it in Postgres
- **Basket management** — create named portfolios of funds with weighted allocations
- **Backtesting** — computes CAGR, XIRR, and max drawdown over any historical period
- **Caching** — Redis caches backtest results to avoid redundant computation
- **Local summaries** — template-based plain-language summaries of backtest results

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Container | Docker / Docker Compose |

## Project Structure

```
.
├── cmd/
│   ├── server/         # API server entry point
│   └── backfill/       # One-shot NAV history backfill command
├── config/             # Env-based config loader
├── docker/             # Dockerfile and docker-compose.yml
├── internal/
│   ├── api/            # HTTP router, handlers, and middleware
│   ├── backtest/       # Backtest engine (portfolio simulation)
│   ├── cache/          # Redis client
│   ├── compute/        # CAGR, XIRR, drawdown calculations
│   ├── db/             # Postgres client, queries, migration runner
│   ├── ingestion/      # AMFI NAV fetcher and scheduler
│   ├── models/         # Shared data types
│   └── testsupport/    # Integration-test helpers
├── migrations/         # Embedded SQL schema files
└── web/                # React + TypeScript frontend (Vite)
```

## Setup

### Prerequisites

- Go 1.22+ (the module declares Go 1.26 in `go.mod`; an older toolchain will be auto-fetched)
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
REDIS_URL=redis://localhost:6379
PORT=8080
```

### 2. Run everything with Docker Compose

```bash
docker compose -f docker/docker-compose.yml up -d --build
```

This builds and starts three containers: Postgres (`:5432`), Redis (`:6379`),
and the API server (`:8080`). Database migrations are applied automatically
when the server starts.

### Running the server locally instead

To run the Go server outside Docker (for development):

```bash
# start only the infrastructure
docker compose -f docker/docker-compose.yml up -d postgres redis

# run the server — migrations apply automatically on startup
go run ./cmd/server
```

### Backfilling NAV history

The server backfills a fund's full NAV history lazily, the first time it is
backtested. To pre-load history for every fund already used in a basket, run
the one-shot backfill command:

```bash
go run ./cmd/backfill
```

### Frontend

The web frontend is a React + TypeScript app (Vite), in [`web/`](web/):

```bash
cd web
npm install
npm run dev      # dev server on http://localhost:5173
```

It calls the API at `http://localhost:8080` by default; override that with
`VITE_API_BASE_URL` (see `web/.env.example`). The backend enables CORS, so the
dev server can call it directly — just make sure the server is running.

## Database Schema

- **funds** — mutual fund metadata (scheme code, name, fund house)
- **nav** — daily NAV records per fund
- **baskets** — named user-defined portfolios
- **basket_items** — funds within a basket with weight allocations

## API

The server listens on `PORT` (default `8080`). All requests and responses are JSON.
Requests are rate-limited per client IP (burst of 10, then 5 sustained requests/second);
exceeding the limit returns `429 Too Many Requests`.

### `GET /health`

Liveness check.

```json
{ "status": "ok" }
```

### `POST /baskets`

Create a named basket with weighted fund allocations. Weights are relative —
they are normalised at backtest time, so they need not sum to 100.

```json
{
  "name": "My Equity Basket",
  "items": [
    { "fund_id": 1, "weight": 60 },
    { "fund_id": 2, "weight": 40 }
  ]
}
```

Returns `201 Created` with the persisted basket and its items.

### `GET /baskets/{id}`

Fetch a basket and its fund allocations. Returns `404` if the basket does not exist.

### `POST /backtest`

Simulate an investment into a basket over a historical date range. Each
contribution is split across funds by weight and buys units at that day's
NAVs; the portfolio is valued (forward-filled) through the end date.

`mode` selects the investment style:

- `lumpsum` (default) — `amount` is invested once at the start.
- `sip` — `amount` is invested every month (a Systematic Investment Plan).

`rebalance` optionally resets the basket to its target weights on a schedule —
`none` (default), `monthly`, `quarterly`, or `yearly`. Without it, holdings
drift as funds grow at different rates.

`benchmark_fund_id` is optional; when set, the basket is also backtested
against that single fund over the same range, amount, mode and rebalance.

```json
{
  "basket_id": 1,
  "start_date": "2020-01-01",
  "end_date": "2024-01-01",
  "amount": 100000,
  "mode": "lumpsum",
  "rebalance": "yearly",
  "benchmark_fund_id": 119551
}
```

Response:

```json
{
  "mode": "lumpsum",
  "rebalance": "yearly",
  "cagr": 14.2,
  "xirr": 13.8,
  "drawdown": 22.1,
  "total_invested": 100000.0,
  "final_value": 152340.0,
  "series": [
    { "date": "2020-01-01", "value": 100000.0 },
    { "date": "2020-01-02", "value": 100450.0 }
  ],
  "benchmark": {
    "mode": "lumpsum",
    "rebalance": "yearly",
    "cagr": 12.1,
    "xirr": 12.1,
    "drawdown": 25.4,
    "total_invested": 100000.0,
    "final_value": 140900.0,
    "series": [ ... ]
  }
}
```

`series` is the daily portfolio value, suitable for charting. For a SIP,
`total_invested` is the sum of all monthly contributions and `xirr` is the
more meaningful return measure (CAGR treats all money as invested upfront).
`benchmark` is present only when `benchmark_fund_id` was supplied, and carries
the same fields for the comparison fund.

Results are cached in Redis; the `X-Cache` response header reports `HIT`,
`MISS`, or `SKIP`.

### `POST /summary`

Runs a backtest (same request body as `/backtest`) and returns the result
alongside a plain-language summary generated locally from the backtest numbers.

```json
{
  "basket": "My Equity Basket",
  "result": { "mode": "lumpsum", "cagr": 14.2, "xirr": 13.8, "drawdown": 22.1,
              "total_invested": 100000.0, "final_value": 152340.0 },
  "summary": "This basket ended at ₹152340.00 from ₹100000.00 invested, with an XIRR of 13.80%. Its worst drawdown was 22.10%, meaning the portfolio fell that much from a prior high during the period."
}
```

## Running Tests

### Unit tests

```bash
go test ./...
```

The `compute` (CAGR, XIRR, drawdown), `backtest` engine, and rate-limiting
middleware packages have pure unit tests that run without any infrastructure.

### Integration tests

The DB query layer, HTTP handlers, the `backtest.Run` wrapper, and the Redis
cache have integration tests. They run only when pointed at a database and
Redis via environment variables, and are **skipped** otherwise:

```bash
TEST_DB_URL=postgres://user:password@localhost:5432/mf_backtester?sslmode=disable \
TEST_REDIS_URL=redis://localhost:6379 \
go test -race ./...
```

Integration tests create their own fixtures with unique identifiers and clean
up after themselves, so they are safe to run against a populated database.

### Continuous integration

`.github/workflows/ci.yml` runs `gofmt`, `go build`, `go vet`, and
`go test -race ./...` on every push and pull request, with Postgres and Redis
provided as service containers.
