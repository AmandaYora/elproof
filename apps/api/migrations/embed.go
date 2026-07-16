// Package migrations embeds the .sql migration files into the compiled
// binary so the single deployed image can run migrations against itself
// (`./api migrate up`) without the golang-migrate CLI or this source
// directory being present on the target host — see docs/DEPLOYMENT.md.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
