package handler

import (
	"net/http"
	"strconv"
	"strings"
)

// APIv1LinksHandler routes requests to /api/v1/links endpoints.
// It handles:
//   - GET /api/v1/links - List all links (paginated)
//   - POST /api/v1/links - Create a new link
//   - GET /api/v1/links/{id} - Get a single link
//   - DELETE /api/v1/links/{id} - Delete a link
func (h *Handler) APIv1LinksHandler(w http.ResponseWriter, r *http.Request) {
	// Strip the /api/v1/links prefix and any format suffix
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/links")
	path = trimFormatSuffix(path)
	path = strings.TrimPrefix(path, "/")

	// Route based on path and method
	switch {
	case path == "" || path == "/":
		// Collection endpoints: GET (list) or POST (create)
		switch r.Method {
		case http.MethodGet:
			h.apiV1ListLinks(w, r)
		case http.MethodPost:
			h.apiV1CreateLink(w, r)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	default:
		// Individual resource endpoints: GET or DELETE
		// Path should be the ID
		id, err := strconv.Atoi(path)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_id", "Invalid link ID")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.apiV1GetLink(w, r, id)
		case http.MethodDelete:
			h.apiV1DeleteLink(w, r, id)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		}
	}
}

// apiV1ListLinks handles GET /api/v1/links
// Returns a paginated list of links.
func (h *Handler) apiV1ListLinks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	limit := parseIntParam(r, "limit", 50, 1000)
	offset := parseIntParam(r, "offset", 0, 1000000)

	// Fetch all links from the last year
	// We fetch more than needed so we can paginate in-memory
	links, err := h.Store.GetRecentIRCLinks(ctx, 365, 0)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch links")
		return
	}

	total := len(links)

	// Apply offset
	if offset >= len(links) {
		links = nil
	} else {
		links = links[offset:]
	}

	// Apply limit
	if limit < len(links) {
		links = links[:limit]
	}

	// Convert to API response format
	data := make([]APILinkResponse, 0, len(links))
	for _, link := range links {
		data = append(data, APILinkResponse{
			ID:        link.ID,
			URL:       link.URL,
			Title:     link.Title,
			User:      link.User,
			Clicks:    link.Clicks,
			CreatedAt: link.Timestamp,
		})
	}

	resp := APILinksResponse{
		Data: data,
		Meta: APIMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// apiV1CreateLink handles POST /api/v1/links
// Stub for now - to be implemented in Task 2.3
func (h *Handler) apiV1CreateLink(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, http.StatusNotImplemented, "not_implemented", "Not yet implemented")
}

// apiV1GetLink handles GET /api/v1/links/{id}
// Returns a single link by ID, supports JSON (default) and plain text responses.
func (h *Handler) apiV1GetLink(w http.ResponseWriter, r *http.Request, id int) {
	ctx := r.Context()

	link, err := h.Store.GetIRCLinkByID(ctx, id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch link")
		return
	}
	if link == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
		return
	}

	// Check content negotiation for plain text
	if wantsPlainText(r) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(link.Title + " - " + link.URL))
		return
	}

	writeJSON(w, http.StatusOK, APILinkResponse{
		ID:        link.ID,
		URL:       link.URL,
		Title:     link.Title,
		User:      link.User,
		Clicks:    link.Clicks,
		CreatedAt: link.Timestamp,
	})
}

// apiV1DeleteLink handles DELETE /api/v1/links/{id}
// Stub for now - to be implemented in Task 2.4
func (h *Handler) apiV1DeleteLink(w http.ResponseWriter, r *http.Request, id int) {
	writeAPIError(w, http.StatusNotImplemented, "not_implemented", "Not yet implemented")
}
