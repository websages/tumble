package templates

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"

	"tumble/internal/assets"
	"tumble/internal/config"
)

//go:embed views/*.html views/*.xml
var viewsFS embed.FS

type Renderer struct {
	cfg      *config.Config
	mu       sync.RWMutex
	html     map[string]*template.Template // keyed by theme name; "" is the default look
	xmlTmpls *texttemplate.Template
}

// templateFuncs provides helper functions for templates
var templateFuncs = template.FuncMap{
	// linkURL builds a click-tracking URL for links (primary endpoint)
	// Uses clean path format: /link/123 or /link/123?sig=abc
	"linkURL": func(baseURL string, id int, sig string) string {
		if sig != "" {
			return fmt.Sprintf("%s/link/%d?sig=%s", baseURL, id, sig)
		}
		return fmt.Sprintf("%s/link/%d", baseURL, id)
	},
	// truncate shortens a string to max characters with ellipsis
	"truncate": func(s string, max int) string {
		if len(s) > max {
			return s[:max] + "..."
		}
		return s
	},
	// safeHTML marks a string as safe HTML (use sparingly)
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	// safeURL marks a string as a safe URL
	"safeURL": func(s string) template.URL {
		return template.URL(s)
	},
}

// textTemplateFuncs is the equivalent for text/template (XML)
var textTemplateFuncs = texttemplate.FuncMap{
	// linkURL builds a click-tracking URL for links (primary endpoint)
	// Uses clean path format: /link/123 or /link/123?sig=abc
	// Note: Uses &amp; for XML-safe output since text/template doesn't auto-escape
	"linkURL": func(baseURL string, id int, sig string) string {
		if sig != "" {
			return fmt.Sprintf("%s/link/%d?sig=%s", baseURL, id, sig)
		}
		return fmt.Sprintf("%s/link/%d", baseURL, id)
	},
	"truncate": func(s string, max int) string {
		if len(s) > max {
			return s[:max] + "..."
		}
		return s
	},
	// xmlEscape escapes special XML characters (&, <, >, ", ')
	"xmlEscape": func(s string) string {
		return html.EscapeString(s)
	},
}

func NewRenderer(cfg *config.Config) (*Renderer, error) {
	r := &Renderer{cfg: cfg}

	// In production, parse once at startup.
	// In development, we can parse now too to fail early on static errors,
	// but Render will re-parse.
	if err := r.parseTemplates(); err != nil {
		return nil, err
	}

	return r, nil
}

// themes returns the themes this renderer serves: just the configured
// theme, or every theme plus the default look when users may pick one.
func (r *Renderer) themes() []string {
	if !r.cfg.AllowUserThemes {
		return []string{r.cfg.Theme}
	}
	return append([]string{""}, assets.Themes()...)
}

// htmlFuncMap returns the HTML template funcs plus config-backed helpers that
// need to close over the renderer's config (e.g. siteName) and the theme the
// template set is parsed for.
func (r *Renderer) htmlFuncMap(theme string) template.FuncMap {
	fm := template.FuncMap{}
	for k, v := range templateFuncs {
		fm[k] = v
	}
	// siteName is the configurable wordmark shown in page titles and the
	// masthead. Defaults to "tumblefish" via config.
	fm["siteName"] = func() string {
		return r.cfg.SiteName
	}
	// themeName is the active theme, or empty for the default look.
	fm["themeName"] = func() string {
		return theme
	}
	// themeStylesheet is the URL of the theme's CSS, loaded after
	// screen.css. Empty when no theme is set, so the default look is used.
	fm["themeStylesheet"] = func() string {
		return themeFile(theme, "theme.css")
	}
	// themeScript is the URL of the theme's optional JavaScript.
	fm["themeScript"] = func() string {
		return themeFile(theme, "theme.js")
	}
	// userThemes lists the themes a visitor can pick from, or nothing when
	// user theme selection is disabled.
	fm["userThemes"] = func() []string {
		if !r.cfg.AllowUserThemes {
			return nil
		}
		return assets.Themes()
	}
	// faviconURL and touchIconURL are the site icons: the configured
	// favicon, else the theme's favicon.png, else the built-in files.
	fm["faviconURL"] = func() string {
		if u := assets.IconURL(r.cfg.Favicon, theme); u != "" {
			return u
		}
		return "/favicon.ico"
	}
	fm["touchIconURL"] = func() string {
		if u := assets.IconURL(r.cfg.Favicon, theme); u != "" {
			return u
		}
		return "/apple-touch-icon.png"
	}
	return fm
}

