CREATE TABLE IF NOT EXISTS stock_prices (
    symbol VARCHAR(10) NOT NULL REFERENCES stocks(symbol) ON DELETE CASCADE,
    date DATE NOT NULL,
    close_price NUMERIC(15, 2) NOT NULL,
    open_price NUMERIC(15, 2),
    high_price NUMERIC(15, 2),
    low_price NUMERIC(15, 2),
    volume BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (symbol, date)
);

CREATE INDEX IF NOT EXISTS idx_stock_prices_symbol_date ON stock_prices (symbol, date DESC);

CREATE TABLE IF NOT EXISTS broker_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(10) NOT NULL REFERENCES stocks(symbol) ON DELETE CASCADE,
    date DATE NOT NULL,
    net_foreign_buy_sell NUMERIC(20, 2) NOT NULL,
    top_buyer VARCHAR(50),
    top_seller VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_symbol_date_broker UNIQUE (symbol, date)
);
