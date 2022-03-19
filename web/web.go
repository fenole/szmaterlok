// Package web contains static web assets such as html files,
// js scripts and css style sheets.
package web

import "embed"

// Assets is a filesystem with static files for szmaterlok
// web server embedded into binary.
//
//go:embed assets
var Assets embed.FS

// UI is a filesystem with html templates.
//
//go:embed ui
var UI embed.FS
