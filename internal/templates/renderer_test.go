package templates

import (
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
	out, err := r.RenderToString("masthead", nil)
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
	r, err := NewRenderer(&config.Config{EmbedAssets: true, Theme: "irc"})
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	if got := r.themeFile("theme.css"); got != "/themes/irc/theme.css" {
		t.Errorf("theme.css URL = %q", got)
	}
	if got := r.themeFile("theme.js"); got != "/themes/irc/theme.js" {
		t.Errorf("theme.js URL = %q", got)
	}
	r.cfg.Theme = "corkboard"
	if got := r.themeFile("theme.js"); got != "" {
		t.Errorf("corkboard has no theme.js, got %q", got)
	}
}
