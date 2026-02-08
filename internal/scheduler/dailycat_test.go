package scheduler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCatURL_Success(t *testing.T) {
	// Mock server that returns JSON with cat URL
	expectedURL := "https://cataas.com/cat/abc123xyz"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(catResponse{ID: "abc123xyz", URL: expectedURL})
	}))
	defer server.Close()

	url, err := fetchCatURL(server.URL)
	if err != nil {
		t.Fatalf("fetchCatURL failed: %v", err)
	}

	if url != expectedURL {
		t.Errorf("expected %s, got %s", expectedURL, url)
	}
}

func TestFetchCatURL_FallbackToID(t *testing.T) {
	// Mock server that returns JSON with only ID (no URL)
	expectedID := "abc123xyz"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(catResponse{ID: expectedID})
	}))
	defer server.Close()

	url, err := fetchCatURL(server.URL)
	if err != nil {
		t.Fatalf("fetchCatURL failed: %v", err)
	}

	expectedURL := "https://cataas.com/cat/" + expectedID
	if url != expectedURL {
		t.Errorf("expected %s, got %s", expectedURL, url)
	}
}

func TestFetchCatURL_EmptyID(t *testing.T) {
	// Mock server that returns JSON with empty ID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(catResponse{ID: ""})
	}))
	defer server.Close()

	_, err := fetchCatURL(server.URL)
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestFetchCatURL_InvalidJSON(t *testing.T) {
	// Mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	_, err := fetchCatURL(server.URL)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFetchCatURL_ServerError(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchCatURL(server.URL)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestFetchCatURL_NotFound(t *testing.T) {
	// Mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchCatURL(server.URL)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}
