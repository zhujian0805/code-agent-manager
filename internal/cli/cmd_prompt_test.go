package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

// --- list ------------------------------------------------------------------

func TestPromptListWhenEmpty(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "prompt", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No prompts installed across agents") {
		t.Fatalf("missing empty state:\n%s", stdout)
	}
}

func TestPromptListShowsInstalled(t *testing.T) {
	home := isolatedHome(t)
	installEntityToApp(t, home, entities.KindPrompt, "", "prompt content", "claude")
	stdout, _, code := execute(t, "prompt", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "claude") {
		t.Fatalf("list missing claude:\n%s", stdout)
	}
}

// --- search ----------------------------------------------------------------

func TestPromptSearchFindsMatch(t *testing.T) {
	isolatedHome(t)
	seedEntity(t, entities.KindPrompt, "greeting", "Hello", "A greeting prompt")
	seedEntity(t, entities.KindPrompt, "farewell", "Bye", "A farewell prompt")
	stdout, _, code := execute(t, "prompt", "search", "greeting", "--local")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "greeting") {
		t.Fatalf("search missing match:\n%s", stdout)
	}
	if strings.Contains(stdout, "farewell") {
		t.Fatalf("search should not include non-matching:\n%s", stdout)
	}
}

// --- install ---------------------------------------------------------------

func TestPromptInstallWritesContentToAppPath(t *testing.T) {
	home := isolatedHome(t)
	seedEntity(t, entities.KindPrompt, "demo", "prompt body", "")
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

func TestPromptInstallWithoutAppErrors(t *testing.T) {
	isolatedHome(t)
	seedEntity(t, entities.KindPrompt, "demo", "body", "")
	_, stderr, code := execute(t, "prompt", "install", "demo")
	if code == 0 {
		t.Fatal("expected non-zero exit without --app")
	}
	if !strings.Contains(stderr, "--app is required") {
		t.Fatalf("stderr missing --app guidance: %s", stderr)
	}
}

// --- alias -----------------------------------------------------------------

func TestPromptAliasP(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "p", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No prompts installed across agents") {
		t.Fatalf("alias p output: %s", stdout)
	}
}
