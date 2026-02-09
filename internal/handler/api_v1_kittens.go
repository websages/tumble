package handler

import (
	"net/http"
	"time"

	"tumble/internal/scheduler"
)

const catAASUser = "cat AAS"

// APIv1KittensDailyHandler handles the /api/v1/kittens/daily endpoint.
// GET: Returns today's kitten (404 if none exists)
// PUT: Ensures today's kitten exists (fetches if missing), requires auth
// DELETE: Removes today's kitten, requires auth
func (h *Handler) APIv1KittensDailyHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetKittenDaily(w, r)

	case http.MethodPut:
		if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
			writeAPIError(w, http.StatusForbidden, "forbidden", "Valid API key required")
			return
		}
		h.handlePutKittenDaily(w, r)

	case http.MethodDelete:
		if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
			writeAPIError(w, http.StatusForbidden, "forbidden", "Valid API key required")
			return
		}
		h.handleDeleteKittenDaily(w, r)

	default:
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET, PUT, or DELETE")
	}
}

// handleGetKittenDaily returns today's kitten or 404 if none exists.
func (h *Handler) handleGetKittenDaily(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := time.Now().Format("2006-01-02")

	image, err := h.Store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to get today's kitten")
		return
	}

	if image == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "No kitten for today")
		return
	}

	resp := APIKittenResponse{
		URL:     image.URL,
		Date:    today,
		Fetched: false,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePutKittenDaily ensures today's kitten exists, fetching if needed.
func (h *Handler) handlePutKittenDaily(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := time.Now().Format("2006-01-02")

	// Check if kitten already exists
	existing, err := h.Store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to check for existing kitten")
		return
	}

	if existing != nil {
		// Kitten already exists
		resp := APIKittenResponse{
			URL:     existing.URL,
			Date:    today,
			Fetched: false,
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Fetch a new kitten
	fetched, err := scheduler.FetchAndStoreDailyCat(ctx, h.Store)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch kitten")
		return
	}

	// Get the newly stored kitten
	newKitten, err := h.Store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil || newKitten == nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve new kitten")
		return
	}

	resp := APIKittenResponse{
		URL:     newKitten.URL,
		Date:    today,
		Fetched: fetched,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleDeleteKittenDaily removes today's kitten.
func (h *Handler) handleDeleteKittenDaily(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Check if kitten exists before deleting
	existing, err := h.Store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to check for existing kitten")
		return
	}

	if existing == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "No kitten for today")
		return
	}

	if err := h.Store.DeleteTodayImageByLink(ctx, catAASUser); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to delete kitten")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
