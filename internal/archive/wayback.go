package archive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

const waybackEndpoint = "https://archive.org/wayback/available"

// Result holds the outcome of a Wayback Machine availability check.
type Result struct {
	ArchiveURL string
	SnapshotAt time.Time
	Found      bool
}

// Client checks the Wayback Machine Availability API with rate limiting.
type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	endpoint   string
}

// NewClient creates a Wayback client with the given rate limit (requests/sec).
func NewClient(rps float64) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		limiter:    rate.NewLimiter(rate.Limit(rps), 1),
		endpoint:   waybackEndpoint,
	}
}

// waybackResponse matches the JSON structure from archive.org.
type waybackResponse struct {
	ArchivedSnapshots struct {
		Closest *struct {
			URL       string `json:"url"`
			Timestamp string `json:"timestamp"`
			Status    string `json:"status"`
			Available bool   `json:"available"`
		} `json:"closest"`
	} `json:"archived_snapshots"`
}

// parseWaybackTimestamp parses the Wayback Machine timestamp format (YYYYMMDDHHmmss).
func parseWaybackTimestamp(ts string) (time.Time, error) {
	return time.Parse("20060102150405", ts)
}

// Check queries the Wayback Machine for a snapshot of the given URL.
// It blocks until the rate limiter allows the request.
func (c *Client) Check(ctx context.Context, targetURL string) (*Result, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	reqURL := fmt.Sprintf("%s?url=%s", c.endpoint, url.QueryEscape(targetURL))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("wayback API status %d", resp.StatusCode)
	}

	var data waybackResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	snap := data.ArchivedSnapshots.Closest
	if snap == nil || !snap.Available || snap.URL == "" {
		return &Result{Found: false}, nil
	}

	snapshotAt, err := parseWaybackTimestamp(snap.Timestamp)
	if err != nil {
		return &Result{
			ArchiveURL: snap.URL,
			Found:      true,
		}, nil
	}

	return &Result{
		ArchiveURL: snap.URL,
		SnapshotAt: snapshotAt,
		Found:      true,
	}, nil
}
