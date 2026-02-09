# API v1 Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a RESTful v1 API with consistent JSON responses, proper status codes, and clean resource-oriented endpoints.

**Architecture:** New handlers in `internal/handler/api_v1.go` with shared response helpers. Routes registered under `/api/v1/` prefix. Existing handlers remain unchanged for backward compatibility during transition.

**Tech Stack:** Go 1.25, standard library `net/http`, GORM, `encoding/json`

**Design Document:** `docs/plans/2026-02-08-api-redesign-design.md`

---

## Phase 1: Foundation (Response Structures & Helpers)

### Task 1.1: Create API v1 Response Types

**Files:**
- Create: `internal/handler/api_v1_types.go`
- Test: `internal/handler/api_v1_types_test.go`

**Step 1: Write the test for error response JSON structure**

```go
// internal/handler/api_v1_types_test.go
package handler

import (
	"encoding/json"
	"testing"
)

func TestAPIErrorResponse_JSON(t *testing.T) {
	err := APIError{
		Code:    "not_found",
		Message: "Link 42 does not exist",
	}
	resp := APIErrorResponse{Error: err}

	data, jsonErr := json.Marshal(resp)
	if jsonErr != nil {
		t.Fatalf("failed to marshal: %v", jsonErr)
	}

	expected := `{"error":{"code":"not_found","message":"Link 42 does not exist"}}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", string(data), expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIErrorResponse_JSON -v`

Expected: FAIL with "undefined: APIError"

**Step 3: Write the types**

```go
// internal/handler/api_v1_types.go
package handler

import "time"

// APIError represents a structured error response
type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// APIErrorResponse wraps an error for JSON responses
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// APIMeta contains pagination metadata
type APIMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// APILinkResponse represents a single link in API responses
type APILinkResponse struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	User      string    `json:"user"`
	Clicks    int       `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}

// APILinkCreateResponse includes duplicate detection info
type APILinkCreateResponse struct {
	APILinkResponse
	IsDuplicate         bool                      `json:"is_duplicate"`
	PreviousSubmissions []APIPreviousSubmission   `json:"previous_submissions,omitempty"`
}

