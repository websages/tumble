package handler

import (
	"time"
)

// APIError represents a structured error in API responses.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// APIErrorResponse wraps an APIError for consistent error responses.
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// APIMeta contains pagination metadata for list responses.
type APIMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// APILinkResponse represents a single link in API responses.
type APILinkResponse struct {
	ID             int       `json:"id"`
	URL            string    `json:"url"`
	Title          string    `json:"title"`
	User           string    `json:"user"`
	Clicks         int       `json:"clicks"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []string  `json:"tags,omitempty"`
	ClientType     *string   `json:"client_type,omitempty"`
	ClientNetwork  *string   `json:"client_network,omitempty"`
	ClientChannel  *string   `json:"client_channel,omitempty"`
	ClientUserID   *string   `json:"client_user_id,omitempty"`
	ClientUserName *string   `json:"client_user_name,omitempty"`
}

// APIPreviousSubmission contains information about a previous submission
// of the same URL for duplicate detection.
type APIPreviousSubmission struct {
	ID        int       `json:"id"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	Title     string    `json:"title"`
}

// APILinkCreateResponse extends APILinkResponse with duplicate detection info.
type APILinkCreateResponse struct {
	APILinkResponse
	IsDuplicate         bool                    `json:"is_duplicate"`
	PreviousSubmissions []APIPreviousSubmission `json:"previous_submissions,omitempty"`
}

// APILinksResponse is the paginated response for a list of links.
type APILinksResponse struct {
	Data []APILinkResponse `json:"data"`
	Meta APIMeta           `json:"meta"`
}

// APIQuoteResponse represents a single quote in API responses.
type APIQuoteResponse struct {
	ID             int       `json:"id"`
	Quote          string    `json:"quote"`
	Author         string    `json:"author"`
	Poster         string    `json:"poster,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []string  `json:"tags,omitempty"`
	ClientType     *string   `json:"client_type,omitempty"`
	ClientNetwork  *string   `json:"client_network,omitempty"`
	ClientChannel  *string   `json:"client_channel,omitempty"`
	ClientUserID   *string   `json:"client_user_id,omitempty"`
	ClientUserName *string   `json:"client_user_name,omitempty"`
}

// APIQuotesResponse is the paginated response for a list of quotes.
type APIQuotesResponse struct {
	Data []APIQuoteResponse `json:"data"`
	Meta APIMeta            `json:"meta"`
}

// APISiteStats contains site-wide statistics.
type APISiteStats struct {
	TotalLinks  int `json:"total_links"`
	TotalQuotes int `json:"total_quotes"`
	TotalUsers  int `json:"total_users"`
}

// APIUserStats contains per-user statistics.
type APIUserStats struct {
	User       string `json:"user"`
	LinkCount  int    `json:"link_count"`
	QuoteCount int    `json:"quote_count"`
}

// APIStatsResponse includes site stats and leaderboard.
type APIStatsResponse struct {
	Site        APISiteStats   `json:"site"`
	Leaderboard []APIUserStats `json:"leaderboard"`
	Meta        APIMeta        `json:"meta"`
}

// APISearchMeta extends APIMeta with search-specific counts.
type APISearchMeta struct {
	APIMeta
	TotalLinks  int `json:"total_links"`
	TotalQuotes int `json:"total_quotes"`
}

// APISearchResponse contains separate arrays for links and quotes.
type APISearchResponse struct {
	Links  []APILinkResponse  `json:"links"`
	Quotes []APIQuoteResponse `json:"quotes"`
	Meta   APISearchMeta      `json:"meta"`
}

// APICacheClearResponse represents the response from cache clear operations.
// Cleared is either the specific URL that was cleared, or "all" if all cache was cleared.
// Count is only present when clearing all cache.
type APICacheClearResponse struct {
	Cleared string `json:"cleared"`
	Count   int    `json:"count,omitempty"`
}

// APICacheRefreshResponse represents the response from cache refresh operations.
type APICacheRefreshResponse struct {
	URL     string            `json:"url"`
	Preview map[string]string `json:"preview"`
	Cached  bool              `json:"cached"`
}

// APIKittenResponse represents the response from kitten operations.
type APIKittenResponse struct {
	URL     string `json:"url"`
	Date    string `json:"date"`
	Fetched bool   `json:"fetched"`
}

// APITagResponse represents a single tag in API responses.
type APITagResponse struct {
	ID           int       `json:"id"`
	Tag          string    `json:"tag"`
	ResourceType string    `json:"resource_type"`
	ResourceID   int       `json:"resource_id"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// APITagsResponse is the response for a list of tags on a resource.
type APITagsResponse struct {
	Data []APITagResponse `json:"data"`
}

// APITagCreateRequest is the request body for creating tags.
type APITagCreateRequest struct {
	Tags []string `json:"tags"`
	User string   `json:"user"`
}
