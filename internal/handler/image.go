package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
)

// ImageHandler handles /image/{id} permalinks for image posts (e.g. the
// daily kitten). It supports JSON and ActivityPub Note content negotiation
// in addition to a plain HTML fallback, mirroring QuoteHandler's permalink.
func (h *Handler) ImageHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		http.NotFound(w, r)
		return
	}
	idStr := path[idx+1:]
	returnJSON := false
	if strings.HasSuffix(idStr, ".json") {
		idStr = strings.TrimSuffix(idStr, ".json")
		returnJSON = true
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	image, err := h.Store.GetImageByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	if image == nil {
		http.NotFound(w, r)
		return
	}

	accept := r.Header.Get("Accept")
	if h.ActivityPub.Enabled() && (strings.Contains(accept, "application/activity+json") || strings.Contains(accept, "application/ld+json")) {
		note := h.ActivityPub.NoteForImage(image)
		note.Context = "https://www.w3.org/ns/activitystreams"
		w.Header().Set("Content-Type", "application/activity+json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(note)
		return
	}

	if returnJSON || strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(image)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>%s</title></head><body><img src="%s" alt="%s"><p><a href="%s">%s</a></p></body></html>`,
		html.EscapeString(image.Title), html.EscapeString(image.URL), html.EscapeString(image.Title), html.EscapeString(image.Link), html.EscapeString(image.Title))
}
