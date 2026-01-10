package templates

import (
	"bytes"
	"embed"
	"html/template"
	"io"
)

//go:embed views/*.html views/*.xml
var viewsFS embed.FS

type Renderer struct {
	tmpls *template.Template
}

func NewRenderer() (*Renderer, error) {
	tmpls, err := template.ParseFS(viewsFS, "views/*.html", "views/*.xml")
	if err != nil {
		return nil, err
	}
	return &Renderer{tmpls: tmpls}, nil
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	return r.tmpls.ExecuteTemplate(w, name, data)
}

func (r *Renderer) RenderToString(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpls.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
