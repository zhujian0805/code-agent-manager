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

// Home returns the user's home directory. The HOME environment variable wins so
// tests and callers can intentionally isolate filesystem behavior on all
// platforms, including Windows where os.UserHomeDir does not follow HOME.
func Home() string {
	if dir := os.Getenv("HOME"); dir != "" {
		return dir
	}
	if dir, err := os.UserHomeDir(); err == nil && dir != "" {
		return dir
	}
	return ""
}

func joinHome(path string) string {
	home := Home()
	if strings.HasPrefix(home, "/") && !strings.Contains(home, "\\") {
		path = strings.TrimPrefix(path, "/")
		if path == "" {
			return strings.TrimRight(home, "/")
		}
		return strings.TrimRight(home, "/") + "/" + path
	}
	return filepath.Join(home, path)
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
	if suffix, ok := strings.CutPrefix(path, "~/"); ok {
		return joinHome(suffix)
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
