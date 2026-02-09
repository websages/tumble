package handler

import (
	"net/http"
)

// APIv1CacheHandler handles requests to /api/v1/cache endpoints.
// It handles:
//   - DELETE /api/v1/cache?url=... - Clear specific URL from cache
//   - DELETE /api/v1/cache - Clear all cache
func (h *Handler) APIv1CacheHandler(w http.ResponseWriter, r *http.Request) {
	// Only DELETE method allowed
	if r.Method != http.MethodDelete {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

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
