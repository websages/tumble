package assets

import (
	"slices"
	"testing"
)

func TestThemes(t *testing.T) {
	themes := Themes()
	for _, want := range []string{"aurora", "paper", "terminal"} {
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
		{"aurora", true},
		{"paper", true},
		{"terminal", true},
		{"", false},
		{"nope", false},
		{"../screen", false},
		{"themes/aurora", false},
		{"aurora.css", false},
	}
	for _, tt := range tests {
		if got := ThemeExists(tt.name); got != tt.want {
			t.Errorf("ThemeExists(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
