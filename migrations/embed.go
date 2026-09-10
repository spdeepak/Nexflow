package migrations

import "embed"

// FS is used for the desktop applications SQLite DB
//
//go:embed *.sql
var FS embed.FS
