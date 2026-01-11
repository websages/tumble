package handler

import (
	"net/http"
	"tumble/internal/assets"
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
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Tumble API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js" crossorigin></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: '/api/openapi.json',
      dom_id: '#swagger-ui',
    });
  };
</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
