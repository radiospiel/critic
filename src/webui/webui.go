package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var embeddedFS embed.FS

// DistFS returns the embedded dist filesystem for serving static files.
func DistFS() (http.FileSystem, error) {
	distFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(distFS), nil
}
