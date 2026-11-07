package assets

import (
	"slices"
	"testing"
)

func TestThemes(t *testing.T) {
	themes := Themes()
	for _, want := range []string{"broadsheet", "corkboard", "irc"} {
		if !slices.Contains(themes, want) {
			t.Errorf("expected theme %q in %v", want, themes)
		}
	}
}

func TestThemeExists(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"broadsheet", true},
		{"corkboard", true},
		{"irc", true},
		{"", false},
		{"nope", false},
		{"..", false},
		{"../css", false},
		{"irc/views", false},
		{"irc.css", false},
	}
	for _, tt := range tests {
		if got := ThemeExists(tt.name); got != tt.want {
			t.Errorf("ThemeExists(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestThemeHasFile(t *testing.T) {
	if !ThemeHasFile("irc", "theme.js") {
		t.Error("expected irc to ship theme.js")
	}
	if ThemeHasFile("irc", "nope.js") {
		t.Error("expected nope.js to be missing")
	}
}

func TestIconURL(t *testing.T) {
	tests := []struct {
		name, favicon, theme, want string
	}{
		{"built-in when nothing is set", "", "", ""},
		{"built-in when theme has no icon", "", "irc", ""},
		{"theme icon", "", "corkboard", "/themes/corkboard/favicon.png"},
		{"configured icon wins over theme", "https://example.com/i.png", "corkboard", "https://example.com/i.png"},
		{"configured icon without theme", "/img/me.png", "", "/img/me.png"},
	}
	for _, tt := range tests {
		if got := IconURL(tt.favicon, tt.theme); got != tt.want {
			t.Errorf("%s: IconURL(%q, %q) = %q, want %q", tt.name, tt.favicon, tt.theme, got, tt.want)
		}
	}
}
