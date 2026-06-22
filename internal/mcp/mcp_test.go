package mcp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/camconfig"
	"github.com/chat2anyllm/code-agent-manager/internal/mcp"
)

func TestRegistrySearchByDescription(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp_servers.json")
	writeCatalog(t, path, []mcp.ServerSchema{testSchema("memory-mcp", "Memory catalog server")})
	r, err := mcp.LoadRegistryFromConfig(testCatalogConfig(path))
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig err = %v", err)
	}
	results := r.Search("memory")
	if len(results) == 0 {
		t.Fatal("expected at least one server matching 'memory'")
	}
}

func TestLoadRegistry_loadsDirectArrayFromLocalSource(t *testing.T) {
	// Given
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_servers.json")
	writeCatalog(t, path, []mcp.ServerSchema{testSchema("local-only", "Local only")})
	cfg := testCatalogConfig(path)

	// When
	registry, err := mcp.LoadRegistryFromConfig(cfg)

	// Then
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig err = %v", err)
	}
	if _, ok := registry.Get("local-only"); !ok {
		t.Fatal("expected local-only in catalog registry")
	}
}

func TestLoadRegistry_loadsDirectMapFromLocalSource(t *testing.T) {
	// Given
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_servers.json")
	writeCatalog(t, path, map[string]mcp.ServerSchema{
		"map-only": testSchema("map-only", "Map only"),
	})
	cfg := testCatalogConfig(path)

	// When
	registry, err := mcp.LoadRegistryFromConfig(cfg)

	// Then
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig err = %v", err)
	}
	if _, ok := registry.Get("map-only"); !ok {
		t.Fatal("expected map-only in catalog registry")
	}
}

func TestLoadRegistry_loadsWrappedCatalogFromLocalSource(t *testing.T) {
	// Given
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_servers.json")
	writeCatalog(t, path, map[string]any{
		"schema_version": 1,
		"servers":        []mcp.ServerSchema{testSchema("wrapped-only", "Wrapped only")},
	})
	cfg := testCatalogConfig(path)

	// When
	registry, err := mcp.LoadRegistryFromConfig(cfg)

	// Then
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig err = %v", err)
	}
	if _, ok := registry.Get("wrapped-only"); !ok {
		t.Fatal("expected wrapped-only in catalog registry")
	}
}

func TestLoadRegistry_keepsLocalEntryWhenRemoteDuplicatesName(t *testing.T) {
	// Given
	dir := t.TempDir()
	localPath := filepath.Join(dir, "local.json")
	remotePath := filepath.Join(dir, "remote.json")
	writeCatalog(t, localPath, []mcp.ServerSchema{testSchema("duplicate", "Local description")})
	writeCatalog(t, remotePath, []mcp.ServerSchema{testSchema("duplicate", "Remote description")})
	cfg := camconfig.CamConfig{
		Repositories: map[string]camconfig.RepoSources{
			"mcpServers": {Sources: []camconfig.RepoSource{
				{Type: "local", Path: localPath},
				{Type: "local", Path: remotePath},
			}},
		},
	}

	// When
	registry, err := mcp.LoadRegistryFromConfig(cfg)

	// Then
	if err != nil {
		t.Fatalf("LoadRegistryFromConfig err = %v", err)
	}
	got, ok := registry.Get("duplicate")
	if !ok {
		t.Fatal("expected duplicate in catalog registry")
	}
	if got.Description != "Local description" {
		t.Fatalf("description = %q, want local source priority", got.Description)
	}
}

func TestLoadRegistry_rejectsMalformedInstallableSchema(t *testing.T) {
	// Given
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_servers.json")
	writeCatalog(t, path, []mcp.ServerSchema{{Name: "missing-installations", Description: "broken"}})
	cfg := testCatalogConfig(path)

	// When
	_, err := mcp.LoadRegistryFromConfig(cfg)

	// Then
	if err == nil {
		t.Fatal("expected malformed installable schema error")
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

func testCatalogConfig(path string) camconfig.CamConfig {
	return camconfig.CamConfig{
		Repositories: map[string]camconfig.RepoSources{
			"mcpServers": {Sources: []camconfig.RepoSource{{Type: "local", Path: path}}},
		},
	}
}

func testSchema(name, description string) mcp.ServerSchema {
	return mcp.ServerSchema{
		Name:        name,
		Description: description,
		Installations: map[string]mcp.InstallationEntry{
			"npm": {Type: "npm", Command: "npx", Args: []string{"-y", name}},
		},
	}
}

func writeCatalog(t *testing.T, path string, catalog any) {
	t.Helper()
	raw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
}
