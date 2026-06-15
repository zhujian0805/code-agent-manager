package cli_test

import (
	"strings"
	"testing"
)

// `cam version`, `cam v`, and `cam --version` all emit the embedded build-time
// version string.  The harness injects "test-version" so the assertions are
// deterministic.
func TestVersionCommandAndAlias(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"v"}, {"--version"}} {
		stdout, stderr, code := execute(t, args...)
		if code != 0 {
			t.Fatalf("%v exit code = %d, want 0; stderr=%s", args, code, stderr)
		}
		if strings.TrimSpace(stdout) != "test-version" {
			t.Fatalf("%v stdout = %q, want test-version", args, stdout)
		}
	}
}

// The version command rejects positional args so accidental typos like
// `cam version 1.2.3` fail fast instead of silently ignoring them.
func TestVersionCommandRejectsExtraArgs(t *testing.T) {
	stdout, stderr, code := execute(t, "version", "extra")
	if code == 0 {
		t.Fatalf("expected non-zero exit on extra args; stdout=%s", stdout)
	}
	if !strings.Contains(stderr, "unknown command") && !strings.Contains(stderr, "accepts") {
		t.Fatalf("stderr missing arg-rejection text: %s", stderr)
	}
}

// `cam version --help` documents the subcommand without printing the version.
func TestVersionHelpRendersUsage(t *testing.T) {
	stdout, _, code := execute(t, "version", "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Display current version") {
		t.Fatalf("help missing description:\n%s", stdout)
	}
}
