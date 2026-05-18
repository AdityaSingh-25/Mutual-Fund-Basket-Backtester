// Package migrations embeds the SQL schema files so they can be applied at
// startup without shipping the .sql files alongside the binary.
package migrations

import "embed"

// Files holds the SQL migration files. They are applied in lexical filename
// order, so the numeric prefixes (001_, 002_, ...) define the sequence.
//
//go:embed *.sql
var Files embed.FS
