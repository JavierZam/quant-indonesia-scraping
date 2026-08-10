CREATE TABLE IF NOT EXISTS executives (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol      VARCHAR(20) NOT NULL REFERENCES stocks(symbol) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    title       VARCHAR(255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_executives_symbol ON executives(symbol);
CREATE INDEX idx_executives_name ON executives(name);
