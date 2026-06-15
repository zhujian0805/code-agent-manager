package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

// --- list ------------------------------------------------------------------

func TestAgentListWhenEmpty(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "agent", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No agents installed across agents") {
		t.Fatalf("missing empty state:\n%s", stdout)
	}
}

func TestAgentListShowsInstalled(t *testing.T) {
	home := isolatedHome(t)
	installEntityToApp(t, home, entities.KindAgent, "code-reviewer", "agent content", "claude")
	stdout, _, code := execute(t, "agent", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "code-reviewer") {
		t.Fatalf("list missing agent:\n%s", stdout)
	}
	if !strings.Contains(stdout, "claude") {
		t.Fatalf("list missing app name:\n%s", stdout)
	}
}

// --- search ----------------------------------------------------------------

func TestAgentSearchFindsMatch(t *testing.T) {
	isolatedHome(t)
	seedEntity(t, entities.KindAgent, "code-reviewer", "content", "Code review agent")
	seedEntity(t, entities.KindAgent, "test-runner", "content", "Runs tests")
	stdout, _, code := execute(t, "agent", "search", "review", "--local")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "code-reviewer") {
		t.Fatalf("search missing match:\n%s", stdout)
	}
	if strings.Contains(stdout, "test-runner") {
		t.Fatalf("search should not include non-matching:\n%s", stdout)
	}
}

// --- install ---------------------------------------------------------------

func TestAgentInstallCreatesAgentDirectoryWithMarkdown(t *testing.T) {
	home := isolatedHome(t)
	seedEntity(t, entities.KindAgent, "code-reviewer", "agent body", "")
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

// --- alias -----------------------------------------------------------------

func TestAgentAliasAg(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "ag", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No agents installed across agents") {
		t.Fatalf("alias output: %s", stdout)
	}
}
