package templates

import (
	"bytes"
	"embed"
	"html/template"
	"io"
	"strings"
	texttemplate "text/template"
)

//go:embed views/*.html views/*.xml
var viewsFS embed.FS

type Renderer struct {
	htmlTmpls *template.Template
	xmlTmpls  *texttemplate.Template
}

func NewRenderer() (*Renderer, error) {
	htmlTmpls, err := template.ParseFS(viewsFS, "views/*.html")
	if err != nil {
		return nil, err
	}

	xmlTmpls, err := texttemplate.ParseFS(viewsFS, "views/*.xml")
	if err != nil {
		return nil, err
	}

	return &Renderer{
		htmlTmpls: htmlTmpls,
		xmlTmpls:  xmlTmpls,
	}, nil
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	if strings.HasSuffix(name, ".xml") {
		return r.xmlTmpls.ExecuteTemplate(w, name, data)
	}
	return r.htmlTmpls.ExecuteTemplate(w, name, data)
}

func (r *Renderer) RenderToString(name string, data interface{}) (string, error) {
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
