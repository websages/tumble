package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
)

// QuoteHandler handles /quote/ submissions
func (h *Handler) QuoteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	quote := html.UnescapeString(r.FormValue("quote"))
	author := html.UnescapeString(r.FormValue("author"))

	if quote == "" && author == "" {
		// No params -> Return a random quote (fortune style)
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

		responseText := fmt.Sprintf("%s -- %s", q.Quote, q.Author)

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

	if quote != "" && author != "" {
		// Both params -> Insert Quote
		// Perl code did uri_unescape. net/http request parsing handles standard form encoding.
		// If these come in as query params or post body, FormValue gets them.

		err := h.Store.InsertQuote(ctx, quote, author)
		if err != nil {
			http.Error(w, "Database Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "1")
		return
	}

	// Partial params -> Error
	http.Error(w, "Missing quote or author", http.StatusBadRequest)
}
