// Package pathutil contains shared filesystem path helpers used across the CLI.
//
// The helpers here intentionally have no dependencies on the rest of the CAM
// code base so that any subpackage can use them.
package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// Home returns the user's home directory.  Falls back to the $HOME environment
// variable when os.UserHomeDir fails so that tests can override behaviour via
// t.Setenv.
func Home() string {
	if dir, err := os.UserHomeDir(); err == nil && dir != "" {
		return dir
	}
	return os.Getenv("HOME")
}

// Expand resolves a leading "~" segment to the user's home directory.
//
// Inputs of "" return "" so callers can guard against missing config paths
// without an extra branch.
func Expand(path string) string {
	if path == "" {
		return ""
	}
	if path == "~" {
		return Home()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(Home(), strings.TrimPrefix(path, "~/"))
	}
	return path
}

// ConfigDir returns the directory in which CAM stores its primary config files.
// The CAM_CONFIG_DIR environment variable wins when set so tests can isolate
// changes to a temporary directory.
func ConfigDir() string {
	if dir := os.Getenv("CAM_CONFIG_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(Home(), ".config", "code-agent-manager")
}

// CacheDir returns CAM's cache directory.  Honors CAM_CACHE_DIR when set.
func CacheDir() string {
	if dir := os.Getenv("CAM_CACHE_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(Home(), ".cache", "code-agent-manager")
}

// Exists reports whether the file at path exists (and is reachable via Stat).
// Symlink targets are followed; broken symlinks return false.
func Exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
