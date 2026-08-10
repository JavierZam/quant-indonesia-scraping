package hasher

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// MD5Hash computes the MD5 hash of the given input string.
// Used for URL deduplication — not for cryptographic purposes.
func MD5Hash(input string) string {
	// Normalize: trim whitespace and lowercase
	normalized := strings.ToLower(strings.TrimSpace(input))
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}
