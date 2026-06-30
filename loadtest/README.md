# Load testing

Measures whether the API meets **1,000 RPS at p95 < 30 ms** on the `/backtest`
endpoint, and provides the harness to re-run it after any change.

## What it tests

The realistic high-throughput path is a **cache hit**: a backtest that has
already been computed and is served from Redis. The harness warms the cache
once, then drives `RPS` requests/second at the same backtest for `DURATION` and
asserts `p(95) < 30ms` (plus <1% errors and >99% cache hits).

Out of scope for 30 ms (by design — these cannot meet it):

- **Cache miss** — runs the simulation and serializes the result. Single-digit
  to low-double-digit ms once data is local, but not the 30 ms target.
- **Cold backfill** — the very first backtest for a fund calls the external
  `api.mfapi.in` (up to 30 s × retries). Pre-seed (this harness does) or
  pre-warm to keep that out of the request path.
- **`GET /funds` search** — `ILIKE '%q%'` is an unindexed sequential scan with
  no caching; it is a separate, lower-QPS concern.

## Prerequisites

- Docker (for Postgres + Redis)
- Go (builds and runs the API natively)
- `curl`
- k6 — optional; if not installed, the script runs it via the `grafana/k6`
  Docker image.

## Run

```bash
./loadtest/run.sh
```

This will:

1. `docker compose up -d postgres redis`
2. build and start the API natively with **rate limiting disabled**
   (`RATE_LIMIT_RPS=0`) and `DB_MAX_OPEN_CONNS=50`
3. apply `seed.sql` (3 funds, ~1,096 daily NAVs each, one basket)
4. warm the cache with one `/backtest` call
5. run k6 at 1,000 RPS for 30 s and print the latency distribution

Stop the dependencies afterwards with:

```bash
docker compose -f docker/docker-compose.yml down
```

### Knobs

Override via environment, e.g.:

```bash
RPS=2000 DURATION=60s ./loadtest/run.sh        # push harder / longer
DB_MAX_OPEN_CONNS=100 ./loadtest/run.sh        # widen the DB pool
```

`RPS`, `DURATION`, `PORT`, `DB_MAX_OPEN_CONNS`, `START_DATE`, `END_DATE`.

## Interpreting results

k6 prints `http_req_duration` with `p(95)`. The run **passes** when all
thresholds are met (k6 exits non-zero otherwise):

```
✓ http_req_duration..............: p(95)=...ms   (must be < 30ms)
✓ http_req_failed................: rate=...%      (must be < 1%)
✓ checks.........................: ...%           (must be > 99%, X-Cache: HIT)
```

If p95 exceeds 30 ms, the usual levers are: widen `DB_MAX_OPEN_CONNS` (only
matters on misses), shrink the date range (smaller JSON payload per response),
or run the API and load generator on separate hosts to remove contention.

## Why rate limiting is disabled

The limiter is **per source IP**. A single load generator is one IP, so the
default 5 req/s/IP would just return 429s. `run.sh` sets `RATE_LIMIT_RPS=0` to
disable it for the test. In production, size the limit for real client
distribution, or move limiting to the load balancer / a shared Redis counter.

## Manual cache-miss check (optional)

To see miss-path latency, send a backtest with a date range you have not
requested before and inspect the `X-Cache` response header (`MISS`) and timing:

```bash
curl -s -o /dev/null -w '%{http_code} %{time_total}s\n' \
  -X POST http://localhost:8080/backtest \
  -H 'Content-Type: application/json' \
  -d '{"basket_id":<ID>,"start_date":"2019-06-01","end_date":"2021-06-01","amount":100000,"mode":"lumpsum"}'
```
