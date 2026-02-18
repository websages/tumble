package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

// mockQuoteStore is a mock implementation of data.Store for testing quote API handlers.
type mockQuoteStore struct {
	data.Store
	quotes            []data.Quote
	recentQuotesFn    func(filter data.ClientFilter) ([]data.Quote, error)
	quoteByID         *data.Quote
	quoteByIDFn       func(id int) (*data.Quote, error)
	insertedQuoteID   int
	insertQuoteFn     func(quote *data.Quote) (int, error)
	lastInsertedQuote *data.Quote
	deleteQuoteFn     func(id int) error
	err               error
}

func (m *mockQuoteStore) GetRecentQuotes(ctx context.Context, days int, offsetDays int, filter data.ClientFilter) ([]data.Quote, error) {
	if m.recentQuotesFn != nil {
		return m.recentQuotesFn(filter)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.quotes, nil
}

func (m *mockQuoteStore) GetQuoteByID(ctx context.Context, id int) (*data.Quote, error) {
	if m.quoteByIDFn != nil {
		return m.quoteByIDFn(id)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.quoteByID, nil
}

func (m *mockQuoteStore) InsertQuote(ctx context.Context, quote *data.Quote) (int, error) {
	m.lastInsertedQuote = quote
	if m.insertQuoteFn != nil {
		return m.insertQuoteFn(quote)
	}
	if m.err != nil {
		return 0, m.err
	}
	return m.insertedQuoteID, nil
}

func (m *mockQuoteStore) DeleteQuote(ctx context.Context, id int) error {
	if m.deleteQuoteFn != nil {
		return m.deleteQuoteFn(id)
	}
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockQuoteStore) GetTagsByResource(ctx context.Context, resourceType string, resourceID int) ([]data.Tag, error) {
	return nil, nil
}

func (m *mockQuoteStore) CreateTag(ctx context.Context, tag data.Tag) (*data.Tag, error) {
	return &tag, nil
}

func (m *mockQuoteStore) GetTagByID(ctx context.Context, id int) (*data.Tag, error) {
	return nil, nil
}

func (m *mockQuoteStore) DeleteTag(ctx context.Context, id int) error {
	return nil
}

func (m *mockQuoteStore) DeleteTagsByResource(ctx context.Context, resourceType string, resourceID int) error {
	return nil
}

func TestAPIv1_ListQuotes(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		method         string
		path           string
		quotes         []data.Quote
		storeErr       error
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "returns empty list",
			method:         http.MethodGet,
			path:           "/api/v1/quotes",
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 0 {
					t.Errorf("expected 0 quotes, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 0 {
					t.Errorf("expected total 0, got %d", resp.Meta.Total)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected limit 50, got %d", resp.Meta.Limit)
				}
				if resp.Meta.Offset != 0 {
					t.Errorf("expected offset 0, got %d", resp.Meta.Offset)
				}
			},
		},
		{
			name:   "returns quotes with proper structure",
			method: http.MethodGet,
			path:   "/api/v1/quotes",
			quotes: []data.Quote{
				{
					ID:        1,
					Timestamp: now,
					Quote:     "To be or not to be",
					Author:    "Shakespeare",
					Poster:    "testuser",
				},
				{
					ID:        2,
					Timestamp: now.Add(-time.Hour),
					Quote:     "I think therefore I am",
					Author:    "Descartes",
					Poster:    "anotheruser",
				},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 2 {
					t.Fatalf("expected 2 quotes, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 2 {
					t.Errorf("expected total 2, got %d", resp.Meta.Total)
				}

				// Check first quote structure
				quote := resp.Data[0]
				if quote.ID != 1 {
					t.Errorf("expected ID 1, got %d", quote.ID)
				}
				if quote.Quote != "To be or not to be" {
					t.Errorf("expected quote 'To be or not to be', got %s", quote.Quote)
				}
				if quote.Author != "Shakespeare" {
					t.Errorf("expected author Shakespeare, got %s", quote.Author)
				}
				if quote.Poster != "testuser" {
					t.Errorf("expected poster testuser, got %s", quote.Poster)
				}
			},
		},
		{
			name:   "respects limit parameter",
			method: http.MethodGet,
			path:   "/api/v1/quotes?limit=1",
			quotes: []data.Quote{
				{ID: 1, Timestamp: now, Quote: "Quote 1", Author: "Author 1"},
				{ID: 2, Timestamp: now, Quote: "Quote 2", Author: "Author 2"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 2 {
					t.Errorf("expected total 2, got %d", resp.Meta.Total)
				}
				if resp.Meta.Limit != 1 {
					t.Errorf("expected limit 1, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:   "respects offset parameter",
			method: http.MethodGet,
			path:   "/api/v1/quotes?offset=1",
			quotes: []data.Quote{
				{ID: 1, Timestamp: now, Quote: "Quote 1", Author: "Author 1"},
				{ID: 2, Timestamp: now, Quote: "Quote 2", Author: "Author 2"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Data))
				}
				if resp.Meta.Offset != 1 {
					t.Errorf("expected offset 1, got %d", resp.Meta.Offset)
				}
				// First item should be the second quote
				if resp.Data[0].ID != 2 {
					t.Errorf("expected quote ID 2, got %d", resp.Data[0].ID)
				}
			},
		},
		{
			name:           "limit capped at 1000",
			method:         http.MethodGet,
			path:           "/api/v1/quotes?limit=5000",
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
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
			path:           "/api/v1/quotes?limit=abc",
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected default limit 50, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:   "offset beyond range returns empty",
			method: http.MethodGet,
			path:   "/api/v1/quotes?offset=100",
			quotes: []data.Quote{
				{ID: 1, Timestamp: now, Quote: "Quote 1", Author: "Author 1"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 0 {
					t.Errorf("expected 0 quotes, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 1 {
					t.Errorf("expected total 1, got %d", resp.Meta.Total)
				}
			},
		},
		{
			name:           "method not allowed for PUT",
			method:         http.MethodPut,
			path:           "/api/v1/quotes",
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
			path:   "/api/v1/quotes.json",
			quotes: []data.Quote{
				{ID: 1, Timestamp: now, Quote: "Quote 1", Author: "Author 1"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuotesResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockQuoteStore{quotes: tt.quotes, err: tt.storeErr}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345" // Localhost for auth bypass
			w := httptest.NewRecorder()

			handler.APIv1QuotesHandler(w, req)

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

func TestAPIv1_ListQuotes_StoreError(t *testing.T) {
	store := &mockQuoteStore{
		quotes: nil,
		err:    context.DeadlineExceeded,
	}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quotes", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.APIv1QuotesHandler(w, req)

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

func TestAPIv1_GetQuote(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		path           string
		quoteByID      *data.Quote
		storeErr       error
		acceptHeader   string
		expectedStatus int
		expectedType   string
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "returns existing quote",
			path: "/api/v1/quotes/1",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
				Poster:    "testuser",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuoteResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 1 {
					t.Errorf("expected ID 1, got %d", resp.ID)
				}
				if resp.Quote != "To be or not to be" {
					t.Errorf("expected quote 'To be or not to be', got %s", resp.Quote)
				}
				if resp.Author != "Shakespeare" {
					t.Errorf("expected author Shakespeare, got %s", resp.Author)
				}
				if resp.Poster != "testuser" {
					t.Errorf("expected poster testuser, got %s", resp.Poster)
				}
			},
		},
		{
			name:           "returns 404 for non-existent quote",
			path:           "/api/v1/quotes/999",
			quoteByID:      nil,
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
				if resp.Error.Message != "Quote not found" {
					t.Errorf("expected message 'Quote not found', got %s", resp.Error.Message)
				}
			},
		},
		{
			name:           "returns 500 on store error",
			path:           "/api/v1/quotes/1",
			quoteByID:      nil,
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
			path: "/api/v1/quotes/1",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
				Poster:    "testuser",
			},
			acceptHeader:   "text/plain",
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "\"To be or not to be\" - Shakespeare"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name: "returns plain text with .txt suffix",
			path: "/api/v1/quotes/1.txt",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
				Poster:    "testuser",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "\"To be or not to be\" - Shakespeare"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name: "plain text with no author",
			path: "/api/v1/quotes/1.txt",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "Anonymous wisdom",
				Author:    "",
				Poster:    "testuser",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "\"Anonymous wisdom\""
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name:           "returns 400 for invalid ID",
			path:           "/api/v1/quotes/abc",
			expectedStatus: http.StatusBadRequest,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "invalid_id" {
					t.Errorf("expected code invalid_id, got %s", resp.Error.Code)
				}
			},
		},
		{
			name: "works with .json suffix",
			path: "/api/v1/quotes/1.json",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
				Poster:    "testuser",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuoteResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 1 {
					t.Errorf("expected ID 1, got %d", resp.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockQuoteStore{quoteByID: tt.quoteByID, err: tt.storeErr}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345" // Localhost for auth bypass
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.APIv1QuotesHandler(w, req)

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

func TestAPIv1_CreateQuote(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		insertedID     int
		storeErr       error
		acceptHeader   string
		expectedStatus int
		expectedType   string
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "valid quote with all fields returns 201",
			body:           `{"quote":"To be or not to be","author":"Shakespeare","poster":"testuser"}`,
			insertedID:     42,
			expectedStatus: http.StatusCreated,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuoteResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 42 {
					t.Errorf("expected ID 42, got %d", resp.ID)
				}
				if resp.Quote != "To be or not to be" {
					t.Errorf("expected quote 'To be or not to be', got %s", resp.Quote)
				}
				if resp.Author != "Shakespeare" {
					t.Errorf("expected author Shakespeare, got %s", resp.Author)
				}
				if resp.Poster != "testuser" {
					t.Errorf("expected poster testuser, got %s", resp.Poster)
				}
			},
		},
		{
			name:           "valid quote with only required field returns 201",
			body:           `{"quote":"Anonymous wisdom"}`,
			insertedID:     43,
			expectedStatus: http.StatusCreated,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIQuoteResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 43 {
					t.Errorf("expected ID 43, got %d", resp.ID)
				}
				if resp.Quote != "Anonymous wisdom" {
					t.Errorf("expected quote 'Anonymous wisdom', got %s", resp.Quote)
				}
				if resp.Author != "" {
					t.Errorf("expected empty author, got %s", resp.Author)
				}
				if resp.Poster != "" {
					t.Errorf("expected empty poster, got %s", resp.Poster)
				}
			},
		},
		{
			name:           "missing quote returns 422",
			body:           `{"author":"Shakespeare"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp ValidationErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "validation_error" {
					t.Errorf("expected code validation_error, got %s", resp.Error.Code)
				}
				if resp.Error.Details["quote"] == "" {
					t.Errorf("expected quote validation error")
				}
			},
		},
		{
			name:           "empty quote returns 422",
			body:           `{"quote":"","author":"Shakespeare"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp ValidationErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "validation_error" {
					t.Errorf("expected code validation_error, got %s", resp.Error.Code)
				}
				if resp.Error.Details["quote"] == "" {
					t.Errorf("expected quote validation error")
				}
			},
		},
		{
			name:           "invalid JSON returns 400",
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "invalid_request" {
					t.Errorf("expected code invalid_request, got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "plain text response for Accept: text/plain",
			body:           `{"quote":"To be or not to be","author":"Shakespeare","poster":"testuser"}`,
			insertedID:     42,
			acceptHeader:   "text/plain",
			expectedStatus: http.StatusCreated,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "Created quote 42: \"To be or not to be\" - Shakespeare"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name:           "plain text response for quote without author",
			body:           `{"quote":"Anonymous wisdom"}`,
			insertedID:     43,
			acceptHeader:   "text/plain",
			expectedStatus: http.StatusCreated,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "Created quote 43: \"Anonymous wisdom\""
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockQuoteStore{
				insertedQuoteID: tt.insertedID,
				err:             tt.storeErr,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(tt.body))
			req.RemoteAddr = "127.0.0.1:12345" // Localhost for auth bypass
			req.Header.Set("Content-Type", "application/json")
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.APIv1QuotesHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
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

func TestAPIv1_CreateQuote_StoreError(t *testing.T) {
	store := &mockQuoteStore{
		insertQuoteFn: func(quote *data.Quote) (int, error) {
			return 0, context.DeadlineExceeded
		},
	}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(`{"quote":"Test quote"}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.APIv1QuotesHandler(w, req)

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

func TestAPIv1_DeleteQuote(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		path           string
		quoteByID      *data.Quote
		quoteByIDFn    func(id int) (*data.Quote, error)
		deleteQuoteFn  func(id int) error
		remoteAddr     string
		apiKey         string
		adminSecret    string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "authorized delete returns 204",
			path: "/api/v1/quotes/1",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
			},
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusNoContent,
			checkBody: func(t *testing.T, body []byte) {
				if len(body) != 0 {
					t.Errorf("expected empty body, got %q", string(body))
				}
			},
		},
		{
			name:           "unauthorized (no header) returns 403",
			path:           "/api/v1/quotes/1",
			quoteByID:      &data.Quote{ID: 1},
			remoteAddr:     "192.168.1.100:12345",
			adminSecret:    "secret123",
			expectedStatus: http.StatusForbidden,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "forbidden" {
					t.Errorf("expected code forbidden, got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "unauthorized (wrong key) returns 403",
			path:           "/api/v1/quotes/1",
			quoteByID:      &data.Quote{ID: 1},
			remoteAddr:     "192.168.1.100:12345",
			apiKey:         "wrongkey",
			adminSecret:    "secret123",
			expectedStatus: http.StatusForbidden,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "forbidden" {
					t.Errorf("expected code forbidden, got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "delete non-existent quote returns 404",
			path:           "/api/v1/quotes/999",
			quoteByID:      nil,
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusNotFound,
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
		{
			name: "localhost always authorized (no key needed)",
			path: "/api/v1/quotes/1",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
			},
			remoteAddr:     "127.0.0.1:12345",
			adminSecret:    "secret123",
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "valid API key authorizes delete",
			path: "/api/v1/quotes/1",
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
			},
			remoteAddr:     "192.168.1.100:12345",
			apiKey:         "secret123",
			adminSecret:    "secret123",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid ID returns 400",
			path:           "/api/v1/quotes/abc",
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "invalid_id" {
					t.Errorf("expected code invalid_id, got %s", resp.Error.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockQuoteStore{
				quoteByID:     tt.quoteByID,
				quoteByIDFn:   tt.quoteByIDFn,
				deleteQuoteFn: tt.deleteQuoteFn,
			}
			handler := &Handler{
				Store: store,
				Config: &config.Config{
					AdminSecret: tt.adminSecret,
				},
			}

			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			w := httptest.NewRecorder()

			handler.APIv1QuotesHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_DeleteQuote_StoreError(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		quoteByIDFn    func(id int) (*data.Quote, error)
		deleteQuoteFn  func(id int) error
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "GetQuoteByID error returns 500",
			quoteByIDFn: func(id int) (*data.Quote, error) {
				return nil, context.DeadlineExceeded
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
		},
		{
			name: "DeleteQuote error returns 500",
			quoteByIDFn: func(id int) (*data.Quote, error) {
				return &data.Quote{
					ID:        1,
					Timestamp: now,
					Quote:     "To be or not to be",
					Author:    "Shakespeare",
				}, nil
			},
			deleteQuoteFn: func(id int) error {
				return context.DeadlineExceeded
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockQuoteStore{
				quoteByIDFn:   tt.quoteByIDFn,
				deleteQuoteFn: tt.deleteQuoteFn,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/quotes/1", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1QuotesHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp APIErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if resp.Error.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, resp.Error.Code)
			}
		})
	}
}

func TestAPIv1_CreateQuote_ClientFields(t *testing.T) {
	irc := "irc"
	freenode := "freenode"
	channel := "#general"
	userID := "U12345"
	userName := "alice"

	t.Run("client fields passed through to InsertQuote", func(t *testing.T) {
		store := &mockQuoteStore{insertedQuoteID: 50}
		handler := &Handler{
			Store:  store,
			Config: &config.Config{},
		}

		body := `{"quote":"To be or not to be","author":"Shakespeare","poster":"alice","client_type":"irc","client_network":"freenode","client_channel":"#general","client_user_id":"U12345","client_user_name":"alice"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.APIv1QuotesHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify client fields were passed to InsertQuote
		inserted := store.lastInsertedQuote
		if inserted == nil {
			t.Fatal("expected InsertQuote to be called")
		}
		if inserted.ClientType == nil || *inserted.ClientType != irc {
			t.Errorf("expected client_type=%q, got %v", irc, inserted.ClientType)
		}
		if inserted.ClientNetwork == nil || *inserted.ClientNetwork != freenode {
			t.Errorf("expected client_network=%q, got %v", freenode, inserted.ClientNetwork)
		}
		if inserted.ClientChannel == nil || *inserted.ClientChannel != channel {
			t.Errorf("expected client_channel=%q, got %v", channel, inserted.ClientChannel)
		}
		if inserted.ClientUserID == nil || *inserted.ClientUserID != userID {
			t.Errorf("expected client_user_id=%q, got %v", userID, inserted.ClientUserID)
		}
		if inserted.ClientUserName == nil || *inserted.ClientUserName != userName {
			t.Errorf("expected client_user_name=%q, got %v", userName, inserted.ClientUserName)
		}

		// Verify response contains client fields
		var resp APIQuoteResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.ClientType == nil || *resp.ClientType != irc {
			t.Errorf("expected client_type=%q in response, got %v", irc, resp.ClientType)
		}
		if resp.ClientNetwork == nil || *resp.ClientNetwork != freenode {
			t.Errorf("expected client_network=%q in response, got %v", freenode, resp.ClientNetwork)
		}
		if resp.ClientChannel == nil || *resp.ClientChannel != channel {
			t.Errorf("expected client_channel=%q in response, got %v", channel, resp.ClientChannel)
		}
		if resp.ClientUserID == nil || *resp.ClientUserID != userID {
			t.Errorf("expected client_user_id=%q in response, got %v", userID, resp.ClientUserID)
		}
		if resp.ClientUserName == nil || *resp.ClientUserName != userName {
			t.Errorf("expected client_user_name=%q in response, got %v", userName, resp.ClientUserName)
		}
	})
}

func TestAPIv1_ListQuotes_ClientFiltering(t *testing.T) {
	now := time.Now()
	irc := "irc"
	slack := "slack"
	freenode := "freenode"
	workspace := "myworkspace"
	chanGeneral := "#general"
	chanRandom := "#random"

	allQuotes := []data.Quote{
		{ID: 1, Timestamp: now, Quote: "IRC Quote", Author: "Author1", Poster: "poster1",
			ClientType: &irc, ClientNetwork: &freenode, ClientChannel: &chanGeneral},
		{ID: 2, Timestamp: now, Quote: "Slack Quote", Author: "Author2", Poster: "poster2",
			ClientType: &slack, ClientNetwork: &workspace, ClientChannel: &chanRandom},
		{ID: 3, Timestamp: now, Quote: "No Client Quote", Author: "Author3", Poster: "poster3"},
	}

	t.Run("filter by client_type returns subset", func(t *testing.T) {
		ircOnly := []data.Quote{allQuotes[0]}
		store := &mockQuoteStore{
			recentQuotesFn: func(filter data.ClientFilter) ([]data.Quote, error) {
				if filter.ClientType != nil && *filter.ClientType == "irc" {
					return ircOnly, nil
				}
				return allQuotes, nil
			},
		}
		handler := &Handler{
			Store:  store,
			Config: &config.Config{},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/quotes?client_type=irc", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler.APIv1QuotesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APIQuotesResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 quote, got %d", len(resp.Data))
		}
		if resp.Data[0].ClientType == nil || *resp.Data[0].ClientType != "irc" {
			t.Errorf("expected client_type=irc, got %v", resp.Data[0].ClientType)
		}
	})

	t.Run("client fields included in list response", func(t *testing.T) {
		store := &mockQuoteStore{quotes: allQuotes}
		handler := &Handler{
			Store:  store,
			Config: &config.Config{},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/quotes", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler.APIv1QuotesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APIQuotesResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// First quote should have IRC client fields
		q1 := resp.Data[0]
		if q1.ClientType == nil || *q1.ClientType != "irc" {
			t.Errorf("expected quote 1 client_type=irc, got %v", q1.ClientType)
		}
		if q1.ClientNetwork == nil || *q1.ClientNetwork != "freenode" {
			t.Errorf("expected quote 1 client_network=freenode, got %v", q1.ClientNetwork)
		}
		if q1.ClientChannel == nil || *q1.ClientChannel != "#general" {
			t.Errorf("expected quote 1 client_channel=#general, got %v", q1.ClientChannel)
		}

		// Second quote should have Slack client fields
		q2 := resp.Data[1]
		if q2.ClientType == nil || *q2.ClientType != "slack" {
			t.Errorf("expected quote 2 client_type=slack, got %v", q2.ClientType)
		}

		// Third quote should have nil client fields
		q3 := resp.Data[2]
		if q3.ClientType != nil {
			t.Errorf("expected quote 3 client_type=nil, got %v", q3.ClientType)
		}
	})
}

func TestAPIv1_QuoteResponse_ClientOmitEmpty(t *testing.T) {
	t.Run("null client fields omitted from JSON", func(t *testing.T) {
		now := time.Now()
		store := &mockQuoteStore{
			quoteByID: &data.Quote{
				ID:        1,
				Timestamp: now,
				Quote:     "To be or not to be",
				Author:    "Shakespeare",
				Poster:    "testuser",
			},
		}
		handler := &Handler{
			Store:  store,
			Config: &config.Config{},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/quotes/1", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler.APIv1QuotesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if strings.Contains(body, "client_type") {
			t.Error("expected client_type to be omitted from JSON when nil")
		}
		if strings.Contains(body, "client_network") {
			t.Error("expected client_network to be omitted from JSON when nil")
		}
		if strings.Contains(body, "client_channel") {
			t.Error("expected client_channel to be omitted from JSON when nil")
		}
		if strings.Contains(body, "client_user_id") {
			t.Error("expected client_user_id to be omitted from JSON when nil")
		}
		if strings.Contains(body, "client_user_name") {
			t.Error("expected client_user_name to be omitted from JSON when nil")
		}
	})

	t.Run("set client fields present in JSON", func(t *testing.T) {
		now := time.Now()
		irc := "irc"
		freenode := "freenode"
		channel := "#test"
		userID := "U999"
		userName := "bob"

		store := &mockQuoteStore{
			quoteByID: &data.Quote{
				ID:             2,
				Timestamp:      now,
				Quote:          "I think therefore I am",
				Author:         "Descartes",
				Poster:         "testuser",
				ClientType:     &irc,
				ClientNetwork:  &freenode,
				ClientChannel:  &channel,
				ClientUserID:   &userID,
				ClientUserName: &userName,
			},
		}
		handler := &Handler{
			Store:  store,
			Config: &config.Config{},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/quotes/2", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler.APIv1QuotesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APIQuoteResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.ClientType == nil || *resp.ClientType != "irc" {
			t.Errorf("expected client_type=irc, got %v", resp.ClientType)
		}
		if resp.ClientNetwork == nil || *resp.ClientNetwork != "freenode" {
			t.Errorf("expected client_network=freenode, got %v", resp.ClientNetwork)
		}
		if resp.ClientChannel == nil || *resp.ClientChannel != "#test" {
			t.Errorf("expected client_channel=#test, got %v", resp.ClientChannel)
		}
		if resp.ClientUserID == nil || *resp.ClientUserID != "U999" {
			t.Errorf("expected client_user_id=U999, got %v", resp.ClientUserID)
		}
		if resp.ClientUserName == nil || *resp.ClientUserName != "bob" {
			t.Errorf("expected client_user_name=bob, got %v", resp.ClientUserName)
		}

		body := w.Body.String()
		if !strings.Contains(body, `"client_type":"irc"`) {
			t.Error("expected client_type in JSON response body")
		}
	})
}
