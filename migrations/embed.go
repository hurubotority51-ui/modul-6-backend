package migrations

import "embed"

// FS embeds the ordered PostgreSQL migration scripts for the migration command.
//
//go:embed *.sql
var FS embed.FS
