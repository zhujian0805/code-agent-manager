package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/chat2anyllm/code-agent-manager/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

var version = "dev"

func main() {
	services := desktop.NewServices(version, "")
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--services":
			_ = json.NewEncoder(os.Stdout).Encode(map[string][]string{
				"services": {"app", "providers", "mcp", "entities", "tools", "doctor", "config", "launch"},
			})
			return
		case "--version", "version":
			fmt.Printf("cam-desktop %s\n", services.App.Version())
			return
		}
	}

	if err := runDesktopApp(services); err != nil {
		log.Fatal(err)
	}
}

func runDesktopApp(services desktop.Services) error {
	assets, err := desktopAssets()
	if err != nil {
		return fmt.Errorf("load desktop assets: %w", err)
	}

	return wails.Run(&options.App{
		Title:     "Code Agent Manager",
		Width:     1180,
		Height:    780,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			services.App,
			services.Providers,
			services.MCP,
			services.Entities,
			services.Tools,
			services.Doctor,
			services.Config,
			services.Launch,
		},
	})
}
