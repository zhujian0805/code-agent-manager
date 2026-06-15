package mcp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/mcp"
)

func TestRegistryLoadsBundledSchemas(t *testing.T) {
	r, err := mcp.LoadBundledRegistry()
	if err != nil {
		t.Fatalf("LoadBundledRegistry err = %v", err)
	}
	names := r.Names()
	if len(names) < 100 {
		t.Fatalf("expected at least 100 schemas, got %d", len(names))
	}
	if _, ok := r.Get("mem0-mcp"); !ok {
		t.Fatal("expected mem0-mcp in bundled registry")
	}
}

func TestRegistrySearchByDescription(t *testing.T) {
	r, _ := mcp.LoadBundledRegistry()
	results := r.Search("memory")
	if len(results) == 0 {
		t.Fatal("expected at least one server matching 'memory'")
	}
}

func TestClientsCoverFourteenTools(t *testing.T) {
	if got := len(mcp.SupportedClients); got != 15 {
		t.Fatalf("SupportedClients count = %d, want 15", got)
	}
	for _, name := range []string{"claude", "codex", "gemini", "qwen", "copilot", "droid", "iflow", "codebuddy", "crush", "zed", "neovate", "qodercli", "cursor-agent", "opencode", "continue"} {
		if _, ok := mcp.ClientByName(name); !ok {
			t.Fatalf("client %s missing", name)
		}
	}
}

func TestAddListRemoveServerRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	client, _ := mcp.ClientByName("claude")

	server := mcp.Server{Name: "context7", Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}, Type: "stdio"}
	path, err := mcp.AddServer(client, mcp.UserScope, server)
	if err != nil {
		t.Fatalf("AddServer err = %v", err)
	}
	if path != filepath.Join(home, ".claude.json") {
		t.Fatalf("path = %q", path)
	}

	servers, _, err := mcp.ListServers(client, mcp.UserScope)
	if err != nil {
		t.Fatalf("ListServers err = %v", err)
	}
	if len(servers) != 1 || servers[0].Name != "context7" {
		t.Fatalf("listed = %v", servers)
	}
	if servers[0].Command != "npx" {
		t.Fatalf("command = %q", servers[0].Command)
	}
	if len(servers[0].Args) != 2 {
		t.Fatalf("args = %v", servers[0].Args)
	}

	_, found, err := mcp.RemoveServer(client, mcp.UserScope, "context7")
	if err != nil {
		t.Fatalf("RemoveServer err = %v", err)
	}
	if !found {
		t.Fatal("RemoveServer returned found=false")
	}
	servers, _, _ = mcp.ListServers(client, mcp.UserScope)
	if len(servers) != 0 {
		t.Fatalf("post-remove list = %v", servers)
	}
}

func TestServerFromSchemaPicksPreferredInstallation(t *testing.T) {
	schema := mcp.ServerSchema{
		Name: "x",
		Installations: map[string]mcp.InstallationEntry{
			"docker": {Type: "docker", Command: "docker", Args: []string{"run"}},
			"uvx":    {Type: "uvx", Command: "uvx", Args: []string{"--from", "git"}},
		},
	}
	server, err := mcp.ServerFromSchema(schema)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if server.Command != "docker" {
		t.Fatalf("expected docker preferred, got %q", server.Command)
	}
}

func TestAddServerMergesExistingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	client, _ := mcp.ClientByName("claude")
	path := filepath.Join(home, ".claude.json")
	preexisting := map[string]any{"theme": "dark"}
	raw, _ := json.MarshalIndent(preexisting, "", "  ")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := mcp.AddServer(client, mcp.UserScope, mcp.Server{Name: "x", Command: "y"})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["theme"] != "dark" {
		t.Fatalf("theme overwritten: %v", got)
	}
	if _, ok := got["mcpServers"]; !ok {
		t.Fatalf("missing mcpServers: %v", got)
	}
}
