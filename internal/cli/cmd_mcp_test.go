package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Custom STDIO server add → list → remove round-trip against Claude config.
func TestMCPAddListRemoveCustomServerRoundTrip(t *testing.T) {
	home := isolatedHome(t)

	stdout, stderr, code := execute(t, "mcp", "add", "context7",
		"--client", "claude",
		"--command", "npx",
		"--arg", "-y",
		"--arg", "@upstash/context7-mcp",
		"--env", "FOO=bar",
	)
	if code != 0 {
		t.Fatalf("add exit = %d; stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Added context7 to claude") {
		t.Fatalf("add output unexpected: %s", stdout)
	}

	// Verify written file structure.
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	servers, _ := got["mcpServers"].(map[string]any)
	if servers == nil {
		t.Fatalf("mcpServers missing in:\n%s", data)
	}
	entry, _ := servers["context7"].(map[string]any)
	if entry == nil || entry["command"] != "npx" {
		t.Fatalf("context7 entry wrong: %v", entry)
	}

	stdout, _, code = execute(t, "mcp", "list", "--client", "claude")
	if code != 0 || !strings.Contains(stdout, "context7") || !strings.Contains(stdout, "npx") {
		t.Fatalf("list code=%d stdout=%s", code, stdout)
	}

	stdout, _, code = execute(t, "mcp", "remove", "context7", "--client", "claude")
	if code != 0 || !strings.Contains(stdout, "Removed context7 from claude") {
		t.Fatalf("remove code=%d stdout=%s", code, stdout)
	}

	// After remove, list should report no servers.
	stdout, _, code = execute(t, "mcp", "list", "--client", "claude")
	if code != 0 || !strings.Contains(stdout, "No MCP servers installed") {
		t.Fatalf("post-remove list: code=%d stdout=%s", code, stdout)
	}
}

// `cam mcp add` requires --client.
func TestMCPAddRequiresClientFlag(t *testing.T) {
	isolatedHome(t)
	_, stderr, code := execute(t, "mcp", "add", "context7", "--command", "npx")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "required flag") {
		t.Fatalf("stderr missing required-flag message: %s", stderr)
	}
}

// Registry server install (no --command) resolves the bundled schema.
func TestMCPAddRegistryServerResolvesBundledSchema(t *testing.T) {
	isolatedHome(t)
	stdout, stderr, code := execute(t, "mcp", "add", "mem0-mcp", "--client", "claude")
	if code != 0 {
		t.Fatalf("exit = %d; stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Added mem0-mcp to claude") {
		t.Fatalf("output unexpected:\n%s", stdout)
	}
}

// `cam mcp server list` enumerates the bundled registry.
func TestMCPServerListEnumeratesBundledRegistry(t *testing.T) {
	stdout, _, code := execute(t, "mcp", "server", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "mem0-mcp") {
		t.Fatalf("server list missing mem0-mcp (truncated):\n%s", head(stdout, 200))
	}
}

// `cam mcp server search QUERY` finds matches.
func TestMCPServerSearchFindsMatches(t *testing.T) {
	stdout, _, code := execute(t, "mcp", "server", "search", "mem0")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "mem0-mcp") {
		t.Fatalf("search missing mem0-mcp (truncated):\n%s", head(stdout, 400))
	}
}

// `cam mcp server show NAME` prints the schema JSON.
func TestMCPServerShowEmitsSchemaJSON(t *testing.T) {
	stdout, _, code := execute(t, "mcp", "server", "show", "mem0-mcp")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(stdout), &schema); err != nil {
		t.Fatalf("show output not valid JSON: %v\n%s", err, stdout)
	}
	if schema["name"] != "mem0-mcp" {
		t.Fatalf("name = %v", schema["name"])
	}
}

// Unknown clients fail loud.
func TestMCPListUnknownClientErrors(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "mcp", "add", "x", "--client", "ghostly", "--command", "echo")
	if code == 0 {
		t.Fatalf("expected non-zero; stdout=%s", stdout)
	}
}

// Remove of a non-existent server reports "Not installed" with exit 0.
func TestMCPRemoveMissingServerIsBenign(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "mcp", "remove", "never-added", "--client", "claude")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Not installed") {
		t.Fatalf("missing 'Not installed':\n%s", stdout)
	}
}

// `cam m` alias works.
func TestMCPAliasM(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "m", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No MCP servers installed") {
		t.Fatalf("alias m output:\n%s", stdout)
	}
}
