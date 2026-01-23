package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"tumble/internal/assets"
	"tumble/internal/version"
)

// OpenAPISpecHandler serves the raw OpenAPI JSON
func (h *Handler) OpenAPISpecHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Retrieve from embedded assets
	data, err := assets.StaticFS.ReadFile("openapi.json")
	if err != nil {
		http.Error(w, "Spec not found", http.StatusNotFound)
		return
	}
	w.Write(data)
}

// DocsHandler serves the Swagger UI page
func (h *Handler) DocsHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"GitCommit":    version.CommitHash,
		"GitCommitURL": fmt.Sprintf("https://github.com/websages/tumble/commit/%s", version.CommitHash),
		"Hot":          h.getHotHTML(r.Context()),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.Renderer.Render(w, "docs.html", data); err != nil {
		slog.Error("Error rendering docs", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
