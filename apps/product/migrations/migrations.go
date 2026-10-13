// Package migrations embeds the schema migrations, so the service binary
// carries its own schema and migrates the database before it serves.
package migrations

import "embed"

// FS holds every migration in this directory.
//
//go:embed *.sql
var FS embed.FS
