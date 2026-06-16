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

	fmt.Fprintln(os.Stderr, "cam-desktop Wails runtime has been replaced by the Tauri desktop shell.")
	fmt.Fprintln(os.Stderr, "Use `cargo tauri dev --manifest-path src-tauri/Cargo.toml` for the desktop app or `cam-sidecar` for the Go sidecar.")
	os.Exit(1)
}
