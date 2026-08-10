package ingestion

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/javier-garcia/quant-indonesia-scraping/pkg/httpclient"
)

// articleSelectors are CSS selectors commonly used for article body content,
// ordered by specificity. The scraper tries each until it finds content.
var articleSelectors = []string{
	"article",
	"[role='main']",
	".article-body",
	".article-content",
	".post-content",
	".entry-content",
	".story-body",
	".content-body",
	"main",
	".content",
}

// Scraper extracts article content from web pages.
type Scraper struct {
	httpClient *httpclient.Client
	logger     *slog.Logger
}

// NewScraper creates a new Scraper instance.
func NewScraper(httpClient *httpclient.Client, logger *slog.Logger) *Scraper {
	return &Scraper{
		httpClient: httpClient,
		logger:     logger,
	}
}

// ScrapeArticle fetches the URL and extracts the main article text content.
// Returns the cleaned text body.
func (s *Scraper) ScrapeArticle(ctx context.Context, url string) (string, error) {
	body, statusCode, err := s.httpClient.Get(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetching article %s: %w", url, err)
	}

	if statusCode < 200 || statusCode >= 300 {
		return "", fmt.Errorf("article %s returned HTTP %d", url, statusCode)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("parsing HTML for %s: %w", url, err)
	}

	// Remove unwanted elements before extraction
	doc.Find("script, style, nav, header, footer, iframe, noscript, .ad, .advertisement, .social-share, .comments").Remove()

	// Try each selector in order of specificity
	for _, selector := range articleSelectors {
		selection := doc.Find(selector)
		if selection.Length() > 0 {
			text := cleanText(selection.Text())
			if len(text) > 100 { // Only accept if there's substantial content
				s.logger.Debug("scraped article",
					"url", url,
					"selector", selector,
					"length", len(text),
				)
				return text, nil
			}
		}
	}

	// Fallback: extract all paragraph text
	var paragraphs []string
	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		text := cleanText(s.Text())
		if len(text) > 20 {
			paragraphs = append(paragraphs, text)
		}
	})

	if len(paragraphs) > 0 {
		result := strings.Join(paragraphs, "\n\n")
		s.logger.Debug("scraped article via paragraph fallback",
			"url", url,
			"paragraphs", len(paragraphs),
		)
		return result, nil
	}

	return "", fmt.Errorf("no article content found at %s", url)
}

// cleanText normalizes whitespace and trims extracted text.
func cleanText(s string) string {
	// Replace multiple whitespace/newlines with single spaces
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
