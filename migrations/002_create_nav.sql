CREATE TABLE IF NOT EXISTS nav (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id),
    nav NUMERIC NOT NULL,
    date DATE NOT NULL,
    UNIQUE(fund_id, date)
);