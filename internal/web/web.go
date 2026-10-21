package web

import "embed"

// Dist contains the embedded production build.
//
//go:embed all:dist
var Dist embed.FS
