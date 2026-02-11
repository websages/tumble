package data

import "embed"

//go:embed schema.sqlite schema.mysql
var SchemaFS embed.FS
