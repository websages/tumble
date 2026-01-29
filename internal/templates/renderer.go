package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
	texttemplate "text/template"

	"tumble/internal/config"
)

//go:embed views/*.html views/*.xml
var viewsFS embed.FS

type Renderer struct {
	cfg       *config.Config
	htmlTmpls *template.Template
	xmlTmpls  *texttemplate.Template
}

// templateFuncs provides helper functions for templates
var templateFuncs = template.FuncMap{
	// irclinkURL builds a click-tracking URL for IRC links
	"irclinkURL": func(baseURL string, id int) string {
		return fmt.Sprintf("%s/irclink/?%d", baseURL, id)
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
	"irclinkURL": func(baseURL string, id int) string {
		return fmt.Sprintf("%s/irclink/?%d", baseURL, id)
	},
	"truncate": func(s string, max int) string {
		if len(s) > max {
			return s[:max] + "..."
		}
		return s
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

func (r *Renderer) parseTemplates() error {
	var err error
	// Determine source: Embed or Filesystem
	if r.cfg.Mode == "development" {
		// Parse from local filesystem for reload
		// Assuming running from project root
		r.htmlTmpls, err = template.New("").Funcs(templateFuncs).ParseGlob("internal/templates/views/*.html")
		if err != nil {
			return err
		}
		r.xmlTmpls, err = texttemplate.New("").Funcs(textTemplateFuncs).ParseGlob("internal/templates/views/*.xml")
		if err != nil {
			return err
		}
	} else {
		// Use embedded FS for production
		r.htmlTmpls, err = template.New("").Funcs(templateFuncs).ParseFS(viewsFS, "views/*.html")
		if err != nil {
			return err
		}
		r.xmlTmpls, err = texttemplate.New("").Funcs(textTemplateFuncs).ParseFS(viewsFS, "views/*.xml")
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	if r.cfg.Mode == "development" {
		// Re-parse on every request
		if err := r.parseTemplates(); err != nil {
			return err
		}
	}

	if strings.HasSuffix(name, ".xml") {
		return r.xmlTmpls.ExecuteTemplate(w, name, data)
	}
	return r.htmlTmpls.ExecuteTemplate(w, name, data)
}

func (r *Renderer) RenderToString(name string, data interface{}) (string, error) {
	if r.cfg.Mode == "development" {
		if err := r.parseTemplates(); err != nil {
			return "", err
		}
	}

	var buf bytes.Buffer
	var err error
	if strings.HasSuffix(name, ".xml") {
		err = r.xmlTmpls.ExecuteTemplate(&buf, name, data)
	} else {
		err = r.htmlTmpls.ExecuteTemplate(&buf, name, data)
	}

	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
