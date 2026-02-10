package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
)

// APIv1CacheHandler handles requests to /api/v1/cache endpoints.
// It handles:
//   - DELETE /api/v1/cache?url=... - Clear specific URL from cache
//   - DELETE /api/v1/cache - Clear all cache
//   - PUT /api/v1/cache?id=...&url=... - Refresh a cached preview
func (h *Handler) APIv1CacheHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		h.handleCacheDelete(w, r)
	case http.MethodPut:
		h.handleCacheRefresh(w, r)
	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	}
}

func (h *Handler) handleCacheDelete(w http.ResponseWriter, r *http.Request) {

	// Check authorization - requires X-API-Key header
	if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Invalid or missing API key")
		return
	}

	ctx := r.Context()

	// Check if specific URL is provided
	urlParam := r.URL.Query().Get("url")

	if urlParam != "" {
		// Clear specific URL from cache
		if err := h.Store.DeleteLinkPreview(ctx, urlParam); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to clear cache")
			return
		}

		writeJSON(w, http.StatusOK, APICacheClearResponse{
			Cleared: urlParam,
		})
		return
	}

	// Clear all cache
	count, err := h.Store.DeleteAllLinkPreviews(ctx)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to clear cache")
		return
	}

	writeJSON(w, http.StatusOK, APICacheClearResponse{
		Cleared: "all",
		Count:   count,
	})
}

func (h *Handler) handleCacheRefresh(w http.ResponseWriter, r *http.Request) {
	if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Invalid or missing API key")
		return
	}

	ctx := r.Context()
	idParam := r.URL.Query().Get("id")
	urlParam := r.URL.Query().Get("url")

	if idParam == "" && urlParam == "" {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "Either 'id' or 'url' query parameter is required")
		return
	}

	// Resolve URL from id if needed
	targetURL := urlParam
	if idParam != "" {
		id, err := strconv.Atoi(idParam)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "bad_request", "Invalid id parameter")
			return
		}
		resolved, err := h.Store.GetIRCLinkURL(ctx, id)
		if err != nil || resolved == "" {
			writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
			return
		}
		targetURL = resolved
	}

	// Delete existing cache entry
	h.Store.DeleteLinkPreview(ctx, targetURL)

	// Re-fetch preview by invoking OGPreviewHandler internally
	rec := httptest.NewRecorder()
	fakeReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/ogpreview.cgi?url="+targetURL, nil)
	h.OGPreviewHandler(rec, fakeReq)

	// Parse the response from the recorder
	var preview map[string]string
	json.Unmarshal(rec.Body.Bytes(), &preview)

	_, cached := preview["error"]
	writeJSON(w, http.StatusOK, APICacheRefreshResponse{
		URL:     targetURL,
		Preview: preview,
		Cached:  !cached,
	})
}
