package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

// --- list ------------------------------------------------------------------

func TestPluginListWhenEmpty(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "plugin", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No plugins installed across agents") {
		t.Fatalf("missing empty state:\n%s", stdout)
	}
}

func TestPluginListShowsInstalled(t *testing.T) {
	home := isolatedHome(t)
	installEntityToApp(t, home, entities.KindPlugin, "demo-plugin", `{"name":"demo"}`, "claude")
	stdout, _, code := execute(t, "plugin", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "demo-plugin") {
		t.Fatalf("list missing plugin:\n%s", stdout)
	}
	if !strings.Contains(stdout, "claude") {
		t.Fatalf("list missing app name:\n%s", stdout)
	}
}

// --- search ----------------------------------------------------------------

func TestPluginSearchFindsMatch(t *testing.T) {
	isolatedHome(t)
	seedEntity(t, entities.KindPlugin, "demo-plugin", "{}", "Demo")
	seedEntity(t, entities.KindPlugin, "other-plugin", "{}", "Other")
	stdout, _, code := execute(t, "plugin", "search", "demo", "--local")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "demo-plugin") {
		t.Fatalf("search missing match:\n%s", stdout)
	}
}

// --- install ---------------------------------------------------------------

func TestPluginInstallWritesManifestJSON(t *testing.T) {
	home := isolatedHome(t)
	seedEntity(t, entities.KindPlugin, "demo", `{"name":"demo"}`, "")
	stdout, _, code := execute(t, "plugin", "install", "demo", "--app", "claude")
	if code != 0 {
		t.Fatalf("install exit = %d", code)
	}
	wantDir := filepath.Join(home, ".claude", "plugins", "demo")
	if !strings.Contains(stdout, wantDir) {
		t.Fatalf("install output missing dir:\n%s", stdout)
	}
	data, err := os.ReadFile(filepath.Join(wantDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.json missing: %v", err)
	}
	if string(data) != `{"name":"demo"}` {
		t.Fatalf("manifest.json = %q", data)
	}
}

func TestPluginInstallEmptyContentDefaultsToEmptyJSON(t *testing.T) {
	home := isolatedHome(t)
	seedEntity(t, entities.KindPlugin, "demo", "", "")
	if _, _, code := execute(t, "plugin", "install", "demo", "--app", "claude"); code != 0 {
		t.Fatal("install failed")
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", "plugins", "demo", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "{}" {
		t.Fatalf("expected {} manifest, got %q", data)
	}
}

// --- alias -----------------------------------------------------------------

func TestPluginAliasPl(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "pl", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No plugins installed across agents") {
		t.Fatalf("alias output: %s", stdout)
	}
}

// --- Claude-specific plugin listing ----------------------------------------

func TestPluginListClaudeUsesInstalledPluginsJSON(t *testing.T) {
	home := isolatedHome(t)

	// Create the Claude plugins directory with metadata files.
	pluginsDir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write installed_plugins.json
	installedJSON := `{
		"version": 2,
		"plugins": {
			"security-guidance@official-mp": [
				{"scope": "user", "installPath": "/tmp/test", "version": "1.0.0"}
			],
			"playwright@official-mp": [
				{"scope": "user", "installPath": "/tmp/test2", "version": "2.0.0"}
			]
		}
	}`
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(installedJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write known_marketplaces.json
	marketplacesJSON := `{
		"official-mp": {
			"source": {"source": "github", "repo": "anthropics/claude-plugins-official"},
			"installLocation": "/tmp/mp",
			"lastUpdated": "2026-01-01T00:00:00Z"
		}
	}`
	if err := os.WriteFile(filepath.Join(pluginsDir, "known_marketplaces.json"), []byte(marketplacesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := execute(t, "plugin", "list")
	if code != 0 {
		t.Fatalf("exit = %d, output: %s", code, stdout)
	}

	// Should show installed plugins from metadata, not directory scan.
	if !strings.Contains(stdout, "Installed plugins") {
		t.Fatalf("missing 'Installed plugins' header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "playwright@official-mp") {
		t.Fatalf("missing playwright plugin:\n%s", stdout)
	}
	if !strings.Contains(stdout, "security-guidance@official-mp") {
		t.Fatalf("missing security-guidance plugin:\n%s", stdout)
	}

	// Should show marketplaces.
	if !strings.Contains(stdout, "Marketplaces") {
		t.Fatalf("missing 'Marketplaces' header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "official-mp") {
		t.Fatalf("missing marketplace name:\n%s", stdout)
	}
	if !strings.Contains(stdout, "anthropics/claude-plugins-official") {
		t.Fatalf("missing marketplace repo:\n%s", stdout)
	}

	// Should NOT show old-style directory entries like "cache", "data", "marketplaces".
	if strings.Contains(stdout, "\n  cache\n") {
		t.Fatalf("should not list 'cache' as a plugin:\n%s", stdout)
	}
	if strings.Contains(stdout, "\n  marketplaces\n") {
		t.Fatalf("should not list 'marketplaces' as a plugin:\n%s", stdout)
	}
}

func TestPluginListClaudeShowsEnabledStatus(t *testing.T) {
	home := isolatedHome(t)

	pluginsDir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	installedJSON := `{
		"version": 2,
		"plugins": {
			"my-plugin@my-mp": [{"scope":"user","version":"1.0.0"}],
			"disabled-plugin@my-mp": [{"scope":"user","version":"1.0.0"}]
		}
	}`
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(installedJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write settings.json with enabled/disabled status.
	settingsJSON := `{
		"enabledPlugins": {
			"my-plugin@my-mp": true,
			"disabled-plugin@my-mp": false
		}
	}`
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(settingsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := execute(t, "plugin", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}

	if !strings.Contains(stdout, "disabled") {
		t.Fatalf("missing disabled status:\n%s", stdout)
	}
	if !strings.Contains(stdout, "enabled") {
		t.Fatalf("missing enabled status:\n%s", stdout)
	}
}

func TestPluginListNonClaudeFallsBackToDirScan(t *testing.T) {
	home := isolatedHome(t)
	// Install a plugin to codex (not Claude) — should still use dir scan.
	installEntityToApp(t, home, entities.KindPlugin, "my-codex-plugin", `{"name":"test"}`, "codex")
	stdout, _, code := execute(t, "plugin", "list", "--app", "codex")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "my-codex-plugin") {
		t.Fatalf("codex should still use dir scan:\n%s", stdout)
	}
}
