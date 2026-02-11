package handler

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAPIError_JSONMarshal(t *testing.T) {
	err := APIError{
		Code:    "invalid_request",
		Message: "The request was invalid",
		Field:   "url",
	}

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal APIError: %v", marshalErr)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"code":`) {
		t.Errorf("Expected 'code' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"message":`) {
		t.Errorf("Expected 'message' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"field":`) {
		t.Errorf("Expected 'field' field, got: %s", jsonStr)
	}
}

func TestAPIError_OmitEmptyField(t *testing.T) {
	err := APIError{
		Code:    "not_found",
		Message: "Resource not found",
		// Field is empty
	}

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal APIError: %v", marshalErr)
	}

	jsonStr := string(data)

	// Field should be omitted when empty
	if containsJSON(jsonStr, `"field":`) {
		t.Errorf("Expected 'field' to be omitted when empty, got: %s", jsonStr)
	}
}

func TestAPIErrorResponse_JSONMarshal(t *testing.T) {
	resp := APIErrorResponse{
		Error: APIError{
			Code:    "validation_error",
			Message: "Validation failed",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APIErrorResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify structure
	if !containsJSON(jsonStr, `"error":`) {
		t.Errorf("Expected 'error' field, got: %s", jsonStr)
	}
}

func TestAPIMeta_JSONMarshal(t *testing.T) {
	meta := APIMeta{
		Total:  100,
		Limit:  20,
		Offset: 40,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal APIMeta: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"total":`) {
		t.Errorf("Expected 'total' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"limit":`) {
		t.Errorf("Expected 'limit' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"offset":`) {
		t.Errorf("Expected 'offset' field, got: %s", jsonStr)
	}
}

func TestAPILinkResponse_JSONMarshal(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	link := APILinkResponse{
		ID:        42,
		URL:       "http://example.com",
		Title:     "Example",
		User:      "alice",
		Clicks:    15,
		CreatedAt: timestamp,
	}

	data, err := json.Marshal(link)
	if err != nil {
		t.Fatalf("Failed to marshal APILinkResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"id":`) {
		t.Errorf("Expected 'id' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"url":`) {
		t.Errorf("Expected 'url' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"title":`) {
		t.Errorf("Expected 'title' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"user":`) {
		t.Errorf("Expected 'user' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"clicks":`) {
		t.Errorf("Expected 'clicks' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"created_at":`) {
		t.Errorf("Expected 'created_at' field (snake_case), got: %s", jsonStr)
	}

	// Should NOT have camelCase
	if containsJSON(jsonStr, `"createdAt":`) {
		t.Errorf("Should not have camelCase 'createdAt', got: %s", jsonStr)
	}
}

func TestAPILinkCreateResponse_JSONMarshal(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	prevTime := time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)

	resp := APILinkCreateResponse{
		APILinkResponse: APILinkResponse{
			ID:        101,
			URL:       "http://example.com",
			Title:     "Example",
			User:      "bob",
			Clicks:    0,
			CreatedAt: timestamp,
		},
		IsDuplicate: true,
		PreviousSubmissions: []APIPreviousSubmission{
			{
				ID:        50,
				User:      "alice",
				CreatedAt: prevTime,
				Title:     "Example",
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APILinkCreateResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"is_duplicate":`) {
		t.Errorf("Expected 'is_duplicate' field (snake_case), got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"previous_submissions":`) {
		t.Errorf("Expected 'previous_submissions' field (snake_case), got: %s", jsonStr)
	}

	// Should NOT have camelCase
	if containsJSON(jsonStr, `"isDuplicate":`) {
		t.Errorf("Should not have camelCase 'isDuplicate', got: %s", jsonStr)
	}
	if containsJSON(jsonStr, `"previousSubmissions":`) {
		t.Errorf("Should not have camelCase 'previousSubmissions', got: %s", jsonStr)
	}
}

func TestAPILinkCreateResponse_OmitEmptyPreviousSubmissions(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	resp := APILinkCreateResponse{
		APILinkResponse: APILinkResponse{
			ID:        102,
			URL:       "http://newsite.com",
			Title:     "New Site",
			User:      "charlie",
			Clicks:    0,
			CreatedAt: timestamp,
		},
		IsDuplicate:         false,
		PreviousSubmissions: nil, // No previous submissions
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APILinkCreateResponse: %v", err)
	}

	jsonStr := string(data)

	// previous_submissions should be omitted when nil/empty
	if containsJSON(jsonStr, `"previous_submissions":`) {
		t.Errorf("Expected 'previous_submissions' to be omitted when empty, got: %s", jsonStr)
	}
}

func TestAPIPreviousSubmission_JSONMarshal(t *testing.T) {
	timestamp := time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)
	prev := APIPreviousSubmission{
		ID:        50,
		User:      "alice",
		CreatedAt: timestamp,
		Title:     "Previous Title",
	}

	data, err := json.Marshal(prev)
	if err != nil {
		t.Fatalf("Failed to marshal APIPreviousSubmission: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"id":`) {
		t.Errorf("Expected 'id' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"user":`) {
		t.Errorf("Expected 'user' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"created_at":`) {
		t.Errorf("Expected 'created_at' field (snake_case), got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"title":`) {
		t.Errorf("Expected 'title' field, got: %s", jsonStr)
	}
}

func TestAPILinksResponse_JSONMarshal(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	resp := APILinksResponse{
		Data: []APILinkResponse{
			{
				ID:        1,
				URL:       "http://example.com",
				Title:     "Example",
				User:      "alice",
				Clicks:    5,
				CreatedAt: timestamp,
			},
		},
		Meta: APIMeta{
			Total:  100,
			Limit:  20,
			Offset: 0,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APILinksResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify structure
	if !containsJSON(jsonStr, `"data":`) {
		t.Errorf("Expected 'data' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"meta":`) {
		t.Errorf("Expected 'meta' field, got: %s", jsonStr)
	}
}

func TestAPIQuoteResponse_JSONMarshal(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	quote := APIQuoteResponse{
		ID:        10,
		Quote:     "Hello, World!",
		Author:    "bob",
		Poster:    "alice",
		CreatedAt: timestamp,
	}

	data, err := json.Marshal(quote)
	if err != nil {
		t.Fatalf("Failed to marshal APIQuoteResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"id":`) {
		t.Errorf("Expected 'id' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"quote":`) {
		t.Errorf("Expected 'quote' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"author":`) {
		t.Errorf("Expected 'author' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"poster":`) {
		t.Errorf("Expected 'poster' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"created_at":`) {
		t.Errorf("Expected 'created_at' field (snake_case), got: %s", jsonStr)
	}
}

func TestAPIQuoteResponse_OmitEmptyPoster(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	quote := APIQuoteResponse{
		ID:        11,
		Quote:     "Test quote",
		Author:    "bob",
		Poster:    "", // Empty poster
		CreatedAt: timestamp,
	}

	data, err := json.Marshal(quote)
	if err != nil {
		t.Fatalf("Failed to marshal APIQuoteResponse: %v", err)
	}

	jsonStr := string(data)

	// poster should be omitted when empty
	if containsJSON(jsonStr, `"poster":`) {
		t.Errorf("Expected 'poster' to be omitted when empty, got: %s", jsonStr)
	}
}

func TestAPIQuotesResponse_JSONMarshal(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	resp := APIQuotesResponse{
		Data: []APIQuoteResponse{
			{
				ID:        1,
				Quote:     "Test quote",
				Author:    "bob",
				CreatedAt: timestamp,
			},
		},
		Meta: APIMeta{
			Total:  50,
			Limit:  10,
			Offset: 0,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APIQuotesResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify structure
	if !containsJSON(jsonStr, `"data":`) {
		t.Errorf("Expected 'data' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"meta":`) {
		t.Errorf("Expected 'meta' field, got: %s", jsonStr)
	}
}

func TestAPISiteStats_JSONMarshal(t *testing.T) {
	stats := APISiteStats{
		TotalLinks:  5000,
		TotalQuotes: 1500,
		TotalUsers:  100,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal APISiteStats: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"total_links":`) {
		t.Errorf("Expected 'total_links' field (snake_case), got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"total_quotes":`) {
		t.Errorf("Expected 'total_quotes' field (snake_case), got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"total_users":`) {
		t.Errorf("Expected 'total_users' field (snake_case), got: %s", jsonStr)
	}

	// Should NOT have camelCase
	if containsJSON(jsonStr, `"totalLinks":`) {
		t.Errorf("Should not have camelCase 'totalLinks', got: %s", jsonStr)
	}
}

func TestAPIUserStats_JSONMarshal(t *testing.T) {
	stats := APIUserStats{
		User:       "alice",
		LinkCount:  150,
		QuoteCount: 30,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal APIUserStats: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"user":`) {
		t.Errorf("Expected 'user' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"link_count":`) {
		t.Errorf("Expected 'link_count' field (snake_case), got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"quote_count":`) {
		t.Errorf("Expected 'quote_count' field (snake_case), got: %s", jsonStr)
	}

	// Should NOT have camelCase
	if containsJSON(jsonStr, `"linkCount":`) {
		t.Errorf("Should not have camelCase 'linkCount', got: %s", jsonStr)
	}
}

func TestAPIStatsResponse_JSONMarshal(t *testing.T) {
	resp := APIStatsResponse{
		Site: APISiteStats{
			TotalLinks:  5000,
			TotalQuotes: 1500,
			TotalUsers:  100,
		},
		Leaderboard: []APIUserStats{
			{User: "alice", LinkCount: 150, QuoteCount: 30},
			{User: "bob", LinkCount: 100, QuoteCount: 50},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APIStatsResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify structure
	if !containsJSON(jsonStr, `"site":`) {
		t.Errorf("Expected 'site' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"leaderboard":`) {
		t.Errorf("Expected 'leaderboard' field, got: %s", jsonStr)
	}
}

func TestAPISearchMeta_JSONMarshal(t *testing.T) {
	meta := APISearchMeta{
		APIMeta: APIMeta{
			Total:  150,
			Limit:  20,
			Offset: 0,
		},
		TotalLinks:  100,
		TotalQuotes: 50,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal APISearchMeta: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	if !containsJSON(jsonStr, `"total":`) {
		t.Errorf("Expected 'total' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"total_links":`) {
		t.Errorf("Expected 'total_links' field (snake_case), got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"total_quotes":`) {
		t.Errorf("Expected 'total_quotes' field (snake_case), got: %s", jsonStr)
	}
}

func TestAPISearchResponse_JSONMarshal(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	resp := APISearchResponse{
		Links: []APILinkResponse{
			{
				ID:        1,
				URL:       "http://example.com",
				Title:     "Example",
				User:      "alice",
				Clicks:    5,
				CreatedAt: timestamp,
			},
		},
		Quotes: []APIQuoteResponse{
			{
				ID:        10,
				Quote:     "Test quote",
				Author:    "bob",
				CreatedAt: timestamp,
			},
		},
		Meta: APISearchMeta{
			APIMeta: APIMeta{
				Total:  2,
				Limit:  20,
				Offset: 0,
			},
			TotalLinks:  1,
			TotalQuotes: 1,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APISearchResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify structure
	if !containsJSON(jsonStr, `"links":`) {
		t.Errorf("Expected 'links' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"quotes":`) {
		t.Errorf("Expected 'quotes' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"meta":`) {
		t.Errorf("Expected 'meta' field, got: %s", jsonStr)
	}
}

func TestAPICacheClearResponse_JSONMarshal(t *testing.T) {
	resp := APICacheClearResponse{
		Cleared: "all",
		Count:   42,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APICacheClearResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify field names
	if !containsJSON(jsonStr, `"cleared":`) {
		t.Errorf("Expected 'cleared' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"count":`) {
		t.Errorf("Expected 'count' field, got: %s", jsonStr)
	}
}

func TestAPICacheClearResponse_OmitEmptyCount(t *testing.T) {
	resp := APICacheClearResponse{
		Cleared: "https://example.com/article",
		// Count is 0, should be omitted
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APICacheClearResponse: %v", err)
	}

	jsonStr := string(data)

	// count should be omitted when zero (for specific URL clearing)
	if containsJSON(jsonStr, `"count":`) {
		t.Errorf("Expected 'count' to be omitted when zero, got: %s", jsonStr)
	}
}

func TestAPIKittenResponse_JSONMarshal(t *testing.T) {
	resp := APIKittenResponse{
		URL:     "https://example.com/kitten.jpg",
		Date:    "2026-01-15",
		Fetched: true,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APIKittenResponse: %v", err)
	}

	jsonStr := string(data)

	// Verify field names
	if !containsJSON(jsonStr, `"url":`) {
		t.Errorf("Expected 'url' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"date":`) {
		t.Errorf("Expected 'date' field, got: %s", jsonStr)
	}
	if !containsJSON(jsonStr, `"fetched":`) {
		t.Errorf("Expected 'fetched' field, got: %s", jsonStr)
	}
}

// Test helper function to check if a JSON string contains a field
func containsJSON(jsonStr, field string) bool {
	return len(jsonStr) > 0 && len(field) > 0 && (jsonStr[0:1] == "{" || jsonStr[0:1] == "[") && (containsSubstring(jsonStr, field))
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
