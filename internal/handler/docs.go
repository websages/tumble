package handler

import (
	"fmt"
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
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Tumble API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
  <style>
    body { margin: 0; padding: 0; display: flex; flex-direction: column; min-height: 100vh; font-family: sans-serif; }
    #swagger-ui { flex: 1; }
    .nav-header { padding: 10px 20px; background: #f8f8f8; border-bottom: 1px solid #ddd; }
    .nav-header a { text-decoration: none; color: #333; font-weight: bold; }
    .nav-header a:hover { color: #000; }
    .footer { padding: 20px; text-align: center; background: #f5f5f5; border-top: 1px solid #eee; font-size: 12px; color: #666; }
    .footer a { color: #444; text-decoration: none; }
    .footer a:hover { text-decoration: underline; }
  </style>
</head>
<body>

<div class="nav-header">
    <a href="/">&larr; Back to Tumble</a>
</div>

<div id="swagger-ui"></div>

<div class="footer">
    Source Code Available on <a href="http://github.com/websages/tumble">GitHub</a>
	<svg width="16" height="16" viewBox="0 0 16 16" fill="#000000" style="vertical-align: text-bottom; display: inline-block;"><path fill-rule="evenodd" d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"></path></svg>.
	Revision: <a href="https://github.com/websages/tumble/commit/%s">%s</a>
</div>

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
</html>`, version.CommitHash, version.CommitHash)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
