package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"io/ioutil"
	"strings"
	"time"
)

// IRCLinkHandler handles /irclink/?id (redirect) and POSTing new links
func (h *Handler) IRCLinkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Case 1: Posting a link (user & url params)
	user := r.URL.Query().Get("user")
	url := r.URL.Query().Get("url")

	if user != "" && url != "" {
		// Handle link posting
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

		// Insert
		id, err := h.Store.InsertIRCLink(ctx, user, title, url, contentType)
		if err != nil {
			http.Error(w, "Database Error", http.StatusInternalServerError)
			return
		}

		source := r.URL.Query().Get("source")
		if source == "irc" {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "%d", id)
			return
		}

		// HTML Redirect Page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/><html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">
<head>
    <title>tumblefish link posted</title>
    <META HTTP-EQUIV="Refresh"
          CONTENT="5; URL=%s">
</head>
<body>
    <font size="14px" color="#aaa" face="Helvetica, Arial, sand-serif">
    <b>Your link has been posted!</b><br /><br />
    Redirecting back to <b>%s</b> in 5 seconds...
    </font>
</body>
</html>`, url, url)
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
