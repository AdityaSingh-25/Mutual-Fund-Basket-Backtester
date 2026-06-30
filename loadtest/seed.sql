-- Load-test seed data for the MF Basket Backtester.
--
-- Creates three synthetic funds with ~3 years of daily NAV history and a
-- basket referencing all three. This lets POST /backtest run entirely from
-- local data — no external api.mfapi.in calls in the request path — so the
-- load test measures the service, not the upstream NAV provider.
--
-- Idempotent: safe to re-run. Apply with:
--   docker exec -i mf_postgres psql -U user -d mf_backtester < loadtest/seed.sql

BEGIN;

-- Funds (scheme_code is unique).
INSERT INTO funds (scheme_code, scheme_name, fund_house, scheme_type)
VALUES
    (900001, 'Loadtest Fund A', 'Loadtest AMC', 'Equity'),
    (900002, 'Loadtest Fund B', 'Loadtest AMC', 'Equity'),
    (900003, 'Loadtest Fund C', 'Loadtest AMC', 'Debt')
ON CONFLICT (scheme_code) DO NOTHING;

-- Daily NAV history, 2019-01-01 .. 2022-01-01, deterministic per fund: a slow
-- upward drift plus a per-fund sine oscillation so drawdown/XIRR are non-zero.
INSERT INTO nav (fund_id, nav, date)
SELECT f.id,
       GREATEST(1,
           100
           + 0.03 * (d.day - DATE '2019-01-01')
           + (5 * f.k) * sin((d.day - DATE '2019-01-01') / 25.0)
       )::numeric,
       d.day
FROM (
    SELECT id, row_number() OVER (ORDER BY id) AS k
    FROM funds
    WHERE scheme_code IN (900001, 900002, 900003)
) f
CROSS JOIN (
    SELECT generate_series(DATE '2019-01-01', DATE '2022-01-01', INTERVAL '1 day')::date AS day
) d
ON CONFLICT (fund_id, date) DO NOTHING;

-- Rebuild the load-test basket from scratch so weights stay correct on re-run.
DELETE FROM basket_items
WHERE basket_id IN (SELECT id FROM baskets WHERE name = 'Loadtest Basket');
DELETE FROM baskets WHERE name = 'Loadtest Basket';

WITH b AS (
    INSERT INTO baskets (name) VALUES ('Loadtest Basket') RETURNING id
)
INSERT INTO basket_items (basket_id, fund_id, weight)
SELECT b.id, f.id, w.weight
FROM b
CROSS JOIN (VALUES (900001, 0.5), (900002, 0.3), (900003, 0.2)) AS w(code, weight)
JOIN funds f ON f.scheme_code = w.code;

COMMIT;

-- The basket id the load test should target.
SELECT id AS loadtest_basket_id
FROM baskets
WHERE name = 'Loadtest Basket'
ORDER BY id DESC
LIMIT 1;
