package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/doyensec/safeurl"
)

// LinkSubmissionResponse represents the response when a link is submitted
type LinkSubmissionResponse struct {
	LinkID              int                  `json:"link_id"`
	IsDuplicate         bool                 `json:"is_duplicate"`
	PreviousSubmissions []PreviousSubmission `json:"previous_submissions,omitempty"`
}

// PreviousSubmission contains information about a previous submission of the same URL
type PreviousSubmission struct {
	LinkID    int       `json:"link_id"`
	User      string    `json:"user"`
	Timestamp time.Time `json:"timestamp"`
	Title     string    `json:"title"`
}

// IRCLinkHandler handles /irclink/?id (redirect) and POSTing new links
func (h *Handler) IRCLinkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Case 1: Posting a link (user & url params)
	user := r.URL.Query().Get("user")
	url := r.URL.Query().Get("url")

	if r.Method == http.MethodDelete {
		// Authenticate DELETE requests using admin secret
		if !h.isAuthorizedAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			// Try path if valid (though usually query param here)
			// But wait, the router handles /irclink/ so path info might be the ID
			// e.g. DELETE /irclink/123
			segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(segments) > 1 {
				idStr = segments[len(segments)-1]
			}
		}

		if idStr == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		err = h.Store.DeleteIRCLink(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "Link not found", http.StatusNotFound)
			} else {
				log.Printf("DeleteIRCLink error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Link %d deleted", id)
		return
	}

	if user != "" && url != "" {
		// Handle link posting
		// Check for existing submissions first
		existingLinks, err := h.Store.GetIRCLinksByURL(ctx, url)
		if err != nil {
			log.Printf("GetIRCLinksByURL error: %v", err)
			http.Error(w, fmt.Sprintf("Database Error: %v", err), http.StatusInternalServerError)
			return
		}

		isDuplicate := len(existingLinks) > 0

		// Fetch title using SSRF-safe client
		title := url // Default to URL
		contentType := "0"

		// Configure safeurl to block private/internal IPs and restrict schemes
		config := safeurl.GetConfigBuilder().
			SetTimeout(10 * time.Second).
			Build()
		client := safeurl.Client(config)

		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			// Basic title extraction (should improve for production)
			if strings.Contains(resp.Header.Get("Content-Type"), "image") {
				contentType = "image"
			}
			// Limit response body to 1MB to prevent memory exhaustion
			limitedBody := io.LimitReader(resp.Body, 1024*1024)
			body, _ := io.ReadAll(limitedBody)
			if idx := strings.Index(string(body), "<title>"); idx != -1 {
				end := strings.Index(string(body)[idx:], "</title>")
				if end != -1 {
					title = string(body)[idx+7 : idx+end]
				}
			}
		} else {
			// Log blocked URLs for debugging (private IPs, etc.)
			log.Printf("URL fetch blocked or failed for %s: %v", url, err)
		}

		// Insert the link (always insert, even if duplicate)
		id, err := h.Store.InsertIRCLink(ctx, user, title, url, contentType)
		if err != nil {
			log.Printf("InsertIRCLink error: %v", err)
			http.Error(w, fmt.Sprintf("Database Error: %v", err), http.StatusInternalServerError)
			return
		}

		source := r.URL.Query().Get("source")
		acceptHeader := r.Header.Get("Accept")

		// If client accepts JSON or source is "api", return JSON response
		if strings.Contains(acceptHeader, "application/json") || source == "api" {
			response := LinkSubmissionResponse{
				LinkID:      id,
				IsDuplicate: isDuplicate,
			}

			if isDuplicate {
				response.PreviousSubmissions = make([]PreviousSubmission, 0, len(existingLinks))
				for _, link := range existingLinks {
					response.PreviousSubmissions = append(response.PreviousSubmissions, PreviousSubmission{
						LinkID:    link.ID,
						User:      link.User,
						Timestamp: link.Timestamp,
						Title:     link.Title,
					})
				}
			}

			w.Header().Set("Content-Type", "application/json")
			if isDuplicate {
				// 208 Already Reported - indicates the resource has already been reported
				w.WriteHeader(http.StatusAlreadyReported)
			} else {
				w.WriteHeader(http.StatusCreated)
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// For IRC source, return plain text
		if source == "irc" {
			w.Header().Set("Content-Type", "text/plain")
			if isDuplicate {
				// Return the ID with a marker indicating it's a duplicate
				fmt.Fprintf(w, "%d (duplicate, previously posted by %s)", id, existingLinks[0].User)
			} else {
				fmt.Fprintf(w, "%d", id)
			}
			return
		}

		// HTML Redirect Page - use template
		w.Header().Set("Content-Type", "text/html")
		templateData := map[string]interface{}{
			"RedirectURL": url,
			"IsDuplicate": isDuplicate,
		}
		if isDuplicate {
			templateData["PreviousUser"] = existingLinks[0].User
			templateData["PreviousTimestamp"] = existingLinks[0].Timestamp.Format("2006-01-02 15:04:05")
		}
		if err := h.Renderer.Render(w, "link_posted.html", templateData); err != nil {
			log.Printf("Error rendering link_posted template: %v", err)
			// Fallback to simple response
			fmt.Fprintf(w, "Link posted! Redirecting...")
		}
		return
	}

	// Case 2: Redirecting (id param or query string)
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		// Fallback to RawQuery if param parsing failed
		// Handle formats like /irclink/?12345 or /irclink/?12345&sig=abc123
		rawQuery := r.URL.RawQuery
		if idx := strings.Index(rawQuery, "&"); idx != -1 {
			idStr = rawQuery[:idx]
		} else {
			idStr = rawQuery
		}
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Only increment clicks if signature is valid
	// This prevents bots from inflating click counts by hitting URLs directly
	sig := r.URL.Query().Get("sig")
	if ValidateClickSignature(id, sig, h.Config.ClickSigningKey) {
		go h.Store.IncrementClicks(context.Background(), id) // Async
	}

	// Determine URL
	redirectURL, err := h.Store.GetIRCLinkURL(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	log.Printf("id: [%d] Location: %s", id, redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// isLocalhost checks if the request originates from localhost/127.0.0.1
func isLocalhost(r *http.Request) bool {
	host, _, err := strings.Cut(r.RemoteAddr, ":")
	if err {
		// No port separator found, use the whole string
		host = r.RemoteAddr
	}

	// Check for IPv4 localhost
	if host == "127.0.0.1" || host == "localhost" {
		return true
	}

	// Check for IPv6 localhost
	if host == "::1" || host == "[::1]" {
		return true
	}

	return false
}

// isAuthorizedAdmin checks if the request contains a valid admin secret
func (h *Handler) isAuthorizedAdmin(r *http.Request) bool {
	// If no admin secret is configured, fall back to localhost check
	if h.Config.AdminSecret == "" {
		return isLocalhost(r)
	}

	// Check X-Admin-Secret header first
	secret := r.Header.Get("X-Admin-Secret")
	if secret == "" {
		// Fall back to query parameter
		secret = r.URL.Query().Get("secret")
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(secret), []byte(h.Config.AdminSecret)) == 1
}
