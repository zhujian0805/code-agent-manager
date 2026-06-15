package cli_test

import (
	"strings"
	"testing"
)

// `cam upgrade <tool> --dry-run` previews the upgrade for a single target.
func TestUpgradeDryRunForSpecificTool(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "upgrade", "claude", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would upgrade claude") {
		t.Fatalf("missing dry-run header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "claude-code") {
		t.Fatalf("missing resolved tool key:\n%s", stdout)
	}
}

// `cam upgrade all --dry-run` previews the upgrade across every enabled tool.
func TestUpgradeAllDryRunCoversEveryEnabledTool(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "upgrade", "all", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would upgrade all") {
		t.Fatalf("missing 'Would upgrade all'\n%s", stdout)
	}
	for _, want := range []string{"claude-code", "openai-codex", "gemini-cli", "qwen-code"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing tool %q in upgrade list:\n%s", want, stdout)
		}
	}
}

// `cam u` alias mirrors `cam upgrade`.
func TestUpgradeAliasU(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "u", "gemini-cli", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would upgrade gemini-cli") {
		t.Fatalf("alias output:\n%s", stdout)
	}
}

// Unknown targets fail loud.
func TestUpgradeRejectsUnknownTarget(t *testing.T) {
	isolatedHome(t)
	_, stderr, code := execute(t, "upgrade", "ghostly", "--dry-run")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "Unknown target") {
		t.Fatalf("stderr missing Unknown target: %s", stderr)
	}
}

// `cam upgrade --help` advertises --dry-run + --verbose.
func TestUpgradeHelpDocumentsFlags(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "upgrade", "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"--dry-run", "--verbose"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help missing %q:\n%s", want, stdout)
		}
	}
}
