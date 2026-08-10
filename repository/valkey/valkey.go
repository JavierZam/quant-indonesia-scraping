package valkey

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"

	"github.com/javier-garcia/quant-indonesia-scraping/config"
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

	// Verify connectivity
	pingCmd := client.B().Ping().Build()
	if err := client.Do(context.Background(), pingCmd).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("pinging valkey: %w", err)
	}

	return client, nil
}
