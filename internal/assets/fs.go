package assets

import "embed"

//go:embed css img buttons favicon.ico openapi.json robots.txt apple-touch-icon.png 2202 roast
var StaticFS embed.FS
