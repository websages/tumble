package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tumble/internal/data"
)

// getTagStrings fetches tags for a resource and returns just the tag strings.
func (h *Handler) getTagStrings(ctx context.Context, resourceType string, resourceID int) []string {
	tags, err := h.Store.GetTagsByResource(ctx, resourceType, resourceID)
	if err != nil || len(tags) == 0 {
		return nil
	}
	result := make([]string, len(tags))
	for i, t := range tags {
		result[i] = t.Tag
	}
	return result
}

// createTagsForResource creates tags for a given resource, validating each tag.
// Returns an error string if validation fails, empty string on success.
func (h *Handler) createTagsForResource(ctx context.Context, resourceType string, resourceID int, tags []string, user string) string {
	if len(tags) > maxTagsPerResource {
		return "too many tags: maximum " + strconv.Itoa(maxTagsPerResource) + " tags per resource"
	}
	for _, tagStr := range tags {
		normalized, valid := validateTag(tagStr)
		if !valid {
			return "invalid tag (no spaces allowed, must be non-empty, max 100 chars)"
		}
		tag := data.Tag{
			Tag:          normalized,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			CreatedBy:    user,
		}
		if _, err := h.Store.CreateTag(ctx, tag); err != nil {
			return "failed to create tag"
		}
	}
	return ""
}

// maxTagsPerResource is the maximum number of tags allowed on a single resource.
const maxTagsPerResource = 50

// maxTagLength is the maximum length of a single tag string.
const maxTagLength = 100

// validateTag checks that a tag string is valid: non-empty, no spaces, lowercased, within length limit.
// Returns the normalized tag and whether it is valid.
func validateTag(tag string) (string, bool) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return "", false
	}
	if len(tag) > maxTagLength {
		return "", false
	}
	if strings.ContainsAny(tag, " \t\n\r") {
		return "", false
	}
	return tag, true
}

// isTagWriteAuthorized checks if a tag write (POST/PUT) is authorized.
// Allowed if admin API key is present, or if same username within 10 minutes of post creation.
func isTagWriteAuthorized(r *http.Request, adminSecret string, postUser string, postCreatedAt time.Time, requestUser string) bool {
	if isAuthorizedAPIKey(r, adminSecret) {
		return true
	}
	if requestUser == "" || postUser == "" {
		return false
	}
	if requestUser != postUser {
		return false
	}
	if time.Since(postCreatedAt) > 10*time.Minute {
		return false
	}
	return true
}

