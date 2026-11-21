package templates

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tumble/internal/config"
)

func renderMasthead(t *testing.T, theme string) string {
	t.Helper()
	r, err := NewRenderer(&config.Config{EmbedAssets: true, SiteName: "tumblefish", Theme: theme})
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	out, err := r.RenderToString(context.Background(), "masthead", nil)
	if err != nil {
		t.Fatalf("render masthead: %v", err)
	}
	return out
}

func TestThemeOverridesViews(t *testing.T) {
	if out := renderMasthead(t, ""); strings.Contains(out, "bs-nameplate") {
		t.Errorf("default masthead should not use the broadsheet override:\n%s", out)
	}
	if out := renderMasthead(t, "broadsheet"); !strings.Contains(out, "bs-nameplate") {
		t.Errorf("broadsheet masthead override not applied:\n%s", out)
	}
	// corkboard ships no views, so the base masthead is used
	if out := renderMasthead(t, "corkboard"); !strings.Contains(out, `id="masthead"`) {
		t.Errorf("corkboard should fall back to the base masthead:\n%s", out)
	}
}

func TestThemeFiles(t *testing.T) {
	if got := themeFile("irc", "theme.css"); got != "/themes/irc/theme.css" {
		t.Errorf("theme.css URL = %q", got)
	}
	if got := themeFile("irc", "theme.js"); got != "/themes/irc/theme.js" {
		t.Errorf("theme.js URL = %q", got)
	}
	if got := themeFile("corkboard", "theme.js"); got != "" {
		t.Errorf("corkboard has no theme.js, got %q", got)
	}
	if got := themeFile("", "theme.css"); got != "" {
		t.Errorf("default look has no theme.css, got %q", got)
	}
}

// themeFor runs a request with the given theme cookie through UserTheme
// and returns the theme the renderer would use for it.
func themeFor(t *testing.T, r *Renderer, cookie string) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: ThemeCookie, Value: cookie})
	}
	var got string
	r.UserTheme(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		got = r.Theme(req.Context())
	})).ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestUserThemeDisabled(t *testing.T) {
	r, err := NewRenderer(&config.Config{EmbedAssets: true, Theme: "corkboard"})
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	if got := themeFor(t, r, "irc"); got != "corkboard" {
		t.Errorf("cookie should be ignored when disabled, got %q", got)
	}
}

func TestUserThemeEnabled(t *testing.T) {
	r, err := NewRenderer(&config.Config{EmbedAssets: true, Theme: "corkboard", AllowUserThemes: true})
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	tests := map[string]string{
		"":        "corkboard",
		"irc":     "irc",
		"default": "",
		"nope":    "corkboard",
		"../css":  "corkboard",
	}
	for cookie, want := range tests {
		if got := themeFor(t, r, cookie); got != want {
			t.Errorf("cookie %q: theme = %q, want %q", cookie, got, want)
		}
	}

	// Each theme renders with its own view overrides.
	ctx := WithTheme(context.Background(), "broadsheet")
	out, err := r.RenderToString(ctx, "masthead", nil)
	if err != nil {
		t.Fatalf("render masthead: %v", err)
	}
	if !strings.Contains(out, "bs-nameplate") {
		t.Errorf("broadsheet override not applied for chosen theme:\n%s", out)
	}
	out, err = r.RenderToString(context.Background(), "masthead", nil)
	if err != nil {
		t.Fatalf("render masthead: %v", err)
	}
	if strings.Contains(out, "bs-nameplate") {
		t.Errorf("configured theme should not use broadsheet override:\n%s", out)
	}
}

func TestIconsPartial(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want []string
	}{
		{"default", config.Config{}, []string{`href="/favicon.ico"`, `href="/apple-touch-icon.png"`}},
		{"theme icon", config.Config{Theme: "corkboard"}, []string{`href="/themes/corkboard/favicon.png"`}},
		{"configured icon", config.Config{Theme: "corkboard", Favicon: "https://example.com/i.png"}, []string{`href="https://example.com/i.png"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.EmbedAssets = true
			r, err := NewRenderer(&cfg)
			if err != nil {
				t.Fatalf("NewRenderer: %v", err)
			}
			var buf bytes.Buffer
			if err := r.html[cfg.Theme].ExecuteTemplate(&buf, "icons", nil); err != nil {
				t.Fatalf("render icons: %v", err)
			}
			out := buf.String()
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("icons missing %s:\n%s", w, out)
				}
			}
			if strings.Count(out, `rel="icon"`) != 1 || strings.Count(out, `rel="apple-touch-icon"`) != 1 {
				t.Errorf("expected one icon and one apple-touch-icon link:\n%s", out)
			}
		})
	}
}
