package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"tumble/internal/config"
	"tumble/internal/data"
	"tumble/internal/templates"
)

// MockIRCLinkStore implements data.Store for IRC link testing
type MockIRCLinkStore struct {
	data.Store
	InsertedLinks []data.IRCLink
	ExistingLinks map[string][]data.IRCLink // URL -> list of links
	LinksById     map[int]*data.IRCLink     // ID -> link (for GetIRCLinkByID)
	NextID        int
}

func (m *MockIRCLinkStore) GetIRCLinksByURL(ctx context.Context, url string) ([]data.IRCLink, error) {
	if links, ok := m.ExistingLinks[url]; ok {
		return links, nil
	}
	return []data.IRCLink{}, nil
}

func (m *MockIRCLinkStore) GetIRCLinkByID(ctx context.Context, id int) (*data.IRCLink, error) {
	if link, ok := m.LinksById[id]; ok {
		return link, nil
	}
	return nil, nil
}

func (m *MockIRCLinkStore) InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error) {
	m.NextID++
	link := data.IRCLink{
		ID:          m.NextID,
		User:        user,
		Title:       title,
		URL:         url,
		ContentType: contentType,
		Timestamp:   time.Now(),
		Clicks:      0,
	}
	m.InsertedLinks = append(m.InsertedLinks, link)
	return m.NextID, nil
}

// getTestRenderer creates a renderer for tests, or returns nil if templates can't be found
func getTestRenderer(t *testing.T) *templates.Renderer {
	// Try to create renderer - may fail in CI or if templates aren't available
	cfg := &config.Config{Mode: "development"}

	// Check if we're running from the project root
	if _, err := os.Stat("internal/templates/views"); err != nil {
		t.Skip("Skipping test: templates not found (not running from project root)")
		return nil
	}

	renderer, err := templates.NewRenderer(cfg)
	if err != nil {
		t.Skipf("Skipping test: could not create renderer: %v", err)
		return nil
	}
	return renderer
}