// APIv1LinkTagsHandler handles /api/v1/links/{id}/tags and /api/v1/links/{id}/tags/{tag}
func (h *Handler) APIv1LinkTagsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the path: /api/v1/links/{id}/tags[/{tag}]
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/links/")
	path = trimFormatSuffix(path)

	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[1] != "tags" {
		writeAPIError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}

	linkID, err := strconv.Atoi(parts[0])
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "Invalid link ID")
		return
	}

	// Check if there's a specific tag in the path (for DELETE)
	tagName := ""
	if len(parts) == 3 && parts[2] != "" {
		tagName = parts[2]
	}

	switch r.Method {
	case http.MethodGet:
		h.apiV1GetResourceTags(w, r, "link", linkID)
	case http.MethodPost:
		h.apiV1AddLinkTags(w, r, linkID)
	case http.MethodPut:
		h.apiV1ReplaceLinkTags(w, r, linkID)
	case http.MethodDelete:
		if tagName == "" {
			writeAPIError(w, http.StatusBadRequest, "bad_request", "Tag name required for delete")
			return
		}
		h.apiV1DeleteResourceTag(w, r, "link", linkID, tagName)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

// APIv1QuoteTagsHandler handles /api/v1/quotes/{id}/tags and /api/v1/quotes/{id}/tags/{tag}
func (h *Handler) APIv1QuoteTagsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the path: /api/v1/quotes/{id}/tags[/{tag}]
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/quotes/")
	path = trimFormatSuffix(path)

	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[1] != "tags" {
		writeAPIError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}

	quoteID, err := strconv.Atoi(parts[0])
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "Invalid quote ID")
		return
	}

	tagName := ""
	if len(parts) == 3 && parts[2] != "" {
		tagName = parts[2]
	}

	switch r.Method {
	case http.MethodGet:
		h.apiV1GetResourceTags(w, r, "quote", quoteID)
	case http.MethodPost:
		h.apiV1AddQuoteTags(w, r, quoteID)
	case http.MethodPut:
		h.apiV1ReplaceQuoteTags(w, r, quoteID)
	case http.MethodDelete:
		if tagName == "" {
			writeAPIError(w, http.StatusBadRequest, "bad_request", "Tag name required for delete")
			return
		}
		h.apiV1DeleteResourceTag(w, r, "quote", quoteID, tagName)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

// apiV1GetResourceTags handles GET for tags on a resource.
func (h *Handler) apiV1GetResourceTags(w http.ResponseWriter, r *http.Request, resourceType string, resourceID int) {
	ctx := r.Context()

	tags, err := h.Store.GetTagsByResource(ctx, resourceType, resourceID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch tags")
		return
	}

	apiTags := make([]APITagResponse, 0, len(tags))
	for _, t := range tags {
		apiTags = append(apiTags, APITagResponse{
			ID:           t.ID,
			Tag:          t.Tag,
			ResourceType: t.ResourceType,
			ResourceID:   t.ResourceID,
			CreatedBy:    t.CreatedBy,
			CreatedAt:    t.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, APITagsResponse{Data: apiTags})
}

// apiV1AddLinkTags handles POST /api/v1/links/{id}/tags
func (h *Handler) apiV1AddLinkTags(w http.ResponseWriter, r *http.Request, linkID int) {
	ctx := r.Context()

	link, err := h.Store.GetIRCLinkByID(ctx, linkID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch link")
		return
	}
	if link == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
		return
	}

	var req APITagCreateRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	if !isTagWriteAuthorized(r, h.Config.AdminSecret, link.User, link.Timestamp, req.User) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Not authorized to add tags to this resource")
		return
	}

	h.apiV1CreateTags(w, r, "link", linkID, req)
}

// apiV1AddQuoteTags handles POST /api/v1/quotes/{id}/tags
func (h *Handler) apiV1AddQuoteTags(w http.ResponseWriter, r *http.Request, quoteID int) {
	ctx := r.Context()

	quote, err := h.Store.GetQuoteByID(ctx, quoteID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch quote")
		return
	}
	if quote == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Quote not found")
		return
	}

	var req APITagCreateRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	// For quotes, the poster is the "owner" for tag authorization
	if !isTagWriteAuthorized(r, h.Config.AdminSecret, quote.Author, quote.Timestamp, req.User) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Not authorized to add tags to this resource")
		return
	}

	h.apiV1CreateTags(w, r, "quote", quoteID, req)
}

// apiV1CreateTags is a shared helper for creating tags on a resource.
func (h *Handler) apiV1CreateTags(w http.ResponseWriter, r *http.Request, resourceType string, resourceID int, req APITagCreateRequest) {
	ctx := r.Context()

	if len(req.Tags) == 0 {
		writeValidationError(w, map[string]string{"tags": "at least one tag is required"})
		return
	}

	// Check existing tag count to enforce limit
	existingTags, err := h.Store.GetTagsByResource(ctx, resourceType, resourceID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch existing tags")
		return
	}
	if len(existingTags)+len(req.Tags) > maxTagsPerResource {
		writeValidationError(w, map[string]string{"tags": "too many tags: maximum " + strconv.Itoa(maxTagsPerResource) + " tags per resource"})
		return
	}

	createdTags := make([]APITagResponse, 0, len(req.Tags))
	for _, tagStr := range req.Tags {
		normalized, valid := validateTag(tagStr)
		if !valid {
			writeValidationError(w, map[string]string{"tags": "invalid tag (no spaces allowed, must be non-empty, max 100 chars)"})
			return
		}

		tag := data.Tag{
			Tag:          normalized,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			CreatedBy:    req.User,
		}

		created, err := h.Store.CreateTag(ctx, tag)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to create tag")
			return
		}

		createdTags = append(createdTags, APITagResponse{
			ID:           created.ID,
			Tag:          created.Tag,
			ResourceType: created.ResourceType,
			ResourceID:   created.ResourceID,
			CreatedBy:    created.CreatedBy,
			CreatedAt:    created.CreatedAt,
		})
	}

	writeJSON(w, http.StatusCreated, APITagsResponse{Data: createdTags})
}

