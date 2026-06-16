package cli

// Claude-specific plugin listing that reads the metadata files Claude Code
// maintains (installed_plugins.json, known_marketplaces.json, settings.json)
// instead of scanning directories — matching how `claude plugin list` works.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

// ---------- JSON models matching Claude Code's on-disk format ----------------

// claudeInstalledPlugins matches ~/.claude/plugins/installed_plugins.json.
type claudeInstalledPlugins struct {
	Version int                              `json:"version"`
	Plugins map[string][]claudePluginInstall `json:"plugins"`
}

// claudePluginInstall is one installation entry for a plugin@marketplace key.
type claudePluginInstall struct {
	Scope        string `json:"scope"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha"`
}

// claudeMarketplace matches one entry in ~/.claude/plugins/known_marketplaces.json.
type claudeMarketplace struct {
	Source          claudeMarketplaceSource `json:"source"`
	InstallLocation string                  `json:"installLocation"`
	LastUpdated     string                  `json:"lastUpdated"`
}

type claudeMarketplaceSource struct {
	Source string `json:"source"`         // "github" or "git"
	Repo   string `json:"repo,omitempty"` // for github sources
	URL    string `json:"url,omitempty"`  // for git sources
	Ref    string `json:"ref,omitempty"`  // branch for git sources
}

// ---------- listing logic ----------------------------------------------------

// listClaudePlugins reads Claude Code's metadata files and prints installed
// plugins and marketplaces — matching the information `claude plugin list` and
// `claude plugin marketplace list` provide.
func listClaudePlugins(out io.Writer, pluginsDir string) (pluginCount, marketplaceCount int) {
	// 1. Read installed_plugins.json
	installedPath := filepath.Join(pluginsDir, "installed_plugins.json")
	installed := readClaudeInstalledPlugins(installedPath)

	// 2. Read known_marketplaces.json
	marketplacesPath := filepath.Join(pluginsDir, "known_marketplaces.json")
	marketplaces := readClaudeKnownMarketplaces(marketplacesPath)

	// 3. Read settings.json for enabled status
	homeDir := filepath.Dir(pluginsDir) // ~/.claude
	settingsPath := filepath.Join(homeDir, "settings.json")
	enabledPlugins := readClaudeEnabledPlugins(settingsPath)

	// 4. Print installed plugins
	if len(installed.Plugins) > 0 {
		// Sort plugin keys for deterministic output.
		keys := make([]string, 0, len(installed.Plugins))
		for k := range installed.Plugins {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Fprintf(out, "claude — Installed plugins (%d):\n", len(keys))
		for _, key := range keys {
			installs := installed.Plugins[key]
			if len(installs) == 0 {
				continue
			}
			inst := installs[0] // take the first installation

			version := inst.Version
			if version == "" {
				version = "unknown"
			}
			scope := inst.Scope
			if scope == "" {
				scope = "user"
			}

			// Determine enabled status from settings.json
			status := "enabled"
			// enabledPlugins is keyed by plugin@marketplace and the value is
			// true (enabled) or false (disabled).
			if enabled, ok := enabledPlugins[key]; ok && !enabled {
				status = "disabled"
			}

			fmt.Fprintf(out, "  %-45s %-18s %-8s %s\n", key, version, scope, status)
		}
		pluginCount = len(keys)
	} else {
		fmt.Fprintln(out, "claude — No plugins installed")
	}

	// 5. Print marketplaces
	if len(marketplaces) > 0 {
		fmt.Fprintln(out)

		// Sort marketplace names for deterministic output.
		mpNames := make([]string, 0, len(marketplaces))
		for k := range marketplaces {
			mpNames = append(mpNames, k)
		}
		sort.Strings(mpNames)

		fmt.Fprintf(out, "claude — Marketplaces (%d):\n", len(mpNames))
		for _, name := range mpNames {
			mp := marketplaces[name]
			source := formatMarketplaceSource(mp.Source)
			fmt.Fprintf(out, "  %-45s %s\n", name, source)
		}
		marketplaceCount = len(mpNames)
	}

	return pluginCount, marketplaceCount
}

// formatMarketplaceSource returns a display string for a marketplace source.
func formatMarketplaceSource(src claudeMarketplaceSource) string {
	switch src.Source {
	case "github":
		if src.Repo != "" {
			return "GitHub (" + src.Repo + ")"
		}
		return "GitHub"
	case "git":
		if src.URL != "" {
			s := "Git (" + src.URL
			if src.Ref != "" {
				s += "@" + src.Ref
			}
			s += ")"
			return s
		}
		return "Git"
	}
	return src.Source
}

// ---------- file readers -----------------------------------------------------

func readClaudeInstalledPlugins(path string) claudeInstalledPlugins {
	var out claudeInstalledPlugins
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	_ = json.Unmarshal(data, &out)
	return out
}

func readClaudeKnownMarketplaces(path string) map[string]claudeMarketplace {
	out := make(map[string]claudeMarketplace)
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	_ = json.Unmarshal(data, &out)
	return out
}

func readClaudeEnabledPlugins(settingsPath string) map[string]bool {
	out := make(map[string]bool)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return out
	}
	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return out
	}
	return settings.EnabledPlugins
}

// isClaudePluginApp returns true if the app uses the Claude-style plugin
// metadata model (installed_plugins.json + known_marketplaces.json) rather
// than the simple directory-scan model.
func isClaudePluginApp(app, dest string) bool {
	if app != "claude" {
		return false
	}
	// Verify the metadata files exist — if they don't, fall back to dir scan.
	resolved := pathutil.Expand(dest)
	installedPath := filepath.Join(resolved, "installed_plugins.json")
	if _, err := os.Stat(installedPath); err != nil {
		return false
	}
	return true
}

// claudePluginHasMarketplace returns whether a Claude plugin key contains
// a marketplace reference (name@marketplace format).
func claudePluginHasMarketplace(key string) (name, marketplace string) {
	parts := strings.SplitN(key, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return key, ""
}
