package handler

import (
	"testing"
	"time"

	"tumble/internal/data"
)

func TestIsPreviewExpired(t *testing.T) {
	tests := []struct {
		name          string
		data          string
		age           time.Duration
		linkAge       time.Duration // age of the link itself; 0 means zero-value timestamp
		zeroTimestamp bool          // if true, pass zero time
		expected      bool
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
			name:     "fresh error on recent link (1d error, 5d link) - expired (24h TTL)",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      24 * time.Hour,
			linkAge:  5 * 24 * time.Hour,
			expected: true,
		},
		{
			name:     "fresh error on recent link (12h error, 5d link) - not expired",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      12 * time.Hour,
			linkAge:  5 * 24 * time.Hour,
			expected: false,
		},
		{
			name:     "error on old link (30d error, 90d link) - not expired (60d TTL)",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      30 * 24 * time.Hour,
			linkAge:  90 * 24 * time.Hour,
			expected: false,
		},
		{
			name:     "stale error on old link (61d error, 90d link) - expired",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      61 * 24 * time.Hour,
			linkAge:  90 * 24 * time.Hour,
			expected: true,
		},
		{
			name:          "error with zero linkTimestamp uses old TTL - not expired at 30d",
			data:          `{"error":"Video Unavailable","status":"404"}`,
			age:           30 * 24 * time.Hour,
			zeroTimestamp: true,
			expected:      false,
		},
		{
			name:          "error with zero linkTimestamp uses old TTL - expired at 61d",
			data:          `{"error":"Video Unavailable","status":"404"}`,
			age:           61 * 24 * time.Hour,
			zeroTimestamp: true,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := &data.LinkPreview{
				URL:       "https://example.com",
				Data:      []byte(tt.data),
				UpdatedAt: time.Now().Add(-tt.age),
			}
			var linkTimestamp time.Time
			if !tt.zeroTimestamp && tt.linkAge > 0 {
				linkTimestamp = time.Now().Add(-tt.linkAge)
			}
			result := isPreviewExpired(cached, linkTimestamp)
			if result != tt.expected {
				t.Errorf("isPreviewExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}
