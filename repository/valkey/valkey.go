package valkey

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/JavierZam/quant-indonesia-scraping/config"
)

// NewValkeyClient creates and validates a new Valkey client connection.
func NewValkeyClient(cfg config.ValkeyConfig) (valkey.Client, error) {
	opts := valkey.ClientOption{
		InitAddress: []string{cfg.Addr},
	}

	if cfg.Password != "" {
		opts.Password = cfg.Password
	}

	client, err := valkey.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf("creating valkey client: %w", err)
	}

	// Verify connectivity with a 1-second timeout
	pingCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	pingCmd := client.B().Ping().Build()
	if err := client.Do(pingCtx, pingCmd).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("pinging valkey: %w", err)
	}

	return client, nil
}
