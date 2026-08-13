package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/JavierZam/quant-indonesia-scraping/config"
	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"
	geminiTimeout = 30 * time.Second
)

// fallbackModels is the priority order of models to try when one fails.
// The system will try each model in order until one succeeds.
var fallbackModels = []string{
	"gemini-3.1-flash-lite",
	"gemini-3.6-flash",
	"gemini-3.5-flash",
	"gemini-flash-latest",
	"gemini-pro-latest",
}

// geminiRequest represents the Gemini API request body.
type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature      float64 `json:"temperature"`
	MaxOutputTokens  int     `json:"maxOutputTokens"`
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
// Supports automatic model fallback and multi-API-key rotation when rate-limited.
type GeminiAnalyzer struct {
	apiKeys        []string
	primaryModel   string
	fallbackModels []string
	httpClient     *http.Client
	logger         *slog.Logger

	mu           sync.RWMutex
	activeKeyIdx int
	activeModel  string // last model that worked successfully
}

// NewGeminiAnalyzer creates a new Gemini-based LLM analyzer with auto-fallback and multi-key support.
func NewGeminiAnalyzer(cfg config.LLMConfig, logger *slog.Logger) *GeminiAnalyzer {
	keys := cfg.APIKeys()

	primary := cfg.Model
	if primary == "" {
		primary = "gemini-3.1-flash-lite"
	}

	// Build fallback list: primary first, then all fallbacks (excluding primary to avoid dupes)
	models := []string{primary}
	for _, m := range fallbackModels {
		if m != primary {
			models = append(models, m)
		}
	}

	return &GeminiAnalyzer{
		apiKeys:        keys,
		primaryModel:   primary,
		fallbackModels: models,
		activeModel:    primary,
		httpClient: &http.Client{
			Timeout: geminiTimeout,
		},
		logger: logger,
	}
}

// Analyze sends article content to Gemini for sentiment analysis and entity extraction.
// If the active model fails with 404, it switches to a fallback model.
// If the active model fails with 429 (rate limit), it rotates the API key or waits and retries.
func (g *GeminiAnalyzer) Analyze(ctx context.Context, title, content string) (*domain.AnalysisResult, error) {
	if len(g.apiKeys) == 0 {
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

	// Try the last known working model first, then fallbacks
	modelsToTry := g.getModelsToTry()
	var lastErr error

	for _, model := range modelsToTry {
		// For each model, allow up to 3 retries on rate-limit (429)
		for retry := 0; retry < 3; retry++ {
			result, err := g.callModel(ctx, model, jsonBody)
			if err == nil {
				// Success! Remember this model for next time
				g.setActiveModel(model)
				return g.parseResult(title, result)
			}

			lastErr = err

			if isRateLimitError(err) {
				// If we have multiple API keys, rotate key immediately!
				if len(g.apiKeys) > 1 {
					newKeyIdx := g.rotateAPIKey()
					g.logger.Warn("rate limited, rotated to next API key",
						"model", model,
						"key_index", newKeyIdx,
					)
					// Immediately retry with new key without waiting
					continue
				}

				// Single API key — wait and retry
				waitDuration := time.Duration(5+retry*3) * time.Second
				g.logger.Warn("rate limited, waiting before retry",
					"model", model,
					"retry", retry+1,
					"wait_seconds", waitDuration.Seconds(),
				)
				select {
				case <-time.After(waitDuration):
					continue // retry same model
				case <-ctx.Done():
					return nil, fmt.Errorf("%w: context cancelled during rate-limit wait", domain.ErrLLMProcessing)
				}
			}

			if isModelNotFoundError(err) {
				// 404 Model not found — switch to next model
				g.logger.Warn("model unavailable, trying next fallback",
					"failed_model", model,
					"error", err,
				)
				break // break retry loop, try next model
			}

			// Other errors — don't retry, don't fallback
			return nil, err
		}
	}

	return nil, fmt.Errorf("%w: all models exhausted, last error: %v", domain.ErrLLMProcessing, lastErr)
}

// callModel makes a single API call to a specific model using the active API key.
func (g *GeminiAnalyzer) callModel(ctx context.Context, model string, jsonBody []byte) (string, error) {
	apiKey := g.getActiveAPIKey()
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, model, apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: gemini request failed: %v", domain.ErrLLMProcessing, err)
	}
	defer resp.Body.Close()

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("%w: decoding gemini response: %v", domain.ErrLLMProcessing, err)
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("%w: gemini API error %d: %s",
			domain.ErrLLMProcessing, geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 ||
		len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("%w: empty response from gemini", domain.ErrLLMProcessing)
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// parseResult parses the raw JSON response from Gemini into a domain.AnalysisResult.
func (g *GeminiAnalyzer) parseResult(title, rawJSON string) (*domain.AnalysisResult, error) {
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
		"model", g.getActiveModel(),
		"sentiment", result.SentimentLabel,
		"score", result.SentimentScore,
		"tags", len(result.Tags),
	)

	return &result, nil
}

// getModelsToTry returns the list of models to try, starting with the last known working model.
func (g *GeminiAnalyzer) getModelsToTry() []string {
	g.mu.RLock()
	active := g.activeModel
	g.mu.RUnlock()

	// Put active model first, then append all others
	models := []string{active}
	for _, m := range g.fallbackModels {
		if m != active {
			models = append(models, m)
		}
	}
	return models
}

// setActiveModel updates the last known working model (thread-safe).
func (g *GeminiAnalyzer) setActiveModel(model string) {
	g.mu.Lock()
	if g.activeModel != model {
		g.logger.Info("switched active Gemini model",
			"previous", g.activeModel,
			"new", model,
		)
		g.activeModel = model
	}
	g.mu.Unlock()
}

// getActiveModel returns the current active model (thread-safe).
func (g *GeminiAnalyzer) getActiveModel() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.activeModel
}

// getActiveAPIKey returns the current active API key (thread-safe).
func (g *GeminiAnalyzer) getActiveAPIKey() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(g.apiKeys) == 0 {
		return ""
	}
	return g.apiKeys[g.activeKeyIdx]
}

// rotateAPIKey switches to the next API key in round-robin fashion (thread-safe).
func (g *GeminiAnalyzer) rotateAPIKey() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.apiKeys) > 1 {
		g.activeKeyIdx = (g.activeKeyIdx + 1) % len(g.apiKeys)
	}
	return g.activeKeyIdx
}

// isRateLimitError checks if the error is a 429 rate-limit (quota exceeded).
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsStr(errStr, "error 429") || containsStr(errStr, "RESOURCE_EXHAUSTED") || containsStr(errStr, "quota")
}

// isModelNotFoundError checks if the error is a 404 model-not-found.
func isModelNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsStr(errStr, "error 404") || containsStr(errStr, "not found") || containsStr(errStr, "no longer available")
}

// containsStr is a simple case-insensitive substring check.
func containsStr(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}


