package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"io/ioutil"
	"strings"
	"time"
)

// LinkSubmissionResponse represents the response when a link is submitted
type LinkSubmissionResponse struct {
	LinkID           int                  `json:"link_id"`
	IsDuplicate      bool                 `json:"is_duplicate"`
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

		// Fetch title (simple impl)
		title := url // Default to URL
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		contentType := "0"
		if err == nil {
			defer resp.Body.Close()
			// Basic title extraction (should improve for production)
			if strings.Contains(resp.Header.Get("Content-Type"), "image") {
				contentType = "image"
			}
			// Extract title logic omitted for brevity, using URL as fallback
			// In real impl, read body and regex <title>
			body, _ := ioutil.ReadAll(resp.Body)
			if idx := strings.Index(string(body), "<title>"); idx != -1 {
				end := strings.Index(string(body)[idx:], "</title>")
				if end != -1 {
					title = string(body)[idx+7 : idx+end]
				}
			}
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

		// HTML Redirect Page
		w.Header().Set("Content-Type", "text/html")
		duplicateMessage := ""
		if isDuplicate {
			duplicateMessage = fmt.Sprintf(`<br /><br /><font color="#ff9900"><i>Note: This link was previously posted by <b>%s</b> on %s</i></font>`,
				existingLinks[0].User,
				existingLinks[0].Timestamp.Format("2006-01-02 15:04:05"))
		}
		fmt.Fprintf(w, `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/><html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">
<head>
    <title>tumblefish link posted</title>
    <META HTTP-EQUIV="Refresh"
          CONTENT="5; URL=%s">
</head>
<body>
    <font size="14px" color="#aaa" face="Helvetica, Arial, sand-serif">
    <b>Your link has been posted!</b>%s<br /><br />
    Redirecting back to <b>%s</b> in 5 seconds...
    </font>
</body>
</html>`, url, duplicateMessage, url)
		return
	}

	// Case 2: Redirecting (id param or query string)
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		// Fallback to RawQuery if param parsing failed or mostly likely it's /irclink/?12345
		idStr = r.URL.RawQuery
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Increment Clicks
	go h.Store.IncrementClicks(context.Background(), id) // Async

	// Determine URL
	redirectURL, err := h.Store.GetIRCLinkURL(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	log.Printf("id: [%d] Location: %s", id, redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
