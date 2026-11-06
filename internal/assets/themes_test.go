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
