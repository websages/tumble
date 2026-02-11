package archive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"url": "https://example.com",
			"archived_snapshots": {
				"closest": {
					"status": "200",
					"available": true,
					"url": "https://web.archive.org/web/20230615120000/https://example.com",
					"timestamp": "20230615120000"
				}
			}
		}`))
	}))
	defer server.Close()

	c := NewClient(100)
	c.endpoint = server.URL
	c.httpClient = server.Client()

	result, err := c.Check(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Found {
		t.Fatal("expected Found to be true")
	}

	expectedURL := "https://web.archive.org/web/20230615120000/https://example.com"
	if result.ArchiveURL != expectedURL {
		t.Errorf("ArchiveURL = %q, want %q", result.ArchiveURL, expectedURL)
	}

	expectedTime := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)
	if !result.SnapshotAt.Equal(expectedTime) {
		t.Errorf("SnapshotAt = %v, want %v", result.SnapshotAt, expectedTime)
	}
}

func TestCheckNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"archived_snapshots": {}}`))
	}))
	defer server.Close()

	c := NewClient(100)
	c.endpoint = server.URL
	c.httpClient = server.Client()

	result, err := c.Check(context.Background(), "https://no-snapshot.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Found {
		t.Fatal("expected Found to be false")
	}
}

func TestCheckAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient(100)
	c.endpoint = server.URL
	c.httpClient = server.Client()

	_, err := c.Check(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected an error for 500 status")
	}
}

func TestTimestampParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			input:    "20230615120000",
			expected: time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			input:    "20000101000000",
			expected: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			input:    "20241231235959",
			expected: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			input:   "invalid",
			wantErr: true,
		},
		{
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseWaybackTimestamp(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("parseWaybackTimestamp(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
