package templates

import (
	"bytes"
	"embed"
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
		r.htmlTmpls, err = template.ParseGlob("internal/templates/views/*.html")
		if err != nil {
			return err
		}
		r.xmlTmpls, err = texttemplate.ParseGlob("internal/templates/views/*.xml")
		if err != nil {
			return err
		}
	} else {
		// Use embedded FS for production
		r.htmlTmpls, err = template.ParseFS(viewsFS, "views/*.html")
		if err != nil {
			return err
		}
		r.xmlTmpls, err = texttemplate.ParseFS(viewsFS, "views/*.xml")
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
