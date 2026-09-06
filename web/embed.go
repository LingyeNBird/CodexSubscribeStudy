// Package web embeds the Vue SPA in the single Go binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

func Assets() fs.FS {
	content, err := fs.Sub(files, "dist")
	if err != nil {
		panic(err)
	}
	return content
}
