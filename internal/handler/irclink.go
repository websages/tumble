package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tumble/internal/data"

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
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			log.Printf("Link submission invalid scheme: url=%q user=%q path=%q remote_addr=%q user_agent=%q referer=%q",
				url, user, r.URL.Path, r.RemoteAddr, r.UserAgent(), r.Referer())
			http.Error(w, "Invalid URL scheme", http.StatusBadRequest)
			return
		}

		// Handle link posting
		// Check for existing submissions first
		existingLinks, err := h.Store.GetIRCLinksByURL(ctx, url, data.SourceFilter{})
		if err != nil {
			h.ServerError(w, r, err)
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
					// Decode HTML entities (e.g., &amp; -> &, &#39; -> ')
					title = html.UnescapeString(title)
				}
			}
		} else {
			// Log blocked URLs for debugging (private IPs, etc.)
			log.Printf("URL fetch blocked or failed for %s: %v", url, err)
		}

		// Insert the link (always insert, even if duplicate)
		id, err := h.Store.InsertIRCLink(ctx, &data.IRCLink{User: user, Title: title, URL: url, ContentType: contentType})
		if err != nil {
			h.ServerError(w, r, err)
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

	// Case 2: Redirecting (id param, path, or query string)
	idStr := ""
	returnJSON := false

	// First, try to extract ID from path: /link/123 or /irclink/123 or /link/123.json
	path := strings.TrimSuffix(r.URL.Path, "/")
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		pathID := path[idx+1:]
		// Check for .json suffix to return metadata instead of redirect
		if strings.HasSuffix(pathID, ".json") {
			pathID = strings.TrimSuffix(pathID, ".json")
			returnJSON = true
		}
		if _, err := strconv.Atoi(pathID); err == nil {
			idStr = pathID
		}
	}

	// Fallback to query parameter: ?id=123
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}

	// Fallback to RawQuery for legacy format: ?123 or ?123&sig=abc
	if idStr == "" {
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

	// If .json suffix was used, return link metadata as JSON
	if returnJSON {
		link, err := h.Store.GetIRCLinkByID(ctx, id)
		if err != nil || link == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(link)
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

// isLocalhost checks if the request originates from localhost/127.0.0.1
func isLocalhost(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// No port, use the whole string (strip brackets if present)
		host = strings.Trim(r.RemoteAddr, "[]")
	}

	// Check for IPv4 localhost
	if host == "127.0.0.1" || host == "localhost" {
		return true
	}

	// Check for IPv6 localhost
	if host == "::1" {
		return true
	}

	return false
}

// isAuthorizedAdmin checks if the request contains a valid admin secret
func (h *Handler) isAuthorizedAdmin(r *http.Request) bool {
	// Always allow localhost requests
	if isLocalhost(r) {
		return true
	}

	// For non-localhost requests, require admin secret
	if h.Config.AdminSecret == "" {
		log.Printf("Admin auth failed: no admin secret configured and request is not from localhost (remote_addr=%s)", r.RemoteAddr)
		return false
	}

	// Check X-Admin-Secret header first
	secret := r.Header.Get("X-Admin-Secret")
	if secret == "" {
		// Fall back to query parameter
		secret = r.URL.Query().Get("secret")
	}

	if secret == "" {
		log.Printf("Admin auth failed: no secret provided (remote_addr=%s)", r.RemoteAddr)
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(secret), []byte(h.Config.AdminSecret)) != 1 {
		log.Printf("Admin auth failed: secret mismatch (remote_addr=%s)", r.RemoteAddr)
		return false
	}

	return true
}
