package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentListWhenEmpty(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "agent", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No agents installed") {
		t.Fatalf("missing empty state:\n%s", stdout)
	}
}

func TestAgentAddListShowRemoveRoundTrip(t *testing.T) {
	isolatedHome(t)
	body := writeTempFile(t, "agent content")
	if _, _, code := execute(t, "agent", "add", "code-reviewer", "-f", body, "--description", "Code review agent"); code != 0 {
		t.Fatalf("add exit = %d", code)
	}
	stdout, _, _ := execute(t, "agent", "list")
	if !strings.Contains(stdout, "code-reviewer") {
		t.Fatalf("list missing agent:\n%s", stdout)
	}
	stdout, _, code := execute(t, "agent", "show", "code-reviewer")
	if code != 0 || !strings.Contains(stdout, "code-reviewer") {
		t.Fatalf("show code=%d stdout=%s", code, stdout)
	}
	stdout, _, code = execute(t, "agent", "remove", "code-reviewer")
	if code != 0 || !strings.Contains(stdout, "Removed code-reviewer") {
		t.Fatalf("remove code=%d stdout=%s", code, stdout)
	}
}

func TestAgentInstallCreatesAgentDirectoryWithMarkdown(t *testing.T) {
	home := isolatedHome(t)
	body := writeTempFile(t, "agent body")
	if _, _, code := execute(t, "agent", "add", "code-reviewer", "-f", body); code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
	stdout, _, code := execute(t, "agent", "install", "code-reviewer", "--app", "claude")
	if code != 0 {
		t.Fatalf("install exit = %d", code)
	}
	wantDir := filepath.Join(home, ".claude", "agents", "code-reviewer")
	if !strings.Contains(stdout, wantDir) {
		t.Fatalf("install output missing dir:\n%s", stdout)
	}
	data, err := os.ReadFile(filepath.Join(wantDir, "AGENT.md"))
	if err != nil {
		t.Fatalf("AGENT.md missing: %v", err)
	}
	if string(data) != "agent body" {
		t.Fatalf("AGENT.md = %q", data)
	}
}

func TestAgentUninstallRemovesDirectory(t *testing.T) {
	home := isolatedHome(t)
	if _, _, code := execute(t, "agent", "add", "x", "-f", writeTempFile(t, "body")); code != 0 {
		t.Fatal("seed failed")
	}
	if _, _, code := execute(t, "agent", "install", "x", "--app", "claude"); code != 0 {
		t.Fatal("install failed")
	}
	if _, _, code := execute(t, "agent", "uninstall", "x", "--app", "claude"); code != 0 {
		t.Fatal("uninstall failed")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "x")); !os.IsNotExist(err) {
		t.Fatalf("directory still present: %v", err)
	}
}

func TestAgentAliasAg(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "ag", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No agents installed") {
		t.Fatalf("alias output: %s", stdout)
	}
}
