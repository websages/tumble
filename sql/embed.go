package sql

import "embed"

//go:embed mysql/*.sql sqlite/*.sql
var MigrationFS embed.FS