func TestIRCLinkHandler_NewLink_JSON(t *testing.T) {
	// Setup
	mockStore := &MockIRCLinkStore{
		ExistingLinks: make(map[string][]data.IRCLink),
		NextID:        100,
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: New link with JSON response
	req := httptest.NewRequest("GET", "/irclink/?user=alice&url=http://example.com", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify status code
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 Created, got %d", w.Code)
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}

	// Verify response body
	var response LinkSubmissionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.LinkID != 101 {
		t.Errorf("Expected LinkID 101, got %d", response.LinkID)
	}

	if response.IsDuplicate {
		t.Errorf("Expected IsDuplicate false, got true")
	}

	if len(response.PreviousSubmissions) != 0 {
		t.Errorf("Expected no previous submissions, got %d", len(response.PreviousSubmissions))
	}
}

func TestIRCLinkHandler_DuplicateLink_JSON(t *testing.T) {
	// Setup
	testURL := "http://example.com/article"
	existingTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	mockStore := &MockIRCLinkStore{
		ExistingLinks: map[string][]data.IRCLink{
			testURL: {
				{
					ID:        50,
					User:      "bob",
					Title:     "Example Article",
					URL:       testURL,
					Timestamp: existingTime,
					Clicks:    5,
				},
			},
		},
		NextID: 100,
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: Duplicate link with JSON response
	req := httptest.NewRequest("GET", "/irclink/?user=charlie&url="+testURL, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify status code
	if w.Code != http.StatusAlreadyReported {
		t.Errorf("Expected status 208 Already Reported, got %d", w.Code)
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}

	// Verify response body
	var response LinkSubmissionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.LinkID != 101 {
		t.Errorf("Expected LinkID 101, got %d", response.LinkID)
	}

	if !response.IsDuplicate {
		t.Errorf("Expected IsDuplicate true, got false")
	}

	if len(response.PreviousSubmissions) != 1 {
		t.Fatalf("Expected 1 previous submission, got %d", len(response.PreviousSubmissions))
	}

	prev := response.PreviousSubmissions[0]
	if prev.LinkID != 50 {
		t.Errorf("Expected previous LinkID 50, got %d", prev.LinkID)
	}
	if prev.User != "bob" {
		t.Errorf("Expected previous User 'bob', got %q", prev.User)
	}
	if prev.Title != "Example Article" {
		t.Errorf("Expected previous Title 'Example Article', got %q", prev.Title)
	}
}

func TestIRCLinkHandler_DuplicateLink_IRC_Source(t *testing.T) {
	// Setup
	testURL := "http://example.com/news"
	mockStore := &MockIRCLinkStore{
		ExistingLinks: map[string][]data.IRCLink{
			testURL: {
				{
					ID:        42,
					User:      "alice",
					Title:     "News Story",
					URL:       testURL,
					Timestamp: time.Now().Add(-24 * time.Hour),
				},
			},
		},
		NextID: 100,
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: Duplicate link with IRC source
	req := httptest.NewRequest("GET", "/irclink/?user=bob&url="+testURL+"&source=irc", nil)
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", w.Code)
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %q", contentType)
	}

	// Verify response contains ID and duplicate marker
	body := w.Body.String()
	if !strings.Contains(body, "101") {
		t.Errorf("Expected body to contain new ID '101', got %q", body)
	}
	if !strings.Contains(body, "duplicate") {
		t.Errorf("Expected body to contain 'duplicate', got %q", body)
	}
	if !strings.Contains(body, "alice") {
		t.Errorf("Expected body to contain previous user 'alice', got %q", body)
	}
}

func TestIRCLinkHandler_NewLink_IRC_Source(t *testing.T) {
	// Setup
	mockStore := &MockIRCLinkStore{
		ExistingLinks: make(map[string][]data.IRCLink),
		NextID:        200,
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: New link with IRC source
	req := httptest.NewRequest("GET", "/irclink/?user=alice&url=http://new-link.com&source=irc", nil)
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", w.Code)
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %q", contentType)
	}

	// Verify response is just the ID
	body := strings.TrimSpace(w.Body.String())
	if body != "201" {
		t.Errorf("Expected body '201', got %q", body)
	}
}

func TestIRCLinkHandler_DuplicateLink_HTML(t *testing.T) {
	renderer := getTestRenderer(t)
	if renderer == nil {
		return
	}

	// Setup
	testURL := "http://example.com/page"
	existingTime := time.Date(2026, 1, 20, 14, 30, 0, 0, time.UTC)

	mockStore := &MockIRCLinkStore{
		ExistingLinks: map[string][]data.IRCLink{
			testURL: {
				{
					ID:        75,
					User:      "dana",
					Title:     "Test Page",
					URL:       testURL,
					Timestamp: existingTime,
				},
			},
		},
		NextID: 150,
	}
	h := &Handler{
		Store:    mockStore,
		Config:   &config.Config{Mode: "development"},
		Renderer: renderer,
	}

	// Test Case: Duplicate link with HTML response (default)
	req := httptest.NewRequest("GET", "/irclink/?user=eve&url="+testURL, nil)
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", w.Code)
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %q", contentType)
	}

	// Verify response contains duplicate notification
	body := w.Body.String()
	if !strings.Contains(body, "Your link has been posted!") {
		t.Errorf("Expected success message in HTML body")
	}
	if !strings.Contains(body, "previously posted by") {
		t.Errorf("Expected duplicate notification in HTML body, got %q", body)
	}
	if !strings.Contains(body, "dana") {
		t.Errorf("Expected previous user 'dana' in HTML body, got %q", body)
	}
	if !strings.Contains(body, "2026-01-20") {
		t.Errorf("Expected date in HTML body, got %q", body)
	}
}

func TestIRCLinkHandler_NewLink_HTML(t *testing.T) {
	renderer := getTestRenderer(t)
	if renderer == nil {
		return
	}

	// Setup
	mockStore := &MockIRCLinkStore{
		ExistingLinks: make(map[string][]data.IRCLink),
		NextID:        300,
	}
	h := &Handler{
		Store:    mockStore,
		Config:   &config.Config{Mode: "development"},
		Renderer: renderer,
	}

	// Test Case: New link with HTML response (default)
	req := httptest.NewRequest("GET", "/irclink/?user=frank&url=http://brand-new.com", nil)
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", w.Code)
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %q", contentType)
	}

	// Verify response does NOT contain duplicate notification
	body := w.Body.String()
	if !strings.Contains(body, "Your link has been posted!") {
		t.Errorf("Expected success message in HTML body")
	}
	if strings.Contains(body, "previously posted") {
		t.Errorf("Did not expect duplicate notification for new link, got %q", body)
	}
}

func TestIRCLinkHandler_MultiplePreviousSubmissions(t *testing.T) {
	// Setup
	testURL := "http://popular.com/article"
	time1 := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC)
	time3 := time.Date(2026, 1, 20, 18, 0, 0, 0, time.UTC)

	mockStore := &MockIRCLinkStore{
		ExistingLinks: map[string][]data.IRCLink{
			testURL: {
				{ID: 30, User: "alice", Title: "Article", URL: testURL, Timestamp: time3},
				{ID: 20, User: "bob", Title: "Article", URL: testURL, Timestamp: time2},
				{ID: 10, User: "charlie", Title: "Article", URL: testURL, Timestamp: time1},
			},
		},
		NextID: 100,
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: Multiple previous submissions
	req := httptest.NewRequest("GET", "/irclink/?user=dana&url="+testURL, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify response
	var response LinkSubmissionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.IsDuplicate {
		t.Errorf("Expected IsDuplicate true")
	}

	if len(response.PreviousSubmissions) != 3 {
		t.Fatalf("Expected 3 previous submissions, got %d", len(response.PreviousSubmissions))
	}

	// Verify all three submissions are included (should be ordered by timestamp DESC)
	users := []string{response.PreviousSubmissions[0].User, response.PreviousSubmissions[1].User, response.PreviousSubmissions[2].User}
	expectedUsers := []string{"alice", "bob", "charlie"}
	for i, user := range users {
		if user != expectedUsers[i] {
			t.Errorf("Expected user %q at position %d, got %q", expectedUsers[i], i, user)
		}
	}
}

func TestIRCLinkHandler_API_Source(t *testing.T) {
	// Setup
	mockStore := &MockIRCLinkStore{
		ExistingLinks: make(map[string][]data.IRCLink),
		NextID:        400,
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: source=api should return JSON
	req := httptest.NewRequest("GET", "/irclink/?user=test&url=http://test.com&source=api", nil)
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify JSON response
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json for source=api, got %q", contentType)
	}

	var response LinkSubmissionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.LinkID != 401 {
		t.Errorf("Expected LinkID 401, got %d", response.LinkID)
	}
}

func TestIRCLinkHandler_GetLinkJSON(t *testing.T) {
	// Setup
	testTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	mockStore := &MockIRCLinkStore{
		ExistingLinks: make(map[string][]data.IRCLink),
		LinksById: map[int]*data.IRCLink{
			42: {
				ID:          42,
				User:        "alice",
				Title:       "Test Article",
				URL:         "http://example.com/article",
				Timestamp:   testTime,
				Clicks:      15,
				ContentType: "0",
			},
		},
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: GET /link/42.json should return link metadata
	req := httptest.NewRequest("GET", "/link/42.json", nil)
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", w.Code)
	}

	// Verify content type
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}

	// Verify response body
	var link data.IRCLink
	if err := json.Unmarshal(w.Body.Bytes(), &link); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if link.ID != 42 {
		t.Errorf("Expected ID 42, got %d", link.ID)
	}
	if link.User != "alice" {
		t.Errorf("Expected User 'alice', got %q", link.User)
	}
	if link.Title != "Test Article" {
		t.Errorf("Expected Title 'Test Article', got %q", link.Title)
	}
	if link.URL != "http://example.com/article" {
		t.Errorf("Expected URL 'http://example.com/article', got %q", link.URL)
	}
	if link.Clicks != 15 {
		t.Errorf("Expected Clicks 15, got %d", link.Clicks)
	}
}

func TestIRCLinkHandler_GetLinkJSON_NotFound(t *testing.T) {
	// Setup
	mockStore := &MockIRCLinkStore{
		ExistingLinks: make(map[string][]data.IRCLink),
		LinksById:     make(map[int]*data.IRCLink),
	}
	h := &Handler{
		Store:  mockStore,
		Config: &config.Config{},
	}

	// Test Case: GET /link/999.json for non-existent link
	req := httptest.NewRequest("GET", "/link/999.json", nil)
	w := httptest.NewRecorder()

	// Execute
	h.IRCLinkHandler(w, req)

	// Verify 404 status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found, got %d", w.Code)
	}
}
