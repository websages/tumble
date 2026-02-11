package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"tumble/internal/config"
	"tumble/internal/data"
)

// MockStore implements data.Store for testing
type MockStore struct {
	data.Store // Embed interface to skip implementing everything
	LastQuote  string
	LastAuthor string
	LastPoster string
	NextID     int
}

func (m *MockStore) InsertQuote(ctx context.Context, quote, author, poster string) (int, error) {
	m.LastQuote = quote
	m.LastAuthor = author
	m.LastPoster = poster
	if m.NextID == 0 {
		m.NextID = 1
	}
	id := m.NextID
	m.NextID++
	return id, nil
}

func (m *MockStore) GetRandomQuote(ctx context.Context) (*data.Quote, error) {
	return &data.Quote{
		Quote:  "Random wisdom",
		Author: "Random Person",
	}, nil
}

func TestQuoteHandler_UnescapesInput(t *testing.T) {
	// Setup
	mockStore := &MockStore{NextID: 42}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{BaseURL: "http://example.com"},
	}

	// Test Case: Encoded HTML entities
	// "I&quot;ve been to fort Dicks" -> "I"ve been to fort Dicks"
	form := url.Values{}
	form.Add("quote", "I&quot;ve been to fort Dicks")
	form.Add("author", "james&lt;white&gt;")

	req := httptest.NewRequest("POST", "/quote/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Execute
	h.QuoteHandler(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check if the store received the UNESCAPED string
	expectedQuote := "I\"ve been to fort Dicks"
	if mockStore.LastQuote != expectedQuote {
		t.Errorf("Expected quote %q, got %q", expectedQuote, mockStore.LastQuote)
	}

	expectedAuthor := "james<white>"
	if mockStore.LastAuthor != expectedAuthor {
		t.Errorf("Expected author %q, got %q", expectedAuthor, mockStore.LastAuthor)
	}

	// Verify response is the permalink URL
	expectedURL := "http://example.com/quote/42"
	if w.Body.String() != expectedURL {
		t.Errorf("Expected body %q, got %q", expectedURL, w.Body.String())
	}
}

func TestQuoteHandler_RandomQuote(t *testing.T) {
	// Setup
	mockStore := &MockStore{}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: No params -> Random Quote
	req := httptest.NewRequest("GET", "/quote/", nil)
	w := httptest.NewRecorder()

	// Execute
	h.QuoteHandler(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expectedBody := "Random wisdom -- Random Person"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestQuoteHandler_RandomQuote_ContentNegotiation(t *testing.T) {
	// Setup
	mockStore := &MockStore{}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: JSON
	reqJSON := httptest.NewRequest("GET", "/quote/", nil)
	reqJSON.Header.Set("Accept", "application/json")
	wJSON := httptest.NewRecorder()
	h.QuoteHandler(wJSON, reqJSON)

	if wJSON.Code != http.StatusOK {
		t.Errorf("JSON: Expected status 200, got %d", wJSON.Code)
	}
	if contentType := wJSON.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("JSON: Expected Content-Type application/json, got %q", contentType)
	}
	// Basic JSON check
	if !strings.Contains(wJSON.Body.String(), `"quote":"Random wisdom"`) {
		t.Errorf("JSON: Expected quote body, got %q", wJSON.Body.String())
	}

	// Test Case: HTML
	reqHTML := httptest.NewRequest("GET", "/quote/", nil)
	reqHTML.Header.Set("Accept", "text/html")
	wHTML := httptest.NewRecorder()
	h.QuoteHandler(wHTML, reqHTML)

	if wHTML.Code != http.StatusOK {
		t.Errorf("HTML: Expected status 200, got %d", wHTML.Code)
	}
	if contentType := wHTML.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("HTML: Expected Content-Type text/html, got %q", contentType)
	}
	// HTML body should be escaped if needed, but "Random wisdom" is safe.
	// Let's verify string presence.
	if !strings.Contains(wHTML.Body.String(), "Random wisdom -- Random Person") {
		t.Errorf("HTML: Expected quote body, got %q", wHTML.Body.String())
	}
}

func TestQuoteHandler_QuoteWithoutAuthor(t *testing.T) {
	// Setup
	mockStore := &MockStore{NextID: 99}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{BaseURL: "http://example.com"},
	}

	// Test Case: Only quote provided (author is optional)
	form := url.Values{}
	form.Add("quote", "A quote without author")
	req := httptest.NewRequest("POST", "/quote/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.QuoteHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if mockStore.LastQuote != "A quote without author" {
		t.Errorf("Expected quote %q, got %q", "A quote without author", mockStore.LastQuote)
	}

	if mockStore.LastAuthor != "" {
		t.Errorf("Expected empty author, got %q", mockStore.LastAuthor)
	}
}
