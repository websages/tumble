package handler

import (
	"testing"
	"time"

	"tumble/internal/data"
)

func TestIsPreviewExpired(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		age      time.Duration
		expected bool
	}{
		{
			name:     "fresh normal preview (1 day old)",
			data:     `{"title":"Example","description":"A page"}`,
			age:      24 * time.Hour,
			expected: false,
		},
		{
			name:     "stale normal preview (11 days old)",
			data:     `{"title":"Example","description":"A page"}`,
			age:      11 * 24 * time.Hour,
			expected: true,
		},
		{
			name:     "fresh error preview (1 day old)",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      24 * time.Hour,
			expected: false,
		},
		{
			name:     "error preview within TTL (170 days old)",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      170 * 24 * time.Hour,
			expected: false,
		},
		{
			name:     "stale error preview (181 days old)",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      181 * 24 * time.Hour,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := &data.LinkPreview{
				URL:       "https://example.com",
				Data:      []byte(tt.data),
				UpdatedAt: time.Now().Add(-tt.age),
			}
			result := isPreviewExpired(cached)
			if result != tt.expected {
				t.Errorf("isPreviewExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}
