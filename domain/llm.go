package domain

import "context"

// AnalysisResult holds the structured output from LLM analysis of a news article.
type AnalysisResult struct {
	SentimentScore float64        `json:"sentiment_score"`
	SentimentLabel SentimentLabel `json:"sentiment_label"`
	Summary        string         `json:"summary"`
	Tags           []ExtractedTag `json:"tags"`
}

// ExtractedTag represents an entity extracted by the LLM from article content.
type ExtractedTag struct {
	Type           string  `json:"type"` // "company", "executive", "sector"
	Value          string  `json:"value"`
	TickerSymbol   string  `json:"ticker_symbol,omitempty"`
	RelevanceScore float64 `json:"relevance_score"`
}

// LLMAnalyzer defines the interface for AI-powered article analysis.
type LLMAnalyzer interface {
	// Analyze processes raw article content and returns structured analysis.
	Analyze(ctx context.Context, title string, content string) (*AnalysisResult, error)
}
