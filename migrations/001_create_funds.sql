CREATE TABLE IF NOT EXISTS funds (
    id SERIAL PRIMARY KEY,
    scheme_code BIGINT UNIQUE NOT NULL,
    scheme_name TEXT NOT NULL,
    fund_house TEXT,
    scheme_type TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);