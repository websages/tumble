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
