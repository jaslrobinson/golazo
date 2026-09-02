// Package assets provides embedded static assets for the golazo application.
package assets

import (
	"embed"
)

// Logo is the golazo logo PNG image, embedded at compile time.
// Used for desktop notifications on Linux and Windows.
//
//go:embed golazo-logo.png
var Logo []byte

// HelmetsFS contains curated, background-removed team helmet artwork,
// generated offline by scripts/helmets/generate.py and rendered by
// internal/ui/helmet. Filenames are ESPN's numeric team ID (e.g. "213.png").
//
//go:embed helmets/*.png
var HelmetsFS embed.FS
