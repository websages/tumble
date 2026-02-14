package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tumble/internal/data"
)

// APIv1QuotesHandler routes requests to /api/v1/quotes endpoints.
// It handles:
//   - GET /api/v1/quotes - List all quotes (paginated)
//   - POST /api/v1/quotes - Create a new quote
//   - GET /api/v1/quotes/{id} - Get a single quote
//   - DELETE /api/v1/quotes/{id} - Delete a quote
func (h *Handler) APIv1QuotesHandler(w http.ResponseWriter, r *http.Request) {
	// Strip the /api/v1/quotes prefix and any format suffix
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/quotes")
	path = trimFormatSuffix(path)
	path = strings.TrimPrefix(path, "/")

	// Route based on path and method
	switch {
	case path == "" || path == "/":
		// Collection endpoints: GET (list) or POST (create)
		switch r.Method {
		case http.MethodGet:
			h.apiV1ListQuotes(w, r)
		case http.MethodPost:
			h.apiV1CreateQuote(w, r)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	default:
		// Check if this is a tags sub-resource (e.g., "5/tags" or "5/tags/foo")
		if strings.Contains(path, "/tags") {
			h.APIv1QuoteTagsHandler(w, r)
			return
		}

		// Individual resource endpoints: GET or DELETE
		// Path should be the ID
		id, err := strconv.Atoi(path)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_id", "Invalid quote ID")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.apiV1GetQuote(w, r, id)
		case http.MethodDelete:
			h.apiV1DeleteQuote(w, r, id)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	}
}

// apiV1ListQuotes handles GET /api/v1/quotes
// Returns a paginated list of quotes.
func (h *Handler) apiV1ListQuotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	limit := parseIntParam(r, "limit", 50, 1000)
	offset := parseIntParam(r, "offset", 0, 1000000)

	// Fetch all quotes from the last year
	// We fetch more than needed so we can paginate in-memory
	quotes, err := h.Store.GetRecentQuotes(ctx, 365, 0, data.SourceFilter{})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch quotes")
		return
	}

	total := len(quotes)

	// Apply offset
	if offset >= len(quotes) {
		quotes = nil
	} else {
		quotes = quotes[offset:]
	}

	// Apply limit
	if limit < len(quotes) {
		quotes = quotes[:limit]
	}

	// Convert to API response format
	data := make([]APIQuoteResponse, 0, len(quotes))
	for _, quote := range quotes {
		data = append(data, APIQuoteResponse{
			ID:        quote.ID,
			Quote:     quote.Quote,
			Author:    quote.Author,
			Poster:    quote.Poster,
			CreatedAt: quote.Timestamp,
			Tags:      h.getTagStrings(ctx, "quote", quote.ID),
		})
	}

	resp := APIQuotesResponse{
		Data: data,
		Meta: APIMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// APIQuoteCreateRequest is the request body for POST /api/v1/quotes.
type APIQuoteCreateRequest struct {
	Quote  string   `json:"quote"`
	Author string   `json:"author"`
	Poster string   `json:"poster"`
	Tags   []string `json:"tags,omitempty"`
}

// apiV1CreateQuote handles POST /api/v1/quotes
// Creates a new quote.
func (h *Handler) apiV1CreateQuote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decode JSON body
	var req APIQuoteCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	// Validate required fields
	errors := make(map[string]string)
	if req.Quote == "" {
		errors["quote"] = "quote is required"
	}
	if len(errors) > 0 {
		writeValidationError(w, errors)
		return
	}

	// Insert the quote
	quoteID, err := h.Store.InsertQuote(ctx, &data.Quote{Quote: req.Quote, Author: req.Author, Poster: req.Poster})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to create quote")
		return
	}

	// Check content negotiation for plain text
	if wantsPlainText(r) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		if req.Author != "" {
			fmt.Fprintf(w, "Created quote %d: \"%s\" - %s", quoteID, req.Quote, req.Author)
		} else {
			fmt.Fprintf(w, "Created quote %d: \"%s\"", quoteID, req.Quote)
		}
		return
	}

	// Create tags if provided
	var tagStrings []string
	if len(req.Tags) > 0 {
		poster := req.Poster
		if poster == "" {
			poster = req.Author
		}
		if errMsg := h.createTagsForResource(ctx, "quote", quoteID, req.Tags, poster); errMsg != "" {
			writeValidationError(w, map[string]string{"tags": errMsg})
			return
		}
		tagStrings = h.getTagStrings(ctx, "quote", quoteID)
	}

	resp := APIQuoteResponse{
		ID:        quoteID,
		Quote:     req.Quote,
		Author:    req.Author,
		Poster:    req.Poster,
		CreatedAt: time.Now(),
		Tags:      tagStrings,
	}

	writeJSON(w, http.StatusCreated, resp)
}

// apiV1GetQuote handles GET /api/v1/quotes/{id}
// Returns a single quote by ID, supports JSON (default) and plain text responses.
func (h *Handler) apiV1GetQuote(w http.ResponseWriter, r *http.Request, id int) {
	ctx := r.Context()

	quote, err := h.Store.GetQuoteByID(ctx, id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch quote")
		return
	}
	if quote == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Quote not found")
		return
	}

	// Check content negotiation for plain text
	if wantsPlainText(r) {
		w.Header().Set("Content-Type", "text/plain")
		if quote.Author != "" {
			fmt.Fprintf(w, "\"%s\" - %s", quote.Quote, quote.Author)
		} else {
			fmt.Fprintf(w, "\"%s\"", quote.Quote)
		}
		return
	}

	writeJSON(w, http.StatusOK, APIQuoteResponse{
		ID:        quote.ID,
		Quote:     quote.Quote,
		Author:    quote.Author,
		Poster:    quote.Poster,
		CreatedAt: quote.Timestamp,
		Tags:      h.getTagStrings(ctx, "quote", quote.ID),
	})
}

// apiV1DeleteQuote handles DELETE /api/v1/quotes/{id}
// Requires X-API-Key header for authorization (localhost always allowed).
func (h *Handler) apiV1DeleteQuote(w http.ResponseWriter, r *http.Request, id int) {
	// Check authorization - requires X-API-Key header
	if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Invalid or missing API key")
		return
	}

	ctx := r.Context()

	// Check if quote exists
	quote, err := h.Store.GetQuoteByID(ctx, id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch quote")
		return
	}
	if quote == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Quote not found")
		return
	}

	// Delete the quote
	if err := h.Store.DeleteQuote(ctx, id); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to delete quote")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204
}
