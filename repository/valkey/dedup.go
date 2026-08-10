package valkey

import (
	"context"
	"fmt"
	"time"

	valkeylib "github.com/valkey-io/valkey-go"
)

const (
	// dedupKeyPrefix is the key prefix for URL deduplication entries in Valkey.
	dedupKeyPrefix = "dedup:url:"

	// dedupTTL is how long a URL hash is remembered. Set to 7 days to prevent
	// re-ingesting articles that were recently processed.
	dedupTTL = 7 * 24 * time.Hour
)

// DedupCache implements domain.DeduplicationCache using Valkey.
type DedupCache struct {
	client valkeylib.Client
}

// NewDedupCache creates a new deduplication cache backed by Valkey.
func NewDedupCache(client valkeylib.Client) *DedupCache {
	return &DedupCache{client: client}
}

// Exists checks whether a URL hash has already been ingested.
// Returns true if the hash exists in the cache, false otherwise.
func (d *DedupCache) Exists(ctx context.Context, urlHash string) (bool, error) {
	key := dedupKeyPrefix + urlHash

	cmd := d.client.B().Exists().Key(key).Build()
	result, err := d.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, fmt.Errorf("checking dedup existence for %s: %w", urlHash, err)
	}

	return result > 0, nil
}

// Set marks a URL hash as ingested by storing it in Valkey with a TTL.
// This prevents the same URL from being re-processed within the TTL window.
func (d *DedupCache) Set(ctx context.Context, urlHash string) error {
	key := dedupKeyPrefix + urlHash

	cmd := d.client.B().Set().Key(key).Value("1").Ex(dedupTTL).Build()
	if err := d.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("setting dedup key for %s: %w", urlHash, err)
	}

	return nil
}
