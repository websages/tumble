package assets

import "embed"

//go:embed css img buttons favicon.ico openapi.json
var StaticFS embed.FS
