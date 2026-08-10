package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/javier-garcia/quant-indonesia-scraping/pkg/httpclient"
)

// FeedItem represents a single parsed item from an RSS/Atom feed.
type FeedItem struct {
	Title       string
	URL         string
	Source      string
	Summary     string
	PublishedAt *time.Time
}

// FeedSource defines an RSS feed to ingest.
type FeedSource struct {
	Name string
	URL  string
}

// FeedParser fetches and parses RSS/Atom feeds.
type FeedParser struct {
	httpClient *httpclient.Client
	parser     *gofeed.Parser
	logger     *slog.Logger
}

// NewFeedParser creates a new FeedParser instance.
func NewFeedParser(httpClient *httpclient.Client, logger *slog.Logger) *FeedParser {
	return &FeedParser{
		httpClient: httpClient,
		parser:     gofeed.NewParser(),
		logger:     logger,
	}
}

// ParseFeed fetches and parses an RSS feed, returning the items.
func (fp *FeedParser) ParseFeed(ctx context.Context, source FeedSource) ([]FeedItem, error) {
	body, statusCode, err := fp.httpClient.Get(ctx, source.URL)
	if err != nil {
		return nil, fmt.Errorf("fetching feed %s: %w", source.Name, err)
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("feed %s returned HTTP %d", source.Name, statusCode)
	}

	feed, err := fp.parser.ParseString(string(body))
	if err != nil {
		return nil, fmt.Errorf("parsing feed %s: %w", source.Name, err)
	}

	items := make([]FeedItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		if item.Link == "" {
			continue // Skip items without a URL
		}

		feedItem := FeedItem{
			Title:  item.Title,
			URL:    item.Link,
			Source: source.Name,
		}

		// Use item description as summary
		if item.Description != "" {
			feedItem.Summary = item.Description
		}

		// Parse published date
		if item.PublishedParsed != nil {
			t := *item.PublishedParsed
			feedItem.PublishedAt = &t
		} else if item.UpdatedParsed != nil {
			t := *item.UpdatedParsed
			feedItem.PublishedAt = &t
		}

		items = append(items, feedItem)
	}

	fp.logger.Info("parsed feed",
		"source", source.Name,
		"items", len(items),
	)

	return items, nil
}
