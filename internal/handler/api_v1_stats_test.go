package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tumble/internal/config"
	"tumble/internal/data"
)

// mockStatsStore is a mock implementation of data.Store for testing stats API handlers.
type mockStatsStore struct {
	data.Store
	userStats   []data.UserStat
	userStatsFn func(sortBy string, limit int, offset int) ([]data.UserStat, error)
	links       []data.IRCLink
	quotes      []data.Quote
	err         error
}

func (m *mockStatsStore) GetUserStats(ctx context.Context, sortBy string, limit int, offset int) ([]data.UserStat, error) {
	if m.userStatsFn != nil {
		return m.userStatsFn(sortBy, limit, offset)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.userStats, nil
}

func (m *mockStatsStore) GetRecentIRCLinks(ctx context.Context, days int, offsetDays int, filter data.ClientFilter) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.links, nil
}

func (m *mockStatsStore) GetRecentQuotes(ctx context.Context, days int, offsetDays int, filter data.ClientFilter) ([]data.Quote, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.quotes, nil
}

func TestAPIv1_Stats(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		userStats      []data.UserStat
		links          []data.IRCLink
		quotes         []data.Quote
		storeErr       error
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "returns stats with empty data",
			method:         http.MethodGet,
			path:           "/api/v1/stats",
			userStats:      []data.UserStat{},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Site.TotalLinks != 0 {
					t.Errorf("expected total_links 0, got %d", resp.Site.TotalLinks)
				}
				if resp.Site.TotalQuotes != 0 {
					t.Errorf("expected total_quotes 0, got %d", resp.Site.TotalQuotes)
				}
				if resp.Site.TotalUsers != 0 {
					t.Errorf("expected total_users 0, got %d", resp.Site.TotalUsers)
				}
				if len(resp.Leaderboard) != 0 {
					t.Errorf("expected 0 leaderboard entries, got %d", len(resp.Leaderboard))
				}
				if resp.Meta.Total != 0 {
					t.Errorf("expected meta total 0, got %d", resp.Meta.Total)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected meta limit 50, got %d", resp.Meta.Limit)
				}
				if resp.Meta.Offset != 0 {
					t.Errorf("expected meta offset 0, got %d", resp.Meta.Offset)
				}
			},
		},
		{
			name:   "returns stats with proper structure",
			method: http.MethodGet,
			path:   "/api/v1/stats",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
				{User: "bob", LinkCount: 300, QuoteCount: 80},
			},
			links:          make([]data.IRCLink, 15000),
			quotes:         make([]data.Quote, 3200),
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Site.TotalLinks != 15000 {
					t.Errorf("expected total_links 15000, got %d", resp.Site.TotalLinks)
				}
				if resp.Site.TotalQuotes != 3200 {
					t.Errorf("expected total_quotes 3200, got %d", resp.Site.TotalQuotes)
				}
				if resp.Site.TotalUsers != 2 {
					t.Errorf("expected total_users 2, got %d", resp.Site.TotalUsers)
				}
				if len(resp.Leaderboard) != 2 {
					t.Fatalf("expected 2 leaderboard entries, got %d", len(resp.Leaderboard))
				}
				if resp.Leaderboard[0].User != "alice" {
					t.Errorf("expected first user alice, got %s", resp.Leaderboard[0].User)
				}
				if resp.Leaderboard[0].LinkCount != 500 {
					t.Errorf("expected alice link_count 500, got %d", resp.Leaderboard[0].LinkCount)
				}
				if resp.Leaderboard[0].QuoteCount != 120 {
					t.Errorf("expected alice quote_count 120, got %d", resp.Leaderboard[0].QuoteCount)
				}
				if resp.Meta.Total != 2 {
					t.Errorf("expected meta total 2, got %d", resp.Meta.Total)
				}
			},
		},
		{
			name:   "respects limit parameter",
			method: http.MethodGet,
			path:   "/api/v1/stats?limit=1",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
				{User: "bob", LinkCount: 300, QuoteCount: 80},
			},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Leaderboard) != 1 {
					t.Errorf("expected 1 leaderboard entry, got %d", len(resp.Leaderboard))
				}
				if resp.Meta.Limit != 1 {
					t.Errorf("expected meta limit 1, got %d", resp.Meta.Limit)
				}
				// Total should still be 2 (all users)
				if resp.Meta.Total != 2 {
					t.Errorf("expected meta total 2, got %d", resp.Meta.Total)
				}
			},
		},
		{
			name:   "respects offset parameter",
			method: http.MethodGet,
			path:   "/api/v1/stats?offset=1",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
				{User: "bob", LinkCount: 300, QuoteCount: 80},
			},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Leaderboard) != 1 {
					t.Errorf("expected 1 leaderboard entry, got %d", len(resp.Leaderboard))
				}
				if resp.Leaderboard[0].User != "bob" {
					t.Errorf("expected first user bob (after offset), got %s", resp.Leaderboard[0].User)
				}
				if resp.Meta.Offset != 1 {
					t.Errorf("expected meta offset 1, got %d", resp.Meta.Offset)
				}
			},
		},
		{
			name:           "limit capped at 1000",
			method:         http.MethodGet,
			path:           "/api/v1/stats?limit=5000",
			userStats:      []data.UserStat{},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Meta.Limit != 1000 {
					t.Errorf("expected limit capped at 1000, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:           "invalid limit uses default",
			method:         http.MethodGet,
			path:           "/api/v1/stats?limit=abc",
			userStats:      []data.UserStat{},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected default limit 50, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:   "offset beyond range returns empty leaderboard",
			method: http.MethodGet,
			path:   "/api/v1/stats?offset=100",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
			},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Leaderboard) != 0 {
					t.Errorf("expected 0 leaderboard entries, got %d", len(resp.Leaderboard))
				}
				if resp.Meta.Total != 1 {
					t.Errorf("expected total 1, got %d", resp.Meta.Total)
				}
			},
		},
		{
			name:           "method not allowed for POST",
			method:         http.MethodPost,
			path:           "/api/v1/stats",
			userStats:      []data.UserStat{},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "method_not_allowed" {
					t.Errorf("expected code method_not_allowed, got %s", resp.Error.Code)
				}
			},
		},
		{
			name:   "works with .json suffix",
			method: http.MethodGet,
			path:   "/api/v1/stats.json",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
			},
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIStatsResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Leaderboard) != 1 {
					t.Errorf("expected 1 leaderboard entry, got %d", len(resp.Leaderboard))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStatsStore{
				userStats: tt.userStats,
				links:     tt.links,
				quotes:    tt.quotes,
				err:       tt.storeErr,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1StatsHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_Stats_StoreError(t *testing.T) {
	store := &mockStatsStore{
		userStats: nil,
		err:       context.DeadlineExceeded,
	}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.APIv1StatsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != "internal_error" {
		t.Errorf("expected code internal_error, got %s", resp.Error.Code)
	}
}

func TestAPIv1_UserStats(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		userStats      []data.UserStat
		userStatsFn    func(sortBy string, limit int, offset int) ([]data.UserStat, error)
		storeErr       error
		acceptHeader   string
		expectedStatus int
		expectedType   string
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "returns existing user stats",
			path: "/api/v1/users/alice/stats",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
				{User: "bob", LinkCount: 300, QuoteCount: 80},
			},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIUserStats
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.User != "alice" {
					t.Errorf("expected user alice, got %s", resp.User)
				}
				if resp.LinkCount != 500 {
					t.Errorf("expected link_count 500, got %d", resp.LinkCount)
				}
				if resp.QuoteCount != 120 {
					t.Errorf("expected quote_count 120, got %d", resp.QuoteCount)
				}
			},
		},
		{
			name:           "returns 404 for non-existent user",
			path:           "/api/v1/users/unknownuser/stats",
			userStats:      []data.UserStat{},
			expectedStatus: http.StatusNotFound,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "not_found" {
					t.Errorf("expected code not_found, got %s", resp.Error.Code)
				}
				if resp.Error.Message != "User not found" {
					t.Errorf("expected message 'User not found', got %s", resp.Error.Message)
				}
			},
		},
		{
			name:           "returns 500 on store error",
			path:           "/api/v1/users/alice/stats",
			userStats:      nil,
			storeErr:       context.DeadlineExceeded,
			expectedStatus: http.StatusInternalServerError,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "internal_error" {
					t.Errorf("expected code internal_error, got %s", resp.Error.Code)
				}
			},
		},
		{
			name: "returns plain text with Accept header",
			path: "/api/v1/users/alice/stats",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
			},
			acceptHeader:   "text/plain",
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "alice: 500 links, 120 quotes"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name: "returns plain text with .txt suffix",
			path: "/api/v1/users/alice/stats.txt",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
			},
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "alice: 500 links, 120 quotes"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name: "works with .json suffix",
			path: "/api/v1/users/alice/stats.json",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
			},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIUserStats
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.User != "alice" {
					t.Errorf("expected user alice, got %s", resp.User)
				}
			},
		},
		{
			name: "user not in results returns 404",
			path: "/api/v1/users/charlie/stats",
			userStats: []data.UserStat{
				{User: "alice", LinkCount: 500, QuoteCount: 120},
				{User: "bob", LinkCount: 300, QuoteCount: 80},
			},
			expectedStatus: http.StatusNotFound,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "not_found" {
					t.Errorf("expected code not_found, got %s", resp.Error.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStatsStore{
				userStats:   tt.userStats,
				userStatsFn: tt.userStatsFn,
				err:         tt.storeErr,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345"
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.APIv1UsersHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("expected Content-Type %s, got %s", tt.expectedType, contentType)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_UsersHandler_MethodNotAllowed(t *testing.T) {
	store := &mockStatsStore{}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/alice/stats", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.APIv1UsersHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}

	var resp APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != "method_not_allowed" {
		t.Errorf("expected code method_not_allowed, got %s", resp.Error.Code)
	}
}

func TestAPIv1_UsersHandler_InvalidPath(t *testing.T) {
	store := &mockStatsStore{}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	// Path that doesn't match expected pattern
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/alice", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.APIv1UsersHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != "not_found" {
		t.Errorf("expected code not_found, got %s", resp.Error.Code)
	}
}
