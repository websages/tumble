package assets

import (
	"io/fs"
	"path"
	"sort"
	"strings"
)

// themeDir holds pluggable themes. Each theme is a single CSS file,
// css/themes/<name>.css, loaded after screen.css.
const themeDir = "css/themes"

// Themes returns the names of the available themes, sorted.
func Themes() []string {
	matches, _ := fs.Glob(StaticFS, themeDir+"/*.css")
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, strings.TrimSuffix(path.Base(m), ".css"))
	}
	sort.Strings(names)
	return names
}

// ThemeExists reports whether name is an available theme.
func ThemeExists(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\.") {
		return false
	}
	_, err := fs.Stat(StaticFS, themeDir+"/"+name+".css")
	return err == nil
}
