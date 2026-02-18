package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tumble/internal/data"
)

// QuoteHandler handles /quote/ submissions and /quote/{id} permalinks
func (h *Handler) QuoteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Handle DELETE requests
	if r.Method == http.MethodDelete {
		if !h.isAuthorizedAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			// Try path: DELETE /quote/123
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

		err = h.Store.DeleteQuote(ctx, id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "Quote not found", http.StatusNotFound)
			} else {
				log.Printf("DeleteQuote error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Quote %d deleted", id)
		return
	}

	// Check if this is a permalink request: /quote/{id} or /quote/{id}.json
	path := strings.TrimSuffix(r.URL.Path, "/")
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		pathID := path[idx+1:]
		returnJSON := false

		// Check for .json suffix
		if strings.HasSuffix(pathID, ".json") {
			pathID = strings.TrimSuffix(pathID, ".json")
			returnJSON = true
		}

		// If we have a numeric ID, this is a permalink request
		if id, err := strconv.Atoi(pathID); err == nil && pathID != "" {
			h.handleQuotePermalink(w, r, id, returnJSON)
			return
		}
	}

	quote := html.UnescapeString(r.FormValue("quote"))
	author := html.UnescapeString(r.FormValue("author"))
	poster := html.UnescapeString(r.FormValue("poster"))

	if quote == "" {
		// No quote param -> Return a random quote (fortune style)
		q, err := h.Store.GetRandomQuote(ctx)
		if err != nil {
			http.Error(w, "Database Error", http.StatusInternalServerError)
			return
		}

		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "application/json") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(q); err != nil {
				http.Error(w, "JSON Encoding Error", http.StatusInternalServerError)
			}
			return
		}

		responseText := q.Quote
		if q.Author != "" {
			responseText = fmt.Sprintf("%s -- %s", q.Quote, q.Author)
		}

		if strings.Contains(accept, "text/html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			// Escape for HTML safety
			fmt.Fprint(w, html.EscapeString(responseText))
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, responseText)
		return
	}

	// Quote provided -> Insert Quote (author is optional)
	id, err := h.Store.InsertQuote(ctx, &data.Quote{Quote: quote, Author: author, Poster: poster})
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%s/quote/%d", h.Config.BaseURL, id)
}

// handleQuotePermalink handles /quote/{id} requests
func (h *Handler) handleQuotePermalink(w http.ResponseWriter, r *http.Request, id int, returnJSON bool) {
	ctx := r.Context()

	quote, err := h.Store.GetQuoteByID(ctx, id)
	if err != nil {
		log.Printf("GetQuoteByID error: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	if quote == nil {
		http.NotFound(w, r)
		return
	}

	// Check if JSON response is requested
	accept := r.Header.Get("Accept")
	if returnJSON || strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(quote)
		return
	}

	// Default: render HTML page
	templateData := map[string]interface{}{
		"Quote":     quote.Quote,
		"Author":    quote.Author,
		"ID":        quote.ID,
		"BaseURL":   h.Config.BaseURL,
		"PageTitle": fmt.Sprintf("Quote by %s", quote.Author),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Renderer.Render(w, "quote_permalink.html", templateData); err != nil {
		log.Printf("Error rendering quote_permalink template: %v", err)
		// Fallback to plain text
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if quote.Author != "" {
			fmt.Fprintf(w, "%s -- %s", quote.Quote, quote.Author)
		} else {
			fmt.Fprint(w, quote.Quote)
		}
	}
}
