package llm

import (
	"fmt"
	"log/slog"

	"github.com/javier-garcia/quant-indonesia-scraping/config"
	"github.com/javier-garcia/quant-indonesia-scraping/domain"
)

// NewAnalyzer creates an LLM analyzer based on the configured provider.
func NewAnalyzer(cfg config.LLMConfig, logger *slog.Logger) (domain.LLMAnalyzer, error) {
	switch cfg.Provider {
	case "gemini":
		return NewGeminiAnalyzer(cfg, logger), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: gemini)", cfg.Provider)
	}
}
