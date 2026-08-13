package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Server    ServerConfig
	DB        DBConfig
	Valkey    ValkeyConfig
	LLM       LLMConfig
	Ingestion IngestionConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	RateLimitRPS   int
	RateLimitBurst int
}

// DBConfig holds PostgreSQL connection configuration.
type DBConfig struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

// DSN returns the PostgreSQL connection string.
func (c *DBConfig) DSN() string {
	if c.URL != "" {
		return c.URL
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

// ValkeyConfig holds Valkey connection configuration.
type ValkeyConfig struct {
	Addr     string
	Password string
	DB       int
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	Provider string
	APIKey   string
	Model    string
}

// APIKeys returns a slice of API keys, split by comma if multiple are provided.
func (c *LLMConfig) APIKeys() []string {
	if c.APIKey == "" {
		return nil
	}
	rawKeys := strings.Split(c.APIKey, ",")
	var keys []string
	for _, k := range rawKeys {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

// IngestionConfig holds ingestion worker pool configuration.
type IngestionConfig struct {
	Workers      int
	RateLimitRPS int
	RetryMax     int
	CronSchedule string
}

// Load reads configuration from environment variables (loading .env if present).
// In production on GCP Cloud Run, secrets are injected as env vars via Secret Manager.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   5 * time.Minute,
			RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 10),
			RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 20),
		},
		DB: DBConfig{
			URL:      getEnv("DATABASE_URL", getEnv("DB_URL", "")),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "quantuser"),
			Password: getEnv("DB_PASSWORD", "quantpass"),
			Name:     getEnv("DB_NAME", "quantintel"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxConns: int32(getEnvInt("DB_MAX_CONNS", 25)),
			MinConns: int32(getEnvInt("DB_MIN_CONNS", 5)),
		},
		Valkey: ValkeyConfig{
			Addr:     getEnv("VALKEY_ADDR", "localhost:6379"),
			Password: getEnv("VALKEY_PASSWORD", ""),
			DB:       getEnvInt("VALKEY_DB", 0),
		},
		LLM: LLMConfig{
			Provider: getEnv("LLM_PROVIDER", "gemini"),
			APIKey:   getEnv("LLM_API_KEY", ""),
			Model:    getEnv("LLM_MODEL", "gemini-2.5-flash"),
		},
		Ingestion: IngestionConfig{
			Workers:      getEnvInt("INGESTION_WORKERS", 10),
			RateLimitRPS: getEnvInt("INGESTION_RATE_LIMIT_RPS", 5),
			RetryMax:     getEnvInt("INGESTION_RETRY_MAX", 3),
			CronSchedule: getEnv("INGESTION_CRON_SCHEDULE", "*/30 * * * *"),
		},
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// getEnvInt retrieves an environment variable as int or returns a default.
func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
