package handler

import (
	"encoding/json"
	"net/http"

	"tumble/internal/scheduler"
)

// KittenFetchResponse is the response for the kitten fetch endpoint
type KittenFetchResponse struct {
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

// FetchKittenHandler forces a new daily kitten to be fetched.
// This deletes today's existing kitten (if any) and fetches a new one.
// Route: POST /api/kitten/fetch
func (h *Handler) FetchKittenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(KittenFetchResponse{
			Status: "error",
			Error:  "Method not allowed. Use POST.",
		})
		return
	}

	catURL, err := scheduler.ForceFetchDailyCat(r.Context(), h.Store)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(KittenFetchResponse{
			Status: "error",
			Error:  err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(KittenFetchResponse{
		Status: "ok",
		URL:    catURL,
	})
}
