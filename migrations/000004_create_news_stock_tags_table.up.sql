CREATE TABLE IF NOT EXISTS news_stock_tags (
    news_id         UUID NOT NULL REFERENCES news_articles(id) ON DELETE CASCADE,
    symbol          VARCHAR(20) NOT NULL REFERENCES stocks(symbol) ON DELETE CASCADE,
    relevance_score NUMERIC(3,2) CHECK (relevance_score >= 0.0 AND relevance_score <= 1.0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (news_id, symbol)
);

CREATE INDEX idx_news_stock_tags_symbol ON news_stock_tags(symbol);
CREATE INDEX idx_news_stock_tags_news_id ON news_stock_tags(news_id);
