CREATE TABLE IF NOT EXISTS news_articles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url_hash        VARCHAR(32) NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    url             TEXT NOT NULL,
    source          VARCHAR(100),
    summary         TEXT,
    content_raw     TEXT,
    sentiment_score NUMERIC(4,3) CHECK (sentiment_score >= -1.0 AND sentiment_score <= 1.0),
    sentiment_label VARCHAR(20) CHECK (sentiment_label IN ('Bullish', 'Bearish', 'Neutral')),
    published_at    TIMESTAMPTZ,
    ingested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_news_articles_url_hash ON news_articles(url_hash);
CREATE INDEX idx_news_articles_published_at ON news_articles(published_at DESC);
CREATE INDEX idx_news_articles_sentiment_label ON news_articles(sentiment_label);
CREATE INDEX idx_news_articles_source ON news_articles(source);
