package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/chat2anyllm/code-agent-manager/internal/desktop"
)

var version = "dev"

func main() {
	services := desktop.NewServices(version, "")
	if len(os.Args) > 1 && os.Args[1] == "--services" {
		_ = json.NewEncoder(os.Stdout).Encode(map[string][]string{
			"services": {"app", "providers", "mcp", "entities", "tools", "doctor", "config", "launch"},
		})
		return
	}
	fmt.Printf("cam-desktop %s\n", services.App.Version())
	fmt.Println("Desktop services are available for Wails registration.")
}
