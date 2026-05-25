package embed

import "embed"

//go:embed all:dist
var distFS embed.FS