// APIPreviousSubmission represents a prior submission of the same URL
type APIPreviousSubmission struct {
	ID        int       `json:"id"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	Title     string    `json:"title"`
}

// APILinksResponse is a paginated list of links
type APILinksResponse struct {
	Data []APILinkResponse `json:"data"`
	Meta APIMeta           `json:"meta"`
}

// APIQuoteResponse represents a single quote
type APIQuoteResponse struct {
	ID        int       `json:"id"`
	Quote     string    `json:"quote"`
	Author    string    `json:"author,omitempty"`
	Poster    string    `json:"poster,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// APIQuotesResponse is a paginated list of quotes
type APIQuotesResponse struct {
	Data []APIQuoteResponse `json:"data"`
	Meta APIMeta            `json:"meta"`
}

// APISiteStats represents site-wide statistics
type APISiteStats struct {
	TotalLinks  int `json:"total_links"`
	TotalQuotes int `json:"total_quotes"`
	TotalUsers  int `json:"total_users"`
}

// APIUserStats represents a user's statistics
type APIUserStats struct {
	User       string `json:"user"`
	LinkCount  int    `json:"link_count"`
	QuoteCount int    `json:"quote_count"`
}

// APIStatsResponse includes site stats and leaderboard
type APIStatsResponse struct {
	Site        APISiteStats   `json:"site"`
	Leaderboard []APIUserStats `json:"leaderboard"`
	Meta        APIMeta        `json:"meta"`
}

// APISearchResponse contains search results in separate arrays
type APISearchResponse struct {
	Links  []APILinkResponse  `json:"links"`
	Quotes []APIQuoteResponse `json:"quotes"`
	Meta   APISearchMeta      `json:"meta"`
}

// APISearchMeta extends APIMeta with per-type totals
type APISearchMeta struct {
	TotalLinks  int `json:"total_links"`
	TotalQuotes int `json:"total_quotes"`
	Limit       int `json:"limit"`
	Offset      int `json:"offset"`
}

// APICacheResponse for cache operations
type APICacheResponse struct {
	Cleared string `json:"cleared"`
	Count   int    `json:"count,omitempty"`
}

// APIKittenResponse for daily kitten
type APIKittenResponse struct {
	URL     string `json:"url"`
	Date    string `json:"date"`
	Fetched bool   `json:"fetched,omitempty"`
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIErrorResponse_JSON -v`

Expected: PASS

**Step 5: Add more type tests**

```go
// Add to internal/handler/api_v1_types_test.go

func TestAPILinkResponse_JSON(t *testing.T) {
	link := APILinkResponse{
		ID:        42,
		URL:       "https://example.com",
		Title:     "Example",
		User:      "alice",
		Clicks:    5,
		CreatedAt: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(link)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify key fields are present with snake_case
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["id"] != float64(42) {
		t.Errorf("id: got %v, want 42", result["id"])
	}
	if result["created_at"] == nil {
		t.Error("created_at should be present")
	}
	if result["createdAt"] != nil {
		t.Error("should use snake_case, not camelCase")
	}
}

func TestAPIErrorResponse_WithDetails(t *testing.T) {
	err := APIError{
		Code:    "validation_error",
		Message: "Invalid input",
		Details: map[string]string{
			"url":  "must be a valid URL",
			"user": "is required",
		},
	}
	resp := APIErrorResponse{Error: err}

	data, jsonErr := json.Marshal(resp)
	if jsonErr != nil {
		t.Fatalf("failed to marshal: %v", jsonErr)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	errObj := result["error"].(map[string]interface{})
	details := errObj["details"].(map[string]interface{})

	if details["url"] != "must be a valid URL" {
		t.Errorf("details.url: got %v", details["url"])
	}
}
```

**Step 6: Run all type tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run "TestAPI.*_JSON\|TestAPIError" -v`

Expected: PASS (3 tests)

**Step 7: Commit**

```bash
git add internal/handler/api_v1_types.go internal/handler/api_v1_types_test.go
git commit -m "feat(api): add v1 API response types with snake_case JSON"
```

---

### Task 1.2: Create API v1 Response Helpers

**Files:**
- Create: `internal/handler/api_v1_helpers.go`
- Test: `internal/handler/api_v1_helpers_test.go`

**Step 1: Write tests for response helpers**

```go
// internal/handler/api_v1_helpers_test.go
package handler

import (
	"net/http"
	"net/http/httptest"
	"encoding/json"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("content-type: got %s", w.Header().Get("Content-Type"))
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["status"] != "ok" {
		t.Errorf("body: got %v", result)
	}
}

func TestWriteAPIError(t *testing.T) {
	w := httptest.NewRecorder()

	writeAPIError(w, http.StatusNotFound, "not_found", "Link 42 does not exist")

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}

	var result APIErrorResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Error.Code != "not_found" {
		t.Errorf("code: got %s", result.Error.Code)
	}
	if result.Error.Message != "Link 42 does not exist" {
		t.Errorf("message: got %s", result.Error.Message)
	}
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]string{
		"url": "must be a valid URL",
	}

	writeValidationError(w, details)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}

	var result APIErrorResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Error.Code != "validation_error" {
		t.Errorf("code: got %s", result.Error.Code)
	}
	if result.Error.Details["url"] != "must be a valid URL" {
		t.Errorf("details: got %v", result.Error.Details)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run "TestWrite" -v`

Expected: FAIL with "undefined: writeJSON"

**Step 3: Implement helpers**

```go
// internal/handler/api_v1_helpers.go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeAPIError writes a structured error response
func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, APIErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
		},
	})
}

// writeValidationError writes a 422 validation error with field details
func writeValidationError(w http.ResponseWriter, details map[string]string) {
	writeJSON(w, http.StatusUnprocessableEntity, APIErrorResponse{
		Error: APIError{
			Code:    "validation_error",
			Message: "Invalid input",
			Details: details,
		},
	})
}

// parseIntParam parses an integer query parameter with a default value
func parseIntParam(r *http.Request, name string, defaultVal, maxVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return defaultVal
	}
	if maxVal > 0 && n > maxVal {
		return maxVal
	}
	return n
}

// wantsJSON checks if the request prefers JSON response
// Checks Accept header and .json suffix
func wantsJSON(r *http.Request) bool {
	// Check for .json suffix in path
	if len(r.URL.Path) > 5 && r.URL.Path[len(r.URL.Path)-5:] == ".json" {
		return true
	}

	accept := r.Header.Get("Accept")
	if accept == "" || accept == "*/*" {
		return true // Default to JSON for API
	}

	// Simple check - could be more sophisticated
	return accept == "application/json" ||
	       len(accept) >= 16 && accept[:16] == "application/json"
}

// wantsPlainText checks if the request prefers plain text
func wantsPlainText(r *http.Request) bool {
	// Check for .txt suffix in path
	if len(r.URL.Path) > 4 && r.URL.Path[len(r.URL.Path)-4:] == ".txt" {
		return true
	}

	accept := r.Header.Get("Accept")
	return accept == "text/plain"
}

// trimFormatSuffix removes .json or .txt suffix from path
func trimFormatSuffix(path string) string {
	if len(path) > 5 && path[len(path)-5:] == ".json" {
		return path[:len(path)-5]
	}
	if len(path) > 4 && path[len(path)-4:] == ".txt" {
		return path[:len(path)-4]
	}
	return path
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run "TestWrite" -v`

Expected: PASS

**Step 5: Add tests for param parsing and content negotiation**

```go
// Add to internal/handler/api_v1_helpers_test.go

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		paramName  string
		defaultVal int
		maxVal     int
		want       int
	}{
		{"empty uses default", "", "limit", 50, 1000, 50},
		{"valid value", "limit=100", "limit", 50, 1000, 100},
		{"exceeds max", "limit=2000", "limit", 50, 1000, 1000},
		{"negative uses default", "limit=-5", "limit", 50, 1000, 50},
		{"invalid uses default", "limit=abc", "limit", 50, 1000, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/?"+tt.query, nil)
			got := parseIntParam(r, tt.paramName, tt.defaultVal, tt.maxVal)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWantsJSON(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		accept string
		want   bool
	}{
		{"json suffix", "/api/v1/links/42.json", "", true},
		{"json accept", "/api/v1/links/42", "application/json", true},
		{"no preference defaults json", "/api/v1/links/42", "", true},
		{"wildcard defaults json", "/api/v1/links/42", "*/*", true},
		{"html accept", "/api/v1/links/42", "text/html", false},
		{"plain text", "/api/v1/links/42", "text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil)
			if tt.accept != "" {
				r.Header.Set("Accept", tt.accept)
			}
			got := wantsJSON(r)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimFormatSuffix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/api/v1/links/42.json", "/api/v1/links/42"},
		{"/api/v1/links/42.txt", "/api/v1/links/42"},
		{"/api/v1/links/42", "/api/v1/links/42"},
		{"/api/v1/links", "/api/v1/links"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := trimFormatSuffix(tt.input)
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
```

**Step 6: Run all helper tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run "TestWrite\|TestParse\|TestWants\|TestTrim" -v`

Expected: PASS

**Step 7: Commit**

```bash
git add internal/handler/api_v1_helpers.go internal/handler/api_v1_helpers_test.go
git commit -m "feat(api): add v1 API response helpers and content negotiation"
```

---

### Task 1.3: Create API v1 Authentication Helper

**Files:**
- Modify: `internal/handler/api_v1_helpers.go`
- Modify: `internal/handler/api_v1_helpers_test.go`

**Step 1: Write test for API key authentication**

```go
// Add to internal/handler/api_v1_helpers_test.go

func TestIsAuthorizedAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		secret    string
		localhost bool
		want      bool
	}{
		{"valid header", "correct-key", "correct-key", false, true},
		{"invalid header", "wrong-key", "correct-key", false, false},
		{"missing header", "", "correct-key", false, false},
		{"no secret configured", "any-key", "", false, false},
		{"localhost always allowed", "", "secret", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("DELETE", "/api/v1/links/42", nil)
			if tt.header != "" {
				r.Header.Set("X-API-Key", tt.header)
			}
			if tt.localhost {
				r.RemoteAddr = "127.0.0.1:12345"
			} else {
				r.RemoteAddr = "203.0.113.1:12345"
			}

			got := isAuthorizedAPIKey(r, tt.secret)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestIsAuthorizedAPIKey -v`

Expected: FAIL with "undefined: isAuthorizedAPIKey"

**Step 3: Implement authentication helper**

```go
// Add to internal/handler/api_v1_helpers.go

import (
	"crypto/subtle"
	"log"
	"strings"
)

// isAuthorizedAPIKey checks if the request has a valid API key
// Uses X-API-Key header only (query param auth removed)
func isAuthorizedAPIKey(r *http.Request, secret string) bool {
	// Always allow localhost requests
	remoteAddr := r.RemoteAddr
	if strings.HasPrefix(remoteAddr, "127.0.0.1") ||
	   strings.HasPrefix(remoteAddr, "localhost") ||
	   strings.HasPrefix(remoteAddr, "[::1]") {
		return true
	}

	// Require configured secret
	if secret == "" {
		log.Printf("API auth failed: no secret configured, remote_addr=%s", remoteAddr)
		return false
	}

	// Check X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		log.Printf("API auth failed: no key provided, remote_addr=%s", remoteAddr)
		return false
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(secret)) != 1 {
		log.Printf("API auth failed: invalid key, remote_addr=%s", remoteAddr)
		return false
	}

	return true
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestIsAuthorizedAPIKey -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/api_v1_helpers.go internal/handler/api_v1_helpers_test.go
git commit -m "feat(api): add X-API-Key authentication helper"
```

---

## Phase 2: Links API

### Task 2.1: Create Links Handler Skeleton

**Files:**
- Create: `internal/handler/api_v1_links.go`
- Test: `internal/handler/api_v1_links_test.go`

**Step 1: Write test for GET /api/v1/links**

```go
// internal/handler/api_v1_links_test.go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

// mockStore implements data.Store for testing
type mockAPIStore struct {
	data.Store
	links  []data.IRCLink
	quotes []data.Quote
}

func (m *mockAPIStore) GetRecentIRCLinks(ctx context.Context, startDays, endDays int) ([]data.IRCLink, error) {
	return m.links, nil
}

func TestAPIv1_ListLinks(t *testing.T) {
	store := &mockAPIStore{
		links: []data.IRCLink{
			{IRCLinkID: 1, User: "alice", Title: "Example", URL: "https://example.com", Timestamp: time.Now()},
			{IRCLinkID: 2, User: "bob", Title: "Test", URL: "https://test.com", Timestamp: time.Now()},
		},
	}
	h := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest("GET", "/api/v1/links", nil)
	w := httptest.NewRecorder()

	h.APIv1LinksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp APILinksResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) != 2 {
		t.Errorf("data length: got %d, want 2", len(resp.Data))
	}
	if resp.Data[0].ID != 1 {
		t.Errorf("first link id: got %d, want 1", resp.Data[0].ID)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_ListLinks -v`

Expected: FAIL with "h.APIv1LinksHandler undefined"

**Step 3: Implement basic handler**

```go
// internal/handler/api_v1_links.go
package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// APIv1LinksHandler handles /api/v1/links routes
func (h *Handler) APIv1LinksHandler(w http.ResponseWriter, r *http.Request) {
	// Remove /api/v1/links prefix and any format suffix
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/links")
	path = trimFormatSuffix(path)

	switch {
	case path == "" || path == "/":
		// Collection operations
		switch r.Method {
		case http.MethodGet:
			h.apiV1ListLinks(w, r)
		case http.MethodPost:
			h.apiV1CreateLink(w, r)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	default:
		// Resource operations: /{id}
		idStr := strings.TrimPrefix(path, "/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "bad_request", "Invalid link ID")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.apiV1GetLink(w, r, id)
		case http.MethodDelete:
			h.apiV1DeleteLink(w, r, id)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	}
}

func (h *Handler) apiV1ListLinks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := parseIntParam(r, "limit", 50, 1000)
	offset := parseIntParam(r, "offset", 0, 0)

	// Get links (using existing store method, we'll need pagination later)
	links, err := h.Store.GetRecentIRCLinks(ctx, 365, 0) // Last year for now
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch links")
		return
	}

	// Apply offset and limit
	total := len(links)
	if offset >= total {
		links = nil
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		links = links[offset:end]
	}

	// Convert to API response format
	data := make([]APILinkResponse, len(links))
	for i, link := range links {
		data[i] = APILinkResponse{
			ID:        link.IRCLinkID,
			URL:       link.URL,
			Title:     link.Title,
			User:      link.User,
			Clicks:    link.Clicks,
			CreatedAt: link.Timestamp,
		}
	}

	writeJSON(w, http.StatusOK, APILinksResponse{
		Data: data,
		Meta: APIMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Placeholder implementations
func (h *Handler) apiV1CreateLink(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, http.StatusNotImplemented, "not_implemented", "Not implemented yet")
}

func (h *Handler) apiV1GetLink(w http.ResponseWriter, r *http.Request, id int) {
	writeAPIError(w, http.StatusNotImplemented, "not_implemented", "Not implemented yet")
}

func (h *Handler) apiV1DeleteLink(w http.ResponseWriter, r *http.Request, id int) {
	writeAPIError(w, http.StatusNotImplemented, "not_implemented", "Not implemented yet")
}
```

**Step 4: Update test with proper imports and mock**

```go
// Update internal/handler/api_v1_links_test.go - add context import
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_ListLinks -v`

Expected: PASS

**Step 6: Commit**

```bash
git add internal/handler/api_v1_links.go internal/handler/api_v1_links_test.go
git commit -m "feat(api): add GET /api/v1/links endpoint"
```

---

### Task 2.2: Implement GET /api/v1/links/{id}

**Step 1: Add test**

```go
// Add to internal/handler/api_v1_links_test.go

func (m *mockAPIStore) GetIRCLinkByID(ctx context.Context, id int) (*data.IRCLink, error) {
	for _, link := range m.links {
		if link.IRCLinkID == id {
			return &link, nil
		}
	}
	return nil, nil
}

func TestAPIv1_GetLink(t *testing.T) {
	store := &mockAPIStore{
		links: []data.IRCLink{
			{IRCLinkID: 42, User: "alice", Title: "Example", URL: "https://example.com", Clicks: 5, Timestamp: time.Now()},
		},
	}
	h := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	t.Run("existing link", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/links/42", nil)
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
		}

		var resp APILinkResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.ID != 42 {
			t.Errorf("id: got %d, want 42", resp.ID)
		}
		if resp.User != "alice" {
			t.Errorf("user: got %s, want alice", resp.User)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/links/999", nil)
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_GetLink -v`

Expected: FAIL (returns 501 Not Implemented)

**Step 3: Implement**

```go
// Replace apiV1GetLink in internal/handler/api_v1_links.go

func (h *Handler) apiV1GetLink(w http.ResponseWriter, r *http.Request, id int) {
	ctx := r.Context()

	link, err := h.Store.GetIRCLinkByID(ctx, id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch link")
		return
	}
	if link == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
		return
	}

	// Check content negotiation
	if wantsPlainText(r) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(link.Title + " - " + link.URL))
		return
	}

	writeJSON(w, http.StatusOK, APILinkResponse{
		ID:        link.IRCLinkID,
		URL:       link.URL,
		Title:     link.Title,
		User:      link.User,
		Clicks:    link.Clicks,
		CreatedAt: link.Timestamp,
	})
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_GetLink -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/api_v1_links.go internal/handler/api_v1_links_test.go
git commit -m "feat(api): add GET /api/v1/links/{id} endpoint"
```

---

### Task 2.3: Implement POST /api/v1/links

**Step 1: Add test**

```go
// Add to internal/handler/api_v1_links_test.go

import "bytes"

func (m *mockAPIStore) InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error) {
	return 123, nil
}

func (m *mockAPIStore) GetIRCLinksByURL(ctx context.Context, url string) ([]data.IRCLink, error) {
	return nil, nil // No duplicates
}

func TestAPIv1_CreateLink(t *testing.T) {
	store := &mockAPIStore{}
	h := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	t.Run("valid link", func(t *testing.T) {
		body := bytes.NewBufferString(`{"url": "https://example.com", "user": "alice"}`)
		req := httptest.NewRequest("POST", "/api/v1/links", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusCreated)
		}

		var resp APILinkCreateResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.ID != 123 {
			t.Errorf("id: got %d, want 123", resp.ID)
		}
		if resp.IsDuplicate {
			t.Error("should not be duplicate")
		}
	})

	t.Run("missing url", func(t *testing.T) {
		body := bytes.NewBufferString(`{"user": "alice"}`)
		req := httptest.NewRequest("POST", "/api/v1/links", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
		}
	})

	t.Run("invalid url scheme", func(t *testing.T) {
		body := bytes.NewBufferString(`{"url": "javascript:alert(1)", "user": "alice"}`)
		req := httptest.NewRequest("POST", "/api/v1/links", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusUnprocessableEntity)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_CreateLink -v`

Expected: FAIL

**Step 3: Implement**

```go
// Add to internal/handler/api_v1_links.go

import (
	"encoding/json"
	"strings"
)

// APILinkCreateRequest is the JSON body for creating a link
type APILinkCreateRequest struct {
	URL  string `json:"url"`
	User string `json:"user"`
}

func (h *Handler) apiV1CreateLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req APILinkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}

	// Validate required fields
	errors := make(map[string]string)
	if req.URL == "" {
		errors["url"] = "is required"
	} else if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		errors["url"] = "must use http or https scheme"
	}
	if req.User == "" {
		errors["user"] = "is required"
	}

	if len(errors) > 0 {
		writeValidationError(w, errors)
		return
	}

	// Check for duplicates
	existingLinks, err := h.Store.GetIRCLinksByURL(ctx, req.URL)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to check duplicates")
		return
	}

	isDuplicate := len(existingLinks) > 0

	// Fetch title (simplified - in production use the existing title fetching logic)
	title := req.URL // Placeholder - actual implementation fetches page title

	// Insert the link
	linkID, err := h.Store.InsertIRCLink(ctx, req.User, title, req.URL, "")
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to create link")
		return
	}

	// Build response
	resp := APILinkCreateResponse{
		APILinkResponse: APILinkResponse{
			ID:        linkID,
			URL:       req.URL,
			Title:     title,
			User:      req.User,
			Clicks:    0,
			CreatedAt: time.Now(),
		},
		IsDuplicate: isDuplicate,
	}

	// Add previous submissions if duplicate
	if isDuplicate {
		resp.PreviousSubmissions = make([]APIPreviousSubmission, len(existingLinks))
		for i, link := range existingLinks {
			resp.PreviousSubmissions[i] = APIPreviousSubmission{
				ID:        link.IRCLinkID,
				User:      link.User,
				CreatedAt: link.Timestamp,
				Title:     link.Title,
			}
		}
	}

	// Check content negotiation for plain text (IRC bot)
	if wantsPlainText(r) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		if isDuplicate && len(existingLinks) > 0 {
			fmt.Fprintf(w, "%d (duplicate, previously posted by %s)", linkID, existingLinks[0].User)
		} else {
			fmt.Fprintf(w, "%d", linkID)
		}
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}
```

**Step 4: Add import for time and fmt**

```go
// Update imports in internal/handler/api_v1_links.go
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_CreateLink -v`

Expected: PASS

**Step 6: Commit**

```bash
git add internal/handler/api_v1_links.go internal/handler/api_v1_links_test.go
git commit -m "feat(api): add POST /api/v1/links endpoint with duplicate detection"
```

---

### Task 2.4: Implement DELETE /api/v1/links/{id}

**Step 1: Add test**

```go
// Add to internal/handler/api_v1_links_test.go

func (m *mockAPIStore) DeleteIRCLink(ctx context.Context, id int) error {
	return nil
}

func TestAPIv1_DeleteLink(t *testing.T) {
	store := &mockAPIStore{
		links: []data.IRCLink{
			{IRCLinkID: 42, User: "alice", Title: "Example", URL: "https://example.com"},
		},
	}
	h := &Handler{
		Store:  store,
		Config: &config.Config{AdminSecret: "test-secret"},
	}

	t.Run("authorized delete", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/links/42", nil)
		req.Header.Set("X-API-Key", "test-secret")
		req.RemoteAddr = "203.0.113.1:12345" // Non-localhost
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusNoContent)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/links/42", nil)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusForbidden)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/links/999", nil)
		req.Header.Set("X-API-Key", "test-secret")
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		h.APIv1LinksHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_DeleteLink -v`

Expected: FAIL

**Step 3: Implement**

```go
// Replace apiV1DeleteLink in internal/handler/api_v1_links.go

func (h *Handler) apiV1DeleteLink(w http.ResponseWriter, r *http.Request, id int) {
	// Check authorization
	if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Invalid or missing API key")
		return
	}

	ctx := r.Context()

	// Check if link exists
	link, err := h.Store.GetIRCLinkByID(ctx, id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch link")
		return
	}
	if link == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
		return
	}

	// Delete the link
	if err := h.Store.DeleteIRCLink(ctx, id); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to delete link")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble-dark/.worktrees/api-v1-redesign && go test ./internal/handler/... -run TestAPIv1_DeleteLink -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/api_v1_links.go internal/handler/api_v1_links_test.go
git commit -m "feat(api): add DELETE /api/v1/links/{id} endpoint"
```

---

## Phase 3: Quotes API

### Task 3.1: Create Quotes Handler

**Files:**
- Create: `internal/handler/api_v1_quotes.go`
- Test: `internal/handler/api_v1_quotes_test.go`

Follow the same TDD pattern as Links API:

1. GET /api/v1/quotes (list)
2. POST /api/v1/quotes (create)
3. GET /api/v1/quotes/{id} (get single)
4. DELETE /api/v1/quotes/{id} (delete)

**Implementation mirrors Task 2.1-2.4 with Quote types instead of Link types.**

---

## Phase 4: Stats API

### Task 4.1: Implement GET /api/v1/stats

**Files:**
- Create: `internal/handler/api_v1_stats.go`
- Test: `internal/handler/api_v1_stats_test.go`

Returns site-wide stats and leaderboard.

### Task 4.2: Implement GET /api/v1/users/{user}/stats

Per-user statistics endpoint.

---

## Phase 5: Search API

### Task 5.1: Implement GET /api/v1/search

**Files:**
- Create: `internal/handler/api_v1_search.go`
- Test: `internal/handler/api_v1_search_test.go`

Returns separate arrays for links and quotes.

---

## Phase 6: Cache API

### Task 6.1: Implement DELETE /api/v1/cache

**Files:**
- Create: `internal/handler/api_v1_cache.go`
- Test: `internal/handler/api_v1_cache_test.go`

Clear all cache or specific URL.

---

## Phase 7: Kittens API

### Task 7.1: Implement GET /api/v1/kittens/daily

### Task 7.2: Implement PUT /api/v1/kittens/daily

### Task 7.3: Implement DELETE /api/v1/kittens/daily

**Files:**
- Create: `internal/handler/api_v1_kittens.go`
- Test: `internal/handler/api_v1_kittens_test.go`

---

## Phase 8: Redirect Shortlink

### Task 8.1: Implement GET /go/{id}

**Files:**
- Create: `internal/handler/redirect.go`
- Test: `internal/handler/redirect_test.go`

Simple redirect endpoint separate from API.

---

## Phase 9: Route Registration

### Task 9.1: Register All v1 Routes

**Files:**
- Modify: `cmd/tumble/main.go`

Add all `/api/v1/` routes to the mux.

---

## Phase 10: Update OpenAPI Spec

### Task 10.1: Update openapi.json

**Files:**
- Modify: `internal/assets/openapi.json`

Replace with new v1 API specification matching the design document.

---

## Phase 11: Integration Testing

### Task 11.1: Add API Integration Tests

**Files:**
- Create: `tests/api_v1_test.go`

End-to-end tests against running server.

---

## Summary

| Phase | Tasks | Est. Steps |
|-------|-------|------------|
| 1. Foundation | 3 | 21 |
| 2. Links API | 4 | 28 |
| 3. Quotes API | 4 | 28 |
| 4. Stats API | 2 | 14 |
| 5. Search API | 1 | 7 |
| 6. Cache API | 1 | 7 |
| 7. Kittens API | 3 | 21 |
| 8. Redirect | 1 | 7 |
| 9. Routes | 1 | 5 |
| 10. OpenAPI | 1 | 5 |
| 11. Integration | 1 | 7 |
| **Total** | **22** | **~150** |
