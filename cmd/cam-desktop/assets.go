package main

import (
	"embed"
	"io/fs"
)

//go:embed all:webui
var embeddedWebUI embed.FS

func desktopAssets() (fs.FS, error) {
	if _, err := fs.Stat(embeddedWebUI, "webui/dist/index.html"); err == nil {
		return fs.Sub(embeddedWebUI, "webui/dist")
	}
	return fs.Sub(embeddedWebUI, "webui/fallback")
}
