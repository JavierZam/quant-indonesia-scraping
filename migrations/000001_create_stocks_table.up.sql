CREATE TABLE IF NOT EXISTS stocks (
    symbol      VARCHAR(20) PRIMARY KEY,
    company_name VARCHAR(255) NOT NULL,
    sector      VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stocks_sector ON stocks(sector);
