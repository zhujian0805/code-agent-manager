package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginListWhenEmpty(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "plugin", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No plugins installed") {
		t.Fatalf("missing empty state:\n%s", stdout)
	}
}

func TestPluginAddListShowRemoveRoundTrip(t *testing.T) {
	isolatedHome(t)
	body := writeTempFile(t, `{"hello":"world"}`)
	if _, _, code := execute(t, "plugin", "add", "demo-plugin", "-f", body, "--description", "Demo"); code != 0 {
		t.Fatalf("add exit = %d", code)
	}
	stdout, _, _ := execute(t, "plugin", "list")
	if !strings.Contains(stdout, "demo-plugin") {
		t.Fatalf("list missing plugin:\n%s", stdout)
	}
	stdout, _, code := execute(t, "plugin", "show", "demo-plugin")
	if code != 0 || !strings.Contains(stdout, "demo-plugin") {
		t.Fatalf("show code=%d stdout=%s", code, stdout)
	}
	stdout, _, code = execute(t, "plugin", "remove", "demo-plugin")
	if code != 0 || !strings.Contains(stdout, "Removed demo-plugin") {
		t.Fatalf("remove code=%d stdout=%s", code, stdout)
	}
}

func TestPluginInstallWritesManifestJSON(t *testing.T) {
	home := isolatedHome(t)
	body := writeTempFile(t, `{"name":"demo"}`)
	if _, _, code := execute(t, "plugin", "add", "demo", "-f", body); code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
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
	if _, _, code := execute(t, "plugin", "add", "demo", "-f", writeTempFile(t, "")); code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
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

func TestPluginUninstallRemovesDirectory(t *testing.T) {
	home := isolatedHome(t)
	if _, _, code := execute(t, "plugin", "add", "demo", "-f", writeTempFile(t, "{}")); code != 0 {
		t.Fatal("seed failed")
	}
	if _, _, code := execute(t, "plugin", "install", "demo", "--app", "claude"); code != 0 {
		t.Fatal("install failed")
	}
	if _, _, code := execute(t, "plugin", "uninstall", "demo", "--app", "claude"); code != 0 {
		t.Fatal("uninstall failed")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "plugins", "demo")); !os.IsNotExist(err) {
		t.Fatalf("directory still present: %v", err)
	}
}

func TestPluginAliasPl(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "pl", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No plugins installed") {
		t.Fatalf("alias output: %s", stdout)
	}
}
