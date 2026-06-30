// k6 load test for POST /backtest.
//
// Targets the cache-hit hot path: the cache is warmed once before the run, so
// every request here is a Redis read served as raw JSON bytes. This is the
// scenario the 1000 RPS @ p95 30 ms goal is about. Cache misses (cold backtest
// compute, and especially the external NAV backfill) are out of scope and will
// not meet 30 ms — see loadtest/README.md.
//
// Tunable via env: BASE_URL, BASKET_ID, RPS, DURATION, START_DATE, END_DATE,
// VUS, MAX_VUS.

import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const RPS = Number(__ENV.RPS || 1000);

export const options = {
  scenarios: {
    backtest_cache_hit: {
      executor: 'constant-arrival-rate',
      rate: RPS,
      timeUnit: '1s',
      duration: __ENV.DURATION || '30s',
      preAllocatedVUs: Number(__ENV.VUS || 200),
      maxVUs: Number(__ENV.MAX_VUS || 1000),
    },
  },
  thresholds: {
    // The headline requirement: 95th-percentile latency under 30 ms.
    http_req_duration: ['p(95)<30'],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

const payload = JSON.stringify({
  basket_id: Number(__ENV.BASKET_ID || 1),
  start_date: __ENV.START_DATE || '2019-01-01',
  end_date: __ENV.END_DATE || '2022-01-01',
  amount: 100000,
  mode: 'lumpsum',
});

const params = { headers: { 'Content-Type': 'application/json' } };

export default function () {
  const res = http.post(`${BASE}/backtest`, payload, params);
  check(res, {
    'status is 200': (r) => r.status === 200,
    'served from cache': (r) => r.headers['X-Cache'] === 'HIT',
  });
}
