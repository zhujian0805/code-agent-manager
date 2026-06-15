package editorconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

// jsonToolConfig implements ToolConfig backed by a JSON file.
type jsonToolConfig struct {
	spec spec
}

func newJSONToolConfig(s spec) *jsonToolConfig {
	return &jsonToolConfig{spec: s}
}

func (c *jsonToolConfig) Name() string        { return c.spec.name }
func (c *jsonToolConfig) Description() string { return c.spec.description }
func (c *jsonToolConfig) Format() Format      { return FormatJSON }

func (c *jsonToolConfig) UserPaths() []string {
	return c.spec.resolveUserPaths()
}

func (c *jsonToolConfig) ProjectPath() string {
	if c.spec.projectPath == "" {
		return ""
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, c.spec.projectPath)
}

// PathFor returns the on-disk path used for read/write at the given scope.
// The first user path is used for write, even when no file exists yet (the
// caller's responsibility is to create the directory).
func (c *jsonToolConfig) PathFor(scope Scope) string {
	switch scope {
	case UserScope:
		paths := c.UserPaths()
		for _, p := range paths {
			if pathutil.Exists(p) {
				return p
			}
		}
		if len(paths) > 0 {
			return paths[0]
		}
	case ProjectScope:
		return c.ProjectPath()
	}
	return ""
}

func (c *jsonToolConfig) Load(scope Scope) (map[string]any, string, error) {
	path := c.PathFor(scope)
	if path == "" {
		return nil, "", fmt.Errorf("editorconfig: %s scope %s unsupported", c.spec.name, scope)
	}
	return loadJSON(path)
}

func loadJSON(path string) (map[string]any, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, path, nil
		}
		return nil, path, fmt.Errorf("editorconfig: read %s: %w", path, err)
	}
	if len(data) == 0 {
		return map[string]any{}, path, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, path, fmt.Errorf("editorconfig: parse %s: %w", path, err)
	}
	return out, path, nil
}

func (c *jsonToolConfig) LoadAll() map[string]ScopedConfig {
	all := map[string]ScopedConfig{}
	for _, scope := range []Scope{UserScope, ProjectScope} {
		path := c.PathFor(scope)
		if path == "" {
			continue
		}
		data, _, err := loadJSON(path)
		if err != nil {
			continue
		}
		all[string(scope)] = ScopedConfig{Data: data, Path: path}
	}
	return all
}

func (c *jsonToolConfig) Set(scope Scope, keyPath string, value any) (string, error) {
	parts, err := Parse(keyPath)
	if err != nil {
		return "", err
	}
	path := c.PathFor(scope)
	if path == "" {
		return "", fmt.Errorf("editorconfig: %s scope %s unsupported", c.spec.name, scope)
	}
	data, _, err := loadJSON(path)
	if err != nil {
		return "", err
	}
	Set(data, parts, value)
	if err := writeJSON(path, data); err != nil {
		return "", err
	}
	return path, nil
}

func (c *jsonToolConfig) Unset(scope Scope, keyPath string) (bool, string, error) {
	parts, err := Parse(keyPath)
	if err != nil {
		return false, "", err
	}
	path := c.PathFor(scope)
	if path == "" {
		return false, "", fmt.Errorf("editorconfig: %s scope %s unsupported", c.spec.name, scope)
	}
	data, _, err := loadJSON(path)
	if err != nil {
		return false, "", err
	}
	found := Unset(data, parts)
	if !found {
		return false, path, nil
	}
	if err := writeJSON(path, data); err != nil {
		return false, "", err
	}
	return true, path, nil
}

func writeJSON(path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("editorconfig: mkdir %s: %w", filepath.Dir(path), err)
	}
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("editorconfig: marshal %s: %w", path, err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("editorconfig: write %s: %w", path, err)
	}
	return nil
}
