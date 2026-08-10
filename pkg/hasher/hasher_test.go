package hasher

import (
	"testing"
)

func TestMD5Hash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple URL",
			input:    "https://example.com/article/123",
			expected: MD5Hash("https://example.com/article/123"),
		},
		{
			name:     "same URL different case produces same hash",
			input:    "HTTPS://EXAMPLE.COM/ARTICLE/123",
			expected: MD5Hash("https://example.com/article/123"),
		},
		{
			name:     "trimmed whitespace produces same hash",
			input:    "  https://example.com/article/123  ",
			expected: MD5Hash("https://example.com/article/123"),
		},
		{
			name:     "hash is 32 hex characters",
			input:    "https://finance.yahoo.com/news",
			expected: MD5Hash("https://finance.yahoo.com/news"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MD5Hash(tt.input)
			if result != tt.expected {
				t.Errorf("MD5Hash(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			// Verify length is always 32 (MD5 hex)
			if len(result) != 32 {
				t.Errorf("MD5Hash length = %d, want 32", len(result))
			}
		})
	}
}

func TestMD5Hash_Deterministic(t *testing.T) {
	url := "https://idx.co.id/en/news/press-release/123"
	hash1 := MD5Hash(url)
	hash2 := MD5Hash(url)
	if hash1 != hash2 {
		t.Errorf("MD5Hash is not deterministic: %q != %q", hash1, hash2)
	}
}

func TestMD5Hash_DifferentInputs(t *testing.T) {
	hash1 := MD5Hash("https://example.com/article/1")
	hash2 := MD5Hash("https://example.com/article/2")
	if hash1 == hash2 {
		t.Error("different URLs should produce different hashes")
	}
}
