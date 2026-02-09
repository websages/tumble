package handler

import (
	"fmt"
	"net/http"
	"strings"
)

// APIv1StatsHandler routes requests to /api/v1/stats endpoint.
// It handles:
//   - GET /api/v1/stats - Site-wide stats and leaderboard
func (h *Handler) APIv1StatsHandler(w http.ResponseWriter, r *http.Request) {
	// Strip the /api/v1/stats prefix and any format suffix
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/stats")
	path = trimFormatSuffix(path)
	path = strings.TrimPrefix(path, "/")

	// Only handle the root stats path
	if path != "" {
		writeAPIError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}

	// Only GET is allowed
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	h.apiV1GetStats(w, r)
}

// apiV1GetStats handles GET /api/v1/stats
// Returns site-wide statistics and a user leaderboard.
func (h *Handler) apiV1GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters for leaderboard
	limit := parseIntParam(r, "limit", 50, 1000)
	offset := parseIntParam(r, "offset", 0, 1000000)

	// Get all user stats for the full leaderboard (to calculate total users)
	allUserStats, err := h.Store.GetUserStats(ctx, "links", 1000000, 0)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch user stats")
		return
	}

	totalUsers := len(allUserStats)

	// Apply pagination to leaderboard
	leaderboard := allUserStats
	if offset >= len(leaderboard) {
		leaderboard = nil
	} else {
		leaderboard = leaderboard[offset:]
	}
	if limit < len(leaderboard) {
		leaderboard = leaderboard[:limit]
	}

	// Get total links and quotes for site stats
	links, err := h.Store.GetRecentIRCLinks(ctx, 36500, 0) // ~100 years to get all
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch links")
		return
	}

	quotes, err := h.Store.GetRecentQuotes(ctx, 36500, 0) // ~100 years to get all
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch quotes")
		return
	}

	// Convert to API response format
	leaderboardData := make([]APIUserStats, 0, len(leaderboard))
	for _, stat := range leaderboard {
		leaderboardData = append(leaderboardData, APIUserStats{
			User:       stat.User,
			LinkCount:  stat.LinkCount,
			QuoteCount: stat.QuoteCount,
		})
	}

	resp := APIStatsResponse{
		Site: APISiteStats{
			TotalLinks:  len(links),
			TotalQuotes: len(quotes),
			TotalUsers:  totalUsers,
		},
		Leaderboard: leaderboardData,
		Meta: APIMeta{
			Total:  totalUsers,
			Limit:  limit,
			Offset: offset,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// APIv1UsersHandler routes requests to /api/v1/users/{user}/stats endpoint.
// It handles:
//   - GET /api/v1/users/{user}/stats - Per-user statistics
func (h *Handler) APIv1UsersHandler(w http.ResponseWriter, r *http.Request) {
	// Strip the /api/v1/users prefix
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/users")
	path = strings.TrimPrefix(path, "/")

	// Expected format: {user}/stats or {user}/stats.json or {user}/stats.txt
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		writeAPIError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}

	username := parts[0]
	subpath := parts[1]

	// Remove format suffix from subpath
	subpath = trimFormatSuffix(subpath)

	// Only /stats subpath is supported
	if subpath != "stats" {
		writeAPIError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}

	// Only GET is allowed
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	h.apiV1GetUserStats(w, r, username)
}

// apiV1GetUserStats handles GET /api/v1/users/{user}/stats
// Returns statistics for a specific user.
func (h *Handler) apiV1GetUserStats(w http.ResponseWriter, r *http.Request, username string) {
	ctx := r.Context()

	// Get all user stats (we need to find this specific user)
	allUserStats, err := h.Store.GetUserStats(ctx, "links", 1000000, 0)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch user stats")
		return
	}

	// Find the user
	var userStat *APIUserStats
	for _, stat := range allUserStats {
		if stat.User == username {
			userStat = &APIUserStats{
				User:       stat.User,
				LinkCount:  stat.LinkCount,
				QuoteCount: stat.QuoteCount,
			}
			break
		}
	}

	if userStat == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	// Check content negotiation for plain text
	if wantsPlainText(r) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s: %d links, %d quotes", userStat.User, userStat.LinkCount, userStat.QuoteCount)
		return
	}

	writeJSON(w, http.StatusOK, userStat)
}
