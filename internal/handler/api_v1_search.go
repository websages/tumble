package handler

import (
	"net/http"
	"strings"

	"tumble/internal/data"
)

// APIv1SearchHandler handles GET /api/v1/search
// Search links and quotes with optional type filtering.
//
// Query parameters:
//   - q (required, min 4 chars) - search query
//   - type (optional, comma-separated: links, quotes, default: both)
//   - limit (default: 50, max: 1000, applies per type)
//   - offset (default: 0, applies per type)
func (h *Handler) APIv1SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Strip any format suffix
	path := r.URL.Path
	path = trimFormatSuffix(path)

	// Only GET is allowed
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	ctx := r.Context()

	// Parse and validate query parameter
	query := r.URL.Query().Get("q")
	if query == "" {
		writeValidationError(w, map[string]string{"q": "q is required"})
		return
	}
	if len(query) < 4 {
		writeValidationError(w, map[string]string{"q": "q must be at least 4 characters"})
		return
	}

	// Parse type filter
	typeParam := r.URL.Query().Get("type")
	searchLinks := true
	searchQuotes := true

	if typeParam != "" {
		types := strings.Split(typeParam, ",")
		searchLinks = false
		searchQuotes = false
		for _, t := range types {
			t = strings.TrimSpace(t)
			switch t {
			case "links":
				searchLinks = true
			case "quotes":
				searchQuotes = true
			}
		}
	}

	// Parse pagination parameters
	limit := parseIntParam(r, "limit", 50, 1000)
	offset := parseIntParam(r, "offset", 0, 1000000)

	// Parse source filter query params
	var sourceFilter data.SourceFilter
	if st := r.URL.Query().Get("source_type"); st != "" {
		sourceFilter.SourceType = &st
	}
	if sn := r.URL.Query().Get("source_network"); sn != "" {
		sourceFilter.SourceNetwork = &sn
	}
	if sc := r.URL.Query().Get("source_channel"); sc != "" {
		sourceFilter.SourceChannel = &sc
	}

	// Initialize response
	resp := APISearchResponse{
		Links:  []APILinkResponse{},
		Quotes: []APIQuoteResponse{},
		Meta: APISearchMeta{
			APIMeta: APIMeta{
				Total:  0,
				Limit:  limit,
				Offset: offset,
			},
			TotalLinks:  0,
			TotalQuotes: 0,
		},
	}

	// Search links if requested
	if searchLinks {
		links, err := h.Store.SearchIRCLinks(ctx, query, sourceFilter)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to search links")
			return
		}

		totalLinks := len(links)
		resp.Meta.TotalLinks = totalLinks

		// Apply offset
		if offset < len(links) {
			links = links[offset:]
		} else {
			links = nil
		}

		// Apply limit
		if limit < len(links) {
			links = links[:limit]
		}

		// Convert to API response format
		for _, link := range links {
			resp.Links = append(resp.Links, APILinkResponse{
				ID:             link.ID,
				URL:            link.URL,
				Title:          link.Title,
				User:           link.User,
				Clicks:         link.Clicks,
				CreatedAt:      link.Timestamp,
				SourceType:     link.SourceType,
				SourceNetwork:  link.SourceNetwork,
				SourceChannel:  link.SourceChannel,
				SourceUserID:   link.SourceUserID,
				SourceUserName: link.SourceUserName,
			})
		}
	}

	// Search quotes if requested
	if searchQuotes {
		quotes, err := h.Store.SearchQuotes(ctx, query, sourceFilter)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to search quotes")
			return
		}

		totalQuotes := len(quotes)
		resp.Meta.TotalQuotes = totalQuotes

		// Apply offset
		if offset < len(quotes) {
			quotes = quotes[offset:]
		} else {
			quotes = nil
		}

		// Apply limit
		if limit < len(quotes) {
			quotes = quotes[:limit]
		}

		// Convert to API response format
		for _, quote := range quotes {
			resp.Quotes = append(resp.Quotes, APIQuoteResponse{
				ID:             quote.ID,
				Quote:          quote.Quote,
				Author:         quote.Author,
				Poster:         quote.Poster,
				CreatedAt:      quote.Timestamp,
				SourceType:     quote.SourceType,
				SourceNetwork:  quote.SourceNetwork,
				SourceChannel:  quote.SourceChannel,
				SourceUserID:   quote.SourceUserID,
				SourceUserName: quote.SourceUserName,
			})
		}
	}

	// Calculate total (sum of links and quotes)
	resp.Meta.Total = resp.Meta.TotalLinks + resp.Meta.TotalQuotes

	writeJSON(w, http.StatusOK, resp)
}
