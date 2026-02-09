package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// APIv1RedirectHandler handles GET /r/{id} - the public shortlink redirect.
// This redirects users to the actual URL associated with a link ID.
// If a valid click signature is provided via the sig query parameter,
// the click count is incremented asynchronously.
func (h *Handler) APIv1RedirectHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET and HEAD methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse ID from path: /r/{id}
	path := r.URL.Path
	idStr := strings.TrimPrefix(path, "/r/")

	// Check if we got a valid ID string
	if idStr == "" || idStr == path {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Only increment clicks if signature is valid
	// This prevents bots from inflating click counts by hitting URLs directly
	sig := r.URL.Query().Get("sig")
	if ValidateClickSignature(id, sig, h.Config.ClickSigningKey) {
		go h.Store.IncrementClicks(context.Background(), id) // Async
	}

	// Get the redirect URL
	redirectURL, err := h.Store.GetIRCLinkURL(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Validate URL scheme to prevent open redirect attacks (javascript:, data:, etc.)
	if !strings.HasPrefix(redirectURL, "http://") && !strings.HasPrefix(redirectURL, "https://") {
		log.Printf("Blocked redirect to invalid scheme: url=%q path=%q remote_addr=%q user_agent=%q referer=%q",
			redirectURL, r.URL.Path, r.RemoteAddr, r.UserAgent(), r.Referer())
		http.Error(w, "Invalid redirect URL", http.StatusBadRequest)
		return
	}

	log.Printf("id: [%d] Location: %s", id, redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
