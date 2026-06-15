package cli_test

import (
	"strings"
	"testing"
)

// Dry-run uninstall must list the resolved tools without touching the system.
func TestUninstallDryRunForSpecificTool(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "uninstall", "gemini", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would uninstall gemini") {
		t.Fatalf("missing header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "gemini-cli") {
		t.Fatalf("missing resolved tool key:\n%s", stdout)
	}
}

// `cam uninstall all --dry-run` previews every enabled tool.
func TestUninstallAllDryRunListsEveryEnabledTool(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "uninstall", "all", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would uninstall all") {
		t.Fatalf("missing aggregate header:\n%s", stdout)
	}
	for _, want := range []string{"claude-code", "openai-codex"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing tool %q:\n%s", want, stdout)
		}
	}
}

// `cam un` alias works.
func TestUninstallAliasUn(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "un", "codex", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would uninstall codex") {
		t.Fatalf("alias output:\n%s", stdout)
	}
}

// Uninstall exposes additional flags --force/--keep-config.
func TestUninstallHelpExposesForceAndKeepConfig(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "uninstall", "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"--force", "--keep-config", "--dry-run"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help missing %q:\n%s", want, stdout)
		}
	}
}

// Unknown targets fail loud.
func TestUninstallRejectsUnknownTarget(t *testing.T) {
	isolatedHome(t)
	_, stderr, code := execute(t, "uninstall", "ghostly", "--dry-run")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "Unknown target") {
		t.Fatalf("stderr missing Unknown target: %s", stderr)
	}
}
