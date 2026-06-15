package cli_test

import (
	"strings"
	"testing"
)

// The completion command and its aliases (comp/c) each emit a Cobra-native
// completion script for the requested shell.  The banner doubles as a
// human-readable header and as a quick contains-check for tests.
func TestCompletionEmitsScriptForEverySupportedShell(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "bash long", args: []string{"completion", "bash"}, want: "bash completion"},
		{name: "comp alias zsh", args: []string{"comp", "zsh"}, want: "zsh completion"},
		{name: "short alias fish", args: []string{"c", "fish"}, want: "fish completion"},
		{name: "powershell", args: []string{"completion", "powershell"}, want: "powershell completion"},
		{name: "pwsh normalises to powershell", args: []string{"completion", "pwsh"}, want: "powershell completion"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := execute(t, tc.args...)
			if code != 0 {
				t.Fatalf("exit = %d; stderr=%s", code, stderr)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("stdout missing %q\nstdout (first 200 bytes):\n%s", tc.want, head(stdout, 200))
			}
		})
	}
}

// Unsupported shells must exit non-zero with a recognisable error so shell
// startup scripts that source the output fail loud.
func TestCompletionRejectsUnsupportedShell(t *testing.T) {
	stdout, stderr, code := execute(t, "completion", "nushell")
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero; stdout=%s", stdout)
	}
	if !strings.Contains(stderr, "Unsupported shell") {
		t.Fatalf("stderr missing 'Unsupported shell': %s", stderr)
	}
}

// Completion requires exactly one positional argument (the shell name).
func TestCompletionRequiresShellArgument(t *testing.T) {
	_, stderr, code := execute(t, "completion")
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero")
	}
	if !strings.Contains(stderr, "accepts 1 arg") {
		t.Fatalf("stderr missing arg-count error: %s", stderr)
	}
}

// The bash script must contain the actual completion function so users can
// source it.  This guards against regressions where we silently print only the
// banner.
func TestCompletionBashIncludesGeneratedFunctionBody(t *testing.T) {
	stdout, _, _ := execute(t, "completion", "bash")
	for _, marker := range []string{"__cam_debug", "COMPREPLY", "__start_cam"} {
		if !strings.Contains(stdout, marker) {
			t.Fatalf("bash completion missing %q (truncated):\n%s", marker, head(stdout, 200))
		}
	}
}

func head(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
