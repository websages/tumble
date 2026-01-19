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
}

func (m *MockStore) InsertQuote(ctx context.Context, quote, author string) error {
	m.LastQuote = quote
	m.LastAuthor = author
	return nil
}

func (m *MockStore) GetRandomQuote(ctx context.Context) (*data.Quote, error) {
	return &data.Quote{
		Quote:  "Random wisdom",
		Author: "Random Person",
	}, nil
}

func TestQuoteHandler_UnescapesInput(t *testing.T) {
	// Setup
	mockStore := &MockStore{}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
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

func TestQuoteHandler_PartialParams(t *testing.T) {
	// Setup
	mockStore := &MockStore{}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: Only quote provided
	form := url.Values{}
	form.Add("quote", "Only quote")
	req := httptest.NewRequest("POST", "/quote/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.QuoteHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for partial params, got %d", w.Code)
	}

	// Test Case: Only author provided
	form = url.Values{}
	form.Add("author", "Only author")
	req = httptest.NewRequest("POST", "/quote/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	h.QuoteHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for partial params, got %d", w.Code)
	}
}
