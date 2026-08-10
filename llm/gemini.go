package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/javier-garcia/quant-indonesia-scraping/config"
	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

const (
	geminiBaseURL  = "https://generativelanguage.googleapis.com/v1beta/models"
	geminiTimeout  = 30 * time.Second
)

// geminiRequest represents the Gemini API request body.
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
}

// geminiResponse represents the Gemini API response.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// GeminiAnalyzer implements domain.LLMAnalyzer using Google's Gemini API.
type GeminiAnalyzer struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewGeminiAnalyzer creates a new Gemini-based LLM analyzer.
func NewGeminiAnalyzer(cfg config.LLMConfig, logger *slog.Logger) *GeminiAnalyzer {
	return &GeminiAnalyzer{
		apiKey: cfg.APIKey,
		model:  cfg.Model,
		httpClient: &http.Client{
			Timeout: geminiTimeout,
		},
		logger: logger,
	}
}

// Analyze sends article content to Gemini for sentiment analysis and entity extraction.
func (g *GeminiAnalyzer) Analyze(ctx context.Context, title, content string) (*domain.AnalysisResult, error) {
	if g.apiKey == "" {
		return nil, fmt.Errorf("%w: gemini API key not configured", domain.ErrLLMProcessing)
	}

	userPrompt := buildUserPrompt(title, content)

	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: userPrompt}},
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			Temperature:      0.1,
			MaxOutputTokens:  2048,
			ResponseMimeType: "application/json",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling gemini request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, g.model, g.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: gemini request failed: %v", domain.ErrLLMProcessing, err)
	}
	defer resp.Body.Close()

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("%w: decoding gemini response: %v", domain.ErrLLMProcessing, err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("%w: gemini API error %d: %s",
			domain.ErrLLMProcessing, geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 ||
		len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("%w: empty response from gemini", domain.ErrLLMProcessing)
	}

	rawJSON := geminiResp.Candidates[0].Content.Parts[0].Text

	var result domain.AnalysisResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		g.logger.Error("failed to parse LLM JSON output",
			"raw", rawJSON,
			"error", err,
		)
		return nil, fmt.Errorf("%w: parsing analysis result: %v", domain.ErrLLMProcessing, err)
	}

	// Validate and clamp sentiment score
	if result.SentimentScore < -1.0 {
		result.SentimentScore = -1.0
	}
	if result.SentimentScore > 1.0 {
		result.SentimentScore = 1.0
	}

	// Validate sentiment label
	switch result.SentimentLabel {
	case domain.SentimentBullish, domain.SentimentBearish, domain.SentimentNeutral:
		// Valid
	default:
		g.logger.Warn("invalid sentiment label from LLM, defaulting to Neutral",
			"received", result.SentimentLabel,
		)
		result.SentimentLabel = domain.SentimentNeutral
	}

	g.logger.Info("LLM analysis complete",
		"title", title,
		"sentiment", result.SentimentLabel,
		"score", result.SentimentScore,
		"tags", len(result.Tags),
	)

	return &result, nil
}
