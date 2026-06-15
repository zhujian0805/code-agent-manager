package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Empty store reports "No skills installed".
func TestSkillListWhenEmpty(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "skill", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No skills installed") {
		t.Fatalf("missing empty state:\n%s", stdout)
	}
}

// Add → list → show → remove round-trip via the entity store.
func TestSkillAddListShowRemoveRoundTrip(t *testing.T) {
	isolatedHome(t)
	body := writeTempFile(t, "skill content")
	if _, _, code := execute(t, "skill", "add", "deep-research", "-f", body, "--description", "Deep research skill"); code != 0 {
		t.Fatalf("add exit = %d", code)
	}
	stdout, _, _ := execute(t, "skill", "list")
	if !strings.Contains(stdout, "deep-research") {
		t.Fatalf("list missing skill:\n%s", stdout)
	}
	stdout, _, code := execute(t, "skill", "show", "deep-research")
	if code != 0 || !strings.Contains(stdout, "deep-research") {
		t.Fatalf("show code=%d stdout=%s", code, stdout)
	}
	stdout, _, code = execute(t, "skill", "remove", "deep-research")
	if code != 0 || !strings.Contains(stdout, "Removed deep-research") {
		t.Fatalf("remove code=%d stdout=%s", code, stdout)
	}
}

// `install --app claude` creates the per-app skill directory + SKILL.md.
func TestSkillInstallCreatesSkillDirectoryWithMarkdown(t *testing.T) {
	home := isolatedHome(t)
	body := writeTempFile(t, "skill body")
	if _, _, code := execute(t, "skill", "add", "deep-research", "-f", body); code != 0 {
		t.Fatalf("seed add exit = %d", code)
	}
	stdout, _, code := execute(t, "skill", "install", "deep-research", "--app", "claude")
	if code != 0 {
		t.Fatalf("install exit = %d", code)
	}
	wantDir := filepath.Join(home, ".claude", "skills", "deep-research")
	if !strings.Contains(stdout, wantDir) {
		t.Fatalf("install output missing dir:\n%s", stdout)
	}
	data, err := os.ReadFile(filepath.Join(wantDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("SKILL.md missing: %v", err)
	}
	if string(data) != "skill body" {
		t.Fatalf("SKILL.md = %q", data)
	}
}

// `uninstall --app` removes the per-app directory.
func TestSkillUninstallRemovesDirectory(t *testing.T) {
	home := isolatedHome(t)
	if _, _, code := execute(t, "skill", "add", "demo", "-f", writeTempFile(t, "body")); code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
	if _, _, code := execute(t, "skill", "install", "demo", "--app", "claude"); code != 0 {
		t.Fatalf("install exit = %d", code)
	}
	stdout, _, code := execute(t, "skill", "uninstall", "demo", "--app", "claude")
	if code != 0 {
		t.Fatalf("uninstall exit = %d", code)
	}
	if !strings.Contains(stdout, "Uninstalled demo") {
		t.Fatalf("output:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "demo")); !os.IsNotExist(err) {
		t.Fatalf("directory still exists: %v", err)
	}

	stdout, _, code = execute(t, "skill", "uninstall", "demo", "--app", "claude")
	if code != 0 {
		t.Fatalf("second uninstall exit = %d", code)
	}
	if !strings.Contains(stdout, "Not installed for claude") {
		t.Fatalf("second uninstall message:\n%s", stdout)
	}
}

// `cam skill installed` walks per-app paths and lists what's on disk.
func TestSkillInstalledListsAcrossApps(t *testing.T) {
	home := isolatedHome(t)
	if _, _, code := execute(t, "skill", "add", "x", "-f", writeTempFile(t, "body")); code != 0 {
		t.Fatal("seed failed")
	}
	if _, _, code := execute(t, "skill", "install", "x", "--app", "claude"); code != 0 {
		t.Fatal("install failed")
	}
	stdout, _, code := execute(t, "skill", "installed")
	if code != 0 {
		t.Fatalf("installed exit = %d", code)
	}
	if !strings.Contains(stdout, "claude") || !strings.Contains(stdout, "x") {
		t.Fatalf("expected claude + x in:\n%s", stdout)
	}
	_ = home
}

// `cam s` alias works.
func TestSkillAliasS(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "s", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No skills installed") {
		t.Fatalf("alias output: %s", stdout)
	}
}