// apiV1ReplaceLinkTags handles PUT /api/v1/links/{id}/tags
func (h *Handler) apiV1ReplaceLinkTags(w http.ResponseWriter, r *http.Request, linkID int) {
	ctx := r.Context()

	link, err := h.Store.GetIRCLinkByID(ctx, linkID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch link")
		return
	}
	if link == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
		return
	}

	var req APITagCreateRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	if !isTagWriteAuthorized(r, h.Config.AdminSecret, link.User, link.Timestamp, req.User) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Not authorized to modify tags on this resource")
		return
	}

	h.apiV1ReplaceTags(w, r, "link", linkID, req)
}

// apiV1ReplaceQuoteTags handles PUT /api/v1/quotes/{id}/tags
func (h *Handler) apiV1ReplaceQuoteTags(w http.ResponseWriter, r *http.Request, quoteID int) {
	ctx := r.Context()

	quote, err := h.Store.GetQuoteByID(ctx, quoteID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch quote")
		return
	}
	if quote == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Quote not found")
		return
	}

	var req APITagCreateRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	if !isTagWriteAuthorized(r, h.Config.AdminSecret, quote.Author, quote.Timestamp, req.User) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Not authorized to modify tags on this resource")
		return
	}

	h.apiV1ReplaceTags(w, r, "quote", quoteID, req)
}

// apiV1ReplaceTags is a shared helper for replacing all tags on a resource.
func (h *Handler) apiV1ReplaceTags(w http.ResponseWriter, r *http.Request, resourceType string, resourceID int, req APITagCreateRequest) {
	ctx := r.Context()

	// Enforce tag limit
	if len(req.Tags) > maxTagsPerResource {
		writeValidationError(w, map[string]string{"tags": "too many tags: maximum " + strconv.Itoa(maxTagsPerResource) + " tags per resource"})
		return
	}

	// Validate ALL tags before making any changes to avoid partial state
	normalizedTags := make([]string, 0, len(req.Tags))
	for _, tagStr := range req.Tags {
		normalized, valid := validateTag(tagStr)
		if !valid {
			writeValidationError(w, map[string]string{"tags": "invalid tag (no spaces allowed, must be non-empty, max 100 chars)"})
			return
		}
		normalizedTags = append(normalizedTags, normalized)
	}

	// Delete existing tags
	if err := h.Store.DeleteTagsByResource(ctx, resourceType, resourceID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to remove existing tags")
		return
	}

	// If no new tags, return empty
	if len(normalizedTags) == 0 {
		writeJSON(w, http.StatusOK, APITagsResponse{Data: []APITagResponse{}})
		return
	}

	createdTags := make([]APITagResponse, 0, len(normalizedTags))
	for _, normalized := range normalizedTags {
		tag := data.Tag{
			Tag:          normalized,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			CreatedBy:    req.User,
		}

		created, err := h.Store.CreateTag(ctx, tag)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to create tag")
			return
		}

		createdTags = append(createdTags, APITagResponse{
			ID:           created.ID,
			Tag:          created.Tag,
			ResourceType: created.ResourceType,
			ResourceID:   created.ResourceID,
			CreatedBy:    created.CreatedBy,
			CreatedAt:    created.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, APITagsResponse{Data: createdTags})
}

// apiV1DeleteResourceTag handles DELETE /api/v1/{links|quotes}/{id}/tags/{tag}
func (h *Handler) apiV1DeleteResourceTag(w http.ResponseWriter, r *http.Request, resourceType string, resourceID int, tagName string) {
	if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Invalid or missing API key")
		return
	}

	ctx := r.Context()

	tags, err := h.Store.GetTagsByResource(ctx, resourceType, resourceID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch tags")
		return
	}

	normalized := strings.ToLower(strings.TrimSpace(tagName))
	for _, t := range tags {
		if t.Tag == normalized {
			if err := h.Store.DeleteTag(ctx, t.ID); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to delete tag")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	writeAPIError(w, http.StatusNotFound, "not_found", "Tag not found on this resource")
}
