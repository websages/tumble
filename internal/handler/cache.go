package handler

import (
	"encoding/json"
	"net/http"
)

// InvalidateCacheHandler allows manual invalidation of a cached preview
// Route: /api/caching/invalidate?url=...
func (h *Handler) InvalidateCacheHandler(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	w.Header().Set("Content-Type", "application/json")

	if urlParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No URL provided"})
		return
	}

	// Delete from DB
	err := h.Store.DeleteLinkPreview(r.Context(), urlParam)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Cache invalidated"})
}
