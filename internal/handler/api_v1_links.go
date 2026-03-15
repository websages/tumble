package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tumble/internal/data"

	"github.com/doyensec/safeurl"
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
		// Check if this is a tags sub-resource (e.g., "5/tags" or "5/tags/foo")
		if strings.Contains(path, "/tags") {
			h.APIv1LinkTagsHandler(w, r)
			return
		}

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

	// Parse client filter query params
	clientFilter, err := parseClientFilter(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_params", err.Error())
		return
	}

	// Fetch all links from the last year
	// We fetch more than needed so we can paginate in-memory
	links, err := h.Store.GetRecentIRCLinks(ctx, 365, 0, clientFilter)
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
			ID:             link.ID,
			URL:            link.URL,
			Title:          link.Title,
			User:           link.User,
			Clicks:         link.Clicks,
			CreatedAt:      link.Timestamp,
			Tags:           h.getTagStrings(ctx, "link", link.ID),
			ClientType:     link.ClientType,
			ClientNetwork:  link.ClientNetwork,
			ClientChannel:  link.ClientChannel,
			ClientUserID:   link.ClientUserID,
			ClientUserName: link.ClientUserName,
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

// APILinkCreateRequest is the request body for POST /api/v1/links.
type APILinkCreateRequest struct {
	URL            string   `json:"url"`
	User           string   `json:"user"`
	Tags           []string `json:"tags,omitempty"`
	ClientType     *string  `json:"client_type,omitempty"`
	ClientNetwork  *string  `json:"client_network,omitempty"`
	ClientChannel  *string  `json:"client_channel,omitempty"`
	ClientUserID   *string  `json:"client_user_id,omitempty"`
	ClientUserName *string  `json:"client_user_name,omitempty"`
}

// apiV1CreateLink handles POST /api/v1/links
// Creates a new link with duplicate detection.
func (h *Handler) apiV1CreateLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decode JSON body
	var req APILinkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	// Validate required fields
	errors := make(map[string]string)
	if req.URL == "" {
		errors["url"] = "url is required"
	} else if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		errors["url"] = "url must start with http:// or https://"
	}
	if req.User == "" {
		errors["user"] = "user is required"
	}
	if len(errors) > 0 {
		writeValidationError(w, errors)
		return
	}

	// Check for duplicates (scoped by client if provided)
	clientFilter := data.ClientFilter{
		ClientType:    req.ClientType,
		ClientNetwork: req.ClientNetwork,
		ClientChannel: req.ClientChannel,
	}
	existingLinks, err := h.Store.GetIRCLinksByURL(ctx, req.URL, clientFilter)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to check for duplicates")
		return
	}

	// Fetch title and content type from URL
	title := req.URL
	contentType := ""

	config := safeurl.GetConfigBuilder().
		SetTimeout(10 * time.Second).
		Build()
	client := safeurl.Client(config)

	fetchResp, fetchErr := client.Get(req.URL)
	if fetchErr == nil {
		defer fetchResp.Body.Close()
		if strings.Contains(fetchResp.Header.Get("Content-Type"), "image") {
			contentType = "image"
		}
		limitedBody := io.LimitReader(fetchResp.Body, 1024*1024)
		body, _ := io.ReadAll(limitedBody)
		if idx := strings.Index(string(body), "<title>"); idx != -1 {
			end := strings.Index(string(body)[idx:], "</title>")
			if end != -1 {
				title = string(body)[idx+7 : idx+end]
				title = html.UnescapeString(title)
			}
		}
	} else {
		log.Printf("URL fetch blocked or failed for %s: %v", req.URL, fetchErr)
	}

	linkID, err := h.Store.InsertIRCLink(ctx, &data.IRCLink{
		User:           req.User,
		Title:          title,
		URL:            req.URL,
		ContentType:    contentType,
		ClientType:     req.ClientType,
		ClientNetwork:  req.ClientNetwork,
		ClientChannel:  req.ClientChannel,
		ClientUserID:   req.ClientUserID,
		ClientUserName: req.ClientUserName,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to create link")
		return
	}

	// Build response
	isDuplicate := len(existingLinks) > 0
	var previousSubmissions []APIPreviousSubmission
	if isDuplicate {
		previousSubmissions = make([]APIPreviousSubmission, 0, len(existingLinks))
		for _, link := range existingLinks {
			previousSubmissions = append(previousSubmissions, APIPreviousSubmission{
				ID:        link.ID,
				User:      link.User,
				CreatedAt: link.Timestamp,
				Title:     link.Title,
			})
		}
	}

	// Check content negotiation for plain text
	if wantsPlainText(r) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		if isDuplicate {
			firstDup := existingLinks[0]
			fmt.Fprintf(w, "Created link %d: %s (duplicate of link %d by %s)", linkID, req.URL, firstDup.ID, firstDup.User)
		} else {
			fmt.Fprintf(w, "Created link %d: %s", linkID, req.URL)
		}
		return
	}

	// Create tags if provided
	var tagStrings []string
	if len(req.Tags) > 0 {
		if errMsg := h.createTagsForResource(ctx, "link", linkID, req.Tags, req.User); errMsg != "" {
			writeValidationError(w, map[string]string{"tags": errMsg})
			return
		}
		tagStrings = h.getTagStrings(ctx, "link", linkID)
	}

	resp := APILinkCreateResponse{
		APILinkResponse: APILinkResponse{
			ID:             linkID,
			URL:            req.URL,
			Title:          title,
			User:           req.User,
			Clicks:         0,
			CreatedAt:      time.Now(),
			Tags:           tagStrings,
			ClientType:     req.ClientType,
			ClientNetwork:  req.ClientNetwork,
			ClientChannel:  req.ClientChannel,
			ClientUserID:   req.ClientUserID,
			ClientUserName: req.ClientUserName,
		},
		IsDuplicate:         isDuplicate,
		PreviousSubmissions: previousSubmissions,
	}

	writeJSON(w, http.StatusCreated, resp)
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
		ID:             link.ID,
		URL:            link.URL,
		Title:          link.Title,
		User:           link.User,
		Clicks:         link.Clicks,
		CreatedAt:      link.Timestamp,
		Tags:           h.getTagStrings(ctx, "link", link.ID),
		ClientType:     link.ClientType,
		ClientNetwork:  link.ClientNetwork,
		ClientChannel:  link.ClientChannel,
		ClientUserID:   link.ClientUserID,
		ClientUserName: link.ClientUserName,
	})
}

// apiV1DeleteLink handles DELETE /api/v1/links/{id}
// Requires X-API-Key header for authorization (localhost always allowed).
func (h *Handler) apiV1DeleteLink(w http.ResponseWriter, r *http.Request, id int) {
	// Check authorization - requires X-API-Key header
	if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Invalid or missing API key")
		return
	}

	ctx := r.Context()

	// Check if link exists
	link, err := h.Store.GetIRCLinkByID(ctx, id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch link")
		return
	}
	if link == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
		return
	}

	// Delete the link
	if err := h.Store.DeleteIRCLink(ctx, id); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to delete link")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204
}