// themeFile returns the URL of a file shipped by the theme, or empty if
// there is no theme or it doesn't ship that file.
func themeFile(theme, file string) string {
	if !assets.ThemeHasFile(theme, file) {
		return ""
	}
	return "/" + path.Join(assets.ThemeDir, theme, file)
}

// parseHTML parses the base HTML templates, then overrides them with the
// theme's views/*.html, matched by file name.
func (r *Renderer) parseHTML(theme string) (*template.Template, error) {
	t := template.New("").Funcs(r.htmlFuncMap(theme))
	var err error
	if r.cfg.EmbedAssets {
		// Use embedded FS (default for production deployments)
		if t, err = t.ParseFS(viewsFS, "views/*.html"); err != nil {
			return nil, err
		}
		if theme == "" {
			return t, nil
		}
		pattern := path.Join(assets.ThemeDir, theme, "views", "*.html")
		if matches, _ := fs.Glob(assets.StaticFS, pattern); len(matches) == 0 {
			return t, nil
		}
		return t.ParseFS(assets.StaticFS, pattern)
	}
	// Parse from local filesystem for hot-reload during development
	// Assumes running from project root
	if t, err = t.ParseGlob("internal/templates/views/*.html"); err != nil {
		return nil, err
	}
	if theme == "" {
		return t, nil
	}
	pattern := filepath.Join("internal", "assets", assets.ThemeDir, theme, "views", "*.html")
	if matches, _ := filepath.Glob(pattern); len(matches) == 0 {
		return t, nil
	}
	return t.ParseGlob(pattern)
}

func (r *Renderer) parseTemplates() error {
	html := map[string]*template.Template{}
	for _, theme := range r.themes() {
		t, err := r.parseHTML(theme)
		if err != nil {
			return fmt.Errorf("theme %q: %w", theme, err)
		}
		html[theme] = t
	}

	var xmlTmpls *texttemplate.Template
	var err error
	if r.cfg.EmbedAssets {
		xmlTmpls, err = texttemplate.New("").Funcs(textTemplateFuncs).ParseFS(viewsFS, "views/*.xml")
	} else {
		xmlTmpls, err = texttemplate.New("").Funcs(textTemplateFuncs).ParseGlob("internal/templates/views/*.xml")
	}
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.html, r.xmlTmpls = html, xmlTmpls
	r.mu.Unlock()
	return nil
}

// execute renders name with the theme carried by ctx.
func (r *Renderer) execute(ctx context.Context, w io.Writer, name string, data interface{}) error {
	if !r.cfg.EmbedAssets {
		// Re-parse on every request when using filesystem (for hot-reload)
		if err := r.parseTemplates(); err != nil {
			return err
		}
	}

	r.mu.RLock()
	xmlTmpls := r.xmlTmpls
	htmlTmpls, ok := r.html[r.Theme(ctx)]
	if !ok {
		htmlTmpls = r.html[r.cfg.Theme]
	}
	r.mu.RUnlock()

	if strings.HasSuffix(name, ".xml") {
		return xmlTmpls.ExecuteTemplate(w, name, data)
	}
	return htmlTmpls.ExecuteTemplate(w, name, data)
}

// Render writes the named template using the theme carried by ctx (see
// UserTheme), or the configured theme.
func (r *Renderer) Render(ctx context.Context, w io.Writer, name string, data interface{}) error {
	return r.execute(ctx, w, name, data)
}

// RenderToString is Render into a string.
func (r *Renderer) RenderToString(ctx context.Context, name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := r.execute(ctx, &buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
