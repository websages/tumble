package assets

import (
	"io/fs"
	"path"
	"sort"
	"strings"
)

// ThemeDir holds pluggable themes. Each theme is a directory,
// themes/<name>/, containing:
//
//	theme.css    required; loaded after screen.css
//	theme.js     optional; loaded with defer
//	views/*.html optional; override base templates with the same name
//	favicon.png  optional; site icon and Apple touch icon (180x180)
//
// Any other files (images, fonts) are served at /themes/<name>/.
const ThemeDir = "themes"

// Themes returns the names of the available themes, sorted.
func Themes() []string {
	matches, _ := fs.Glob(StaticFS, ThemeDir+"/*/theme.css")
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, path.Base(path.Dir(m)))
	}
	sort.Strings(names)
	return names
}

// ThemeExists reports whether name is an available theme.
func ThemeExists(name string) bool {
	return ThemeHasFile(name, "theme.css")
}

// ThemeHasFile reports whether the named theme ships the given file.
func ThemeHasFile(name, file string) bool {
	if name == "" || strings.ContainsAny(name, "/\\.") {
		return false
	}
	_, err := fs.Stat(StaticFS, path.Join(ThemeDir, name, file))
	return err == nil
}

// ThemeIconFile is the optional icon a theme can ship.
const ThemeIconFile = "favicon.png"

// IconURL returns the site icon to use in place of the built-in
// favicon.ico and apple-touch-icon.png: the configured favicon if set,
// otherwise the theme's favicon.png if it ships one. It returns "" when
// the built-in icons should be used.
func IconURL(favicon, theme string) string {
	if favicon != "" {
		return favicon
	}
	if ThemeHasFile(theme, ThemeIconFile) {
		return "/" + path.Join(ThemeDir, theme, ThemeIconFile)
	}
	return ""
}
