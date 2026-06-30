#!/usr/bin/env bash
set -euo pipefail

# End-to-end load test for the MF Basket Backtester /backtest endpoint.
#
# Brings up Postgres + Redis via Docker, runs the API natively with rate
# limiting disabled, seeds deterministic data, warms the cache, then drives
# RPS requests/second with k6 and asserts p95 < 30 ms.
#
# Requires: docker, go, curl. k6 is used if installed locally, otherwise it is
# run via the grafana/k6 Docker image.
#
# Knobs (env): RPS (default 1000), DURATION (30s), PORT (8080),
# DB_MAX_OPEN_CONNS (50), START_DATE (2019-01-01), END_DATE (2022-01-01).

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

COMPOSE="docker compose -f docker/docker-compose.yml"
PORT="${PORT:-8080}"
BASE_URL="http://localhost:${PORT}"
RPS="${RPS:-1000}"
DURATION="${DURATION:-30s}"
START_DATE="${START_DATE:-2019-01-01}"
END_DATE="${END_DATE:-2022-01-01}"

APP_PID=""
cleanup() {
  if [[ -n "$APP_PID" ]] && kill -0 "$APP_PID" 2>/dev/null; then
    echo "==> Stopping API (pid $APP_PID)"
    kill "$APP_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "==> Starting Postgres + Redis"
$COMPOSE up -d postgres redis

# Stop the dockerized app if a previous `docker compose up` left it running —
# otherwise it owns the port and the test would hit the wrong (old) binary.
echo "==> Ensuring the dockerized app is not holding port ${PORT}"
$COMPOSE rm -sf app >/dev/null 2>&1 || true

if command -v lsof >/dev/null 2>&1 && lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
  echo "ERROR: port ${PORT} is already in use. Free it (or set PORT=) and retry." >&2
  exit 1
fi

echo "==> Waiting for Postgres"
for _ in $(seq 1 30); do
  docker exec mf_postgres pg_isready -U user -d mf_backtester >/dev/null 2>&1 && break
  sleep 1
done

echo "==> Building API"
go build -o /tmp/mf-loadtest-server ./cmd/server

echo "==> Starting API (RATE_LIMIT_RPS=0, DB_MAX_OPEN_CONNS=${DB_MAX_OPEN_CONNS:-50})"
DB_URL="postgres://user:password@localhost:5432/mf_backtester?sslmode=disable" \
REDIS_URL="redis://localhost:6379" \
PORT="$PORT" \
RATE_LIMIT_RPS=0 \
DB_MAX_OPEN_CONNS="${DB_MAX_OPEN_CONNS:-50}" \
/tmp/mf-loadtest-server &
APP_PID=$!

echo "==> Waiting for /health"
for _ in $(seq 1 30); do
  if ! kill -0 "$APP_PID" 2>/dev/null; then
    echo "ERROR: API process exited during startup — check the output above." >&2
    exit 1
  fi
  curl -fsS "${BASE_URL}/health" >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS "${BASE_URL}/health" >/dev/null 2>&1 || {
  echo "ERROR: API never became healthy on ${BASE_URL}." >&2
  exit 1
}

echo "==> Seeding data"
docker exec -i mf_postgres psql -U user -d mf_backtester < loadtest/seed.sql

BASKET_ID="$(docker exec mf_postgres psql -U user -d mf_backtester -tAc \
  "SELECT id FROM baskets WHERE name = 'Loadtest Basket' ORDER BY id DESC LIMIT 1" | tr -d '[:space:]')"
echo "==> Load-test basket id: ${BASKET_ID}"

echo "==> Warming the cache"
curl -fsS -X POST "${BASE_URL}/backtest" \
  -H 'Content-Type: application/json' \
  -d "{\"basket_id\":${BASKET_ID},\"start_date\":\"${START_DATE}\",\"end_date\":\"${END_DATE}\",\"amount\":100000,\"mode\":\"lumpsum\"}" \
  -o /dev/null -w "warmup: status=%{http_code} time=%{time_total}s\n"

echo "==> Running k6: ${RPS} RPS for ${DURATION}"
if command -v k6 >/dev/null 2>&1; then
  k6 run \
    -e "BASE_URL=${BASE_URL}" -e "BASKET_ID=${BASKET_ID}" -e "RPS=${RPS}" \
    -e "DURATION=${DURATION}" -e "START_DATE=${START_DATE}" -e "END_DATE=${END_DATE}" \
    loadtest/backtest.js
else
  echo "(k6 not found locally — running via Docker image grafana/k6)"
  docker run --rm -i \
    --add-host host.docker.internal:host-gateway \
    -e "BASE_URL=http://host.docker.internal:${PORT}" \
    -e "BASKET_ID=${BASKET_ID}" -e "RPS=${RPS}" -e "DURATION=${DURATION}" \
    -e "START_DATE=${START_DATE}" -e "END_DATE=${END_DATE}" \
    -v "${REPO_ROOT}/loadtest:/scripts:ro" \
    grafana/k6 run /scripts/backtest.js
fi

echo "==> Done. Stop dependencies with: ${COMPOSE} down"
