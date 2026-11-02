// Package schema embeds the goose migrations so code can apply them without the goose CLI.
package schema

import "embed"

// Migrations holds every goose migration file, named <YYYYMMDDHHMM>_<name>.sql at its root.
//
//go:embed *.sql
var Migrations embed.FS
