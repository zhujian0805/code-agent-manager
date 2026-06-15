package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Empty store reports "No prompts installed".
func TestPromptListWhenEmpty(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "prompt", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No prompts installed") {
		t.Fatalf("missing empty state:\n%s", stdout)
	}
}

// Add → list → show → remove round-trip.
func TestPromptAddListShowRemoveRoundTrip(t *testing.T) {
	isolatedHome(t)
	body := writeTempFile(t, "Hello\n")

	stdout, _, code := execute(t, "prompt", "add", "demo", "--description", "Test", "-f", body)
	if code != 0 {
		t.Fatalf("add exit = %d", code)
	}
	if !strings.Contains(stdout, "Added prompt demo") {
		t.Fatalf("add output:\n%s", stdout)
	}

	stdout, _, _ = execute(t, "prompt", "list")
	if !strings.Contains(stdout, "demo") {
		t.Fatalf("list missing demo:\n%s", stdout)
	}

	stdout, _, code = execute(t, "prompt", "show", "demo")
	if code != 0 {
		t.Fatalf("show exit = %d", code)
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(stdout), &entry); err != nil {
		t.Fatalf("show output not JSON: %v\n%s", err, stdout)
	}
	if entry["name"] != "demo" || entry["kind"] != "prompt" {
		t.Fatalf("show entry wrong: %v", entry)
	}

	stdout, _, code = execute(t, "prompt", "remove", "demo")
	if code != 0 || !strings.Contains(stdout, "Removed demo") {
		t.Fatalf("remove code=%d stdout=%s", code, stdout)
	}
}

// `install` requires --app and writes the rendered content into the app's path.
func TestPromptInstallWritesContentToAppPath(t *testing.T) {
	home := isolatedHome(t)
	if _, _, code := execute(t, "prompt", "add", "demo", "-f", writeTempFile(t, "prompt body")); code != 0 {
		t.Fatalf("seed add exit = %d", code)
	}

	stdout, _, code := execute(t, "prompt", "install", "demo", "--app", "claude")
	if code != 0 {
		t.Fatalf("install exit = %d", code)
	}
	if !strings.Contains(stdout, "Installed demo") || !strings.Contains(stdout, "claude") {
		t.Fatalf("install output:\n%s", stdout)
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil {
		t.Fatalf("expected CLAUDE.md: %v", err)
	}
	if string(data) != "prompt body" {
		t.Fatalf("content = %q", data)
	}
}

// Missing --app on install errors out and mentions supported apps.
func TestPromptInstallWithoutAppErrors(t *testing.T) {
	isolatedHome(t)
	if _, _, code := execute(t, "prompt", "add", "demo", "-f", writeTempFile(t, "body")); code != 0 {
		t.Fatalf("seed add exit = %d", code)
	}
	_, stderr, code := execute(t, "prompt", "install", "demo")
	if code == 0 {
		t.Fatal("expected non-zero exit without --app")
	}
	if !strings.Contains(stderr, "--app is required") {
		t.Fatalf("stderr missing --app guidance: %s", stderr)
	}
}

// Export/import round-trip works through JSON.
func TestPromptExportImportRoundTrip(t *testing.T) {
	isolatedHome(t)
	if _, _, code := execute(t, "prompt", "add", "alpha", "-f", writeTempFile(t, "a")); code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
	if _, _, code := execute(t, "prompt", "add", "beta", "-f", writeTempFile(t, "b")); code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
	exportPath := filepath.Join(t.TempDir(), "out.json")
	if _, _, code := execute(t, "prompt", "export", "-f", exportPath); code != 0 {
		t.Fatalf("export exit = %d", code)
	}
	raw, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Wipe store and reimport.
	isolatedHome(t)
	stdout, _, code := execute(t, "prompt", "import", "-f", exportPath)
	if code != 0 {
		t.Fatalf("import exit = %d", code)
	}
	if !strings.Contains(stdout, "Imported 2 prompts") {
		t.Fatalf("import output: %s", stdout)
	}
}

// Remove of a missing prompt is benign (exit 0, "Not found").
func TestPromptRemoveMissingIsBenign(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "prompt", "remove", "ghost")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Not found") {
		t.Fatalf("missing 'Not found': %s", stdout)
	}
}

// `cam p` alias works.
func TestPromptAliasP(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "p", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No prompts installed") {
		t.Fatalf("alias p output: %s", stdout)
	}
}
