package templates

import (
	"context"
	"net/http"

	"tumble/internal/assets"
)

// ThemeCookie holds a visitor's chosen theme when allow_user_themes is set.
const ThemeCookie = "tumble_theme"

// defaultLook is the cookie value for the base look with no theme.
const defaultLook = "default"

type themeKey struct{}

// WithTheme returns ctx carrying theme ("" for the default look).
func WithTheme(ctx context.Context, theme string) context.Context {
	return context.WithValue(ctx, themeKey{}, theme)
}

// Theme returns the theme to render for ctx: the visitor's choice if one
// was set with WithTheme, otherwise the configured theme.
func (r *Renderer) Theme(ctx context.Context) string {
	if ctx != nil {
		if theme, ok := ctx.Value(themeKey{}).(string); ok {
			return theme
		}
	}
	return r.cfg.Theme
}

// UserTheme applies the visitor's theme cookie to the request context when
// allow_user_themes is set. Unknown theme names are ignored.
func (r *Renderer) UserTheme(next http.Handler) http.Handler {
	if !r.cfg.AllowUserThemes {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if c, err := req.Cookie(ThemeCookie); err == nil {
			switch {
			case c.Value == defaultLook:
				req = req.WithContext(WithTheme(req.Context(), ""))
			case assets.ThemeExists(c.Value):
				req = req.WithContext(WithTheme(req.Context(), c.Value))
			}
		}
		next.ServeHTTP(w, req)
	})
}
