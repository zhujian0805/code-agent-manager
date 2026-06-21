package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

// fileMu protects concurrent read-modify-write operations on MCP config files.
// Keyed by resolved file path.
var fileMu sync.Map

// Scope identifies user-level vs project-level MCP installation.
type Scope string

const (
	UserScope    Scope = "user"
	ProjectScope Scope = "project"
)

// ClientSpec describes one supported MCP client's config file layout.
type ClientSpec struct {
	Name        string
	UserPath    string // ~ expanded relative
	ProjectPath string // relative to cwd (empty = unsupported)
	Container   string // top-level key that holds servers ("mcpServers", "servers", ...)
	Format      string // "json" today; sub-projects may add "toml"
}

// SupportedClients enumerates every client we know how to write MCP entries
// for.  Matches the per-client Python implementations.
var SupportedClients = []ClientSpec{
	{Name: "claude", UserPath: "~/.claude.json", ProjectPath: ".claude/settings.json", Container: "mcpServers"},
	{Name: "codex", UserPath: "~/.codex/config.toml", Container: "mcp_servers"}, // codex uses TOML
	{Name: "cursor-agent", UserPath: "~/.cursor/mcp.json", ProjectPath: ".cursor/mcp.json", Container: "mcpServers"},
	{Name: "gemini", UserPath: "~/.gemini/settings.json", ProjectPath: ".gemini/settings.json", Container: "mcpServers"},
	{Name: "copilot", UserPath: "~/.copilot/mcp-config.json", Container: "mcp"},
	{Name: "qwen", UserPath: "~/.qwen/settings.json", Container: "mcpServers"},
	{Name: "codebuddy", UserPath: "~/.codebuddy.json", ProjectPath: ".codebuddy/mcp.json", Container: "mcpServers"},
	{Name: "droid", UserPath: "~/.factory/mcp.json", Container: "mcpServers"},
	{Name: "iflow", UserPath: "~/.iflow/settings.json", Container: "mcpServers"},
	{Name: "neovate", UserPath: "~/.neovate/config.json", Container: "mcpServers"},
	{Name: "qodercli", UserPath: "~/.qodercli/config.json", Container: "mcpServers"},
	{Name: "crush", UserPath: "~/.config/crush/crush.json", Container: "mcp"},
	{Name: "zed", UserPath: "~/.config/zed/settings.json", Container: "context_servers"},
	{Name: "opencode", UserPath: "~/.config/opencode/config.json", Container: "mcp"},
	{Name: "continue", UserPath: "~/.continue/config.json", Container: "mcpServers"},
}

// ClientByName returns the spec for a client name, or false when unsupported.
func ClientByName(name string) (ClientSpec, bool) {
	for _, c := range SupportedClients {
		if c.Name == name {
			return c, true
		}
	}
	return ClientSpec{}, false
}

// ClientNames returns the supported client names in alphabetical order.
func ClientNames() []string {
	names := make([]string, 0, len(SupportedClients))
	for _, c := range SupportedClients {
		names = append(names, c.Name)
	}
	sort.Strings(names)
	return names
}

// ResolvePath returns the absolute config file path for the given client +
// scope, honoring ~ and the current working directory.
func (c ClientSpec) ResolvePath(scope Scope) string {
	switch scope {
	case UserScope:
		return pathutil.Expand(c.UserPath)
	case ProjectScope:
		if c.ProjectPath == "" {
			return ""
		}
		if filepath.IsAbs(c.ProjectPath) {
			return c.ProjectPath
		}
		wd, _ := os.Getwd()
		return filepath.Join(wd, c.ProjectPath)
	}
	return ""
}

// Server is the canonical CAM-side representation of an installed MCP server.
type Server struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Type    string            `json:"type,omitempty"`
}

