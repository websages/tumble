package handler

import (
	"fmt"
	"net/http"
)

// QuoteHandler handles /quote/ submissions
func (h *Handler) QuoteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	quote := r.FormValue("quote")
	author := r.FormValue("author")

	if quote != "" && author != "" {
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

	http.Error(w, "Missing quote or author", http.StatusBadRequest)
}
