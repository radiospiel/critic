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

// VSCodeExtensionBytes returns the embedded VS Code extension (.vsix) bytes.
// Returns an error if the extension was not embedded at build time.
func VSCodeExtensionBytes() ([]byte, error) {
	return fs.ReadFile(embeddedFS, "dist/extensions/critic-vscode.vsix")
}