// AddServer writes server into the client's config file at scope.  Missing
// files are created with restrictive permissions; existing files are merged.
func AddServer(client ClientSpec, scope Scope, server Server) (string, error) {
	path := client.ResolvePath(scope)
	if path == "" {
		return "", fmt.Errorf("mcp: client %s does not support scope %s", client.Name, scope)
	}
	// Lock per-file to prevent concurrent read-modify-write races.
	mu, _ := fileMu.LoadOrStore(path, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()
	data, err := readJSON(path)
	if err != nil {
		return "", err
	}
	container, _ := data[client.Container].(map[string]any)
	if container == nil {
		container = map[string]any{}
	}
	container[server.Name] = serverToMap(server)
	data[client.Container] = container
	if err := writeJSON(path, data); err != nil {
		return "", err
	}
	return path, nil
}

// RemoveServer deletes server from the client's config file at scope and
// reports whether the entry existed.
func RemoveServer(client ClientSpec, scope Scope, serverName string) (string, bool, error) {
	path := client.ResolvePath(scope)
	if path == "" {
		return "", false, fmt.Errorf("mcp: client %s does not support scope %s", client.Name, scope)
	}
	// Lock per-file to prevent concurrent read-modify-write races.
	mu, _ := fileMu.LoadOrStore(path, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()
	data, err := readJSON(path)
	if err != nil {
		return path, false, err
	}
	container, _ := data[client.Container].(map[string]any)
	if container == nil {
		return path, false, nil
	}
	if _, ok := container[serverName]; !ok {
		return path, false, nil
	}
	delete(container, serverName)
	data[client.Container] = container
	if err := writeJSON(path, data); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// ListServers returns all servers installed for the client at scope.
func ListServers(client ClientSpec, scope Scope) ([]Server, string, error) {
	path := client.ResolvePath(scope)
	if path == "" {
		return nil, "", fmt.Errorf("mcp: client %s does not support scope %s", client.Name, scope)
	}
	data, err := readJSON(path)
	if err != nil {
		return nil, path, err
	}
	container, _ := data[client.Container].(map[string]any)
	if container == nil {
		return nil, path, nil
	}
	out := []Server{}
	keys := make([]string, 0, len(container))
	for k := range container {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		entry, ok := container[k].(map[string]any)
		if !ok {
			continue
		}
		out = append(out, mapToServer(k, entry))
	}
	return out, path, nil
}

func serverToMap(server Server) map[string]any {
	out := map[string]any{}
	if server.Type != "" {
		out["type"] = server.Type
	}
	if server.URL != "" {
		out["url"] = server.URL
	}
	if server.Command != "" {
		out["command"] = server.Command
	}
	if len(server.Args) > 0 {
		out["args"] = server.Args
	}
	if len(server.Env) > 0 {
		out["env"] = server.Env
	}
	return out
}

func mapToServer(name string, raw map[string]any) Server {
	s := Server{Name: name}
	if v, ok := raw["command"].(string); ok {
		s.Command = v
	}
	if v, ok := raw["url"].(string); ok {
		s.URL = v
	}
	if v, ok := raw["type"].(string); ok {
		s.Type = v
	}
	if args, ok := raw["args"].([]any); ok {
		for _, a := range args {
			if as, ok := a.(string); ok {
				s.Args = append(s.Args, as)
			}
		}
	}
	if envRaw, ok := raw["env"].(map[string]any); ok {
		s.Env = map[string]string{}
		for k, v := range envRaw {
			if vs, ok := v.(string); ok {
				s.Env[k] = vs
			}
		}
	}
	return s
}

func readJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("mcp: read %s: %w", path, err)
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("mcp: parse %s: %w", path, err)
	}
	return out, nil
}

func writeJSON(path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mcp: mkdir %s: %w", filepath.Dir(path), err)
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("mcp: marshal %s: %w", path, err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("mcp: write %s: %w", path, err)
	}
	return nil
}

// ServerFromSchema constructs a Server from a registry schema, choosing the
// preferred installation method.  Returns an error when the schema has no
// installations defined.
func ServerFromSchema(schema ServerSchema) (Server, error) {
	_, entry, ok := schema.PreferredInstallation()
	if !ok {
		return Server{}, fmt.Errorf("mcp: schema %s has no installation methods", schema.Name)
	}
	server := Server{Name: schema.Name, Command: entry.Command, Args: entry.Args, Env: entry.Env}
	if entry.URL != "" {
		server.URL = entry.URL
		server.Type = "http"
	} else {
		server.Type = "stdio"
	}
	return server, nil
}
