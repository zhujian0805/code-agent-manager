package cli_test

import (
	"strings"
	"testing"
)

// `cam install <tool> --dry-run` prints the planned install per resolved tool
// without actually executing the install_cmd, so the test stays hermetic.
func TestInstallDryRunForSpecificTool(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "install", "codex", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"Would install codex", "openai-codex"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing %q\n%s", want, stdout)
		}
	}
}

// `cam install all --dry-run` lists every enabled tool.
func TestInstallAllDryRunListsEveryEnabledTool(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "install", "all", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would install all") {
		t.Fatalf("missing aggregate header:\n%s", stdout)
	}
	for _, want := range []string{"claude-code", "openai-codex", "gemini-cli"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing tool %q:\n%s", want, stdout)
		}
	}
}

// `cam i` alias works.
func TestInstallAliasI(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "i", "claude-code", "--dry-run")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Would install claude-code") {
		t.Fatalf("alias i output:\n%s", stdout)
	}
}

// Unknown target errors out instead of running on "all".
func TestInstallRejectsUnknownTarget(t *testing.T) {
	isolatedHome(t)
	_, stderr, code := execute(t, "install", "ghostly", "--dry-run")
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown target")
	}
	if !strings.Contains(stderr, "Unknown target") {
		t.Fatalf("stderr missing Unknown target: %s", stderr)
	}
}

// Both the tools.yaml key (`openai-codex`) and its cli_command (`codex`) are
// accepted as targets, mirroring the menu and Python aliasing.
func TestInstallAcceptsCLICommandAlias(t *testing.T) {
	isolatedHome(t)
	for _, target := range []string{"openai-codex", "codex"} {
		stdout, _, code := execute(t, "install", target, "--dry-run")
		if code != 0 {
			t.Fatalf("install %s exit = %d", target, code)
		}
		if !strings.Contains(stdout, "openai-codex") {
			t.Fatalf("install %s did not resolve to openai-codex:\n%s", target, stdout)
		}
	}
}

// `cam install --help` mentions --dry-run/--verbose.
func TestInstallHelpDocumentsFlags(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "install", "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"--dry-run", "--verbose"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help missing %q:\n%s", want, stdout)
		}
	}
}
