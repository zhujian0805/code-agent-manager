package editorconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
	"github.com/pelletier/go-toml/v2"
)

// tomlToolConfig implements ToolConfig backed by a TOML file (currently only
// used for codex's ~/.codex/config.toml).
type tomlToolConfig struct {
	spec spec
}

func newTOMLToolConfig(s spec) *tomlToolConfig {
	return &tomlToolConfig{spec: s}
}

func (c *tomlToolConfig) Name() string        { return c.spec.name }
func (c *tomlToolConfig) Description() string { return c.spec.description }
func (c *tomlToolConfig) Format() Format      { return FormatTOML }

func (c *tomlToolConfig) UserPaths() []string {
	return c.spec.resolveUserPaths()
}

func (c *tomlToolConfig) ProjectPath() string {
	if c.spec.projectPath == "" {
		return ""
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, c.spec.projectPath)
}

func (c *tomlToolConfig) PathFor(scope Scope) string {
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

func (c *tomlToolConfig) Load(scope Scope) (map[string]any, string, error) {
	path := c.PathFor(scope)
	if path == "" {
		return nil, "", fmt.Errorf("editorconfig: %s scope %s unsupported", c.spec.name, scope)
	}
	return loadTOML(path)
}

func loadTOML(path string) (map[string]any, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, path, nil
		}
		return nil, path, fmt.Errorf("editorconfig: read %s: %w", path, err)
	}
	out := map[string]any{}
	if err := toml.Unmarshal(data, &out); err != nil {
		return nil, path, fmt.Errorf("editorconfig: parse %s: %w", path, err)
	}
	return out, path, nil
}

func (c *tomlToolConfig) LoadAll() map[string]ScopedConfig {
	all := map[string]ScopedConfig{}
	for _, scope := range []Scope{UserScope, ProjectScope} {
		path := c.PathFor(scope)
		if path == "" {
			continue
		}
		data, _, err := loadTOML(path)
		if err != nil {
			continue
		}
		all[string(scope)] = ScopedConfig{Data: data, Path: path}
	}
	return all
}

func (c *tomlToolConfig) Set(scope Scope, keyPath string, value any) (string, error) {
	parts, err := Parse(keyPath)
	if err != nil {
		return "", err
	}
	path := c.PathFor(scope)
	if path == "" {
		return "", fmt.Errorf("editorconfig: %s scope %s unsupported", c.spec.name, scope)
	}
	data, _, err := loadTOML(path)
	if err != nil {
		return "", err
	}
	Set(data, parts, value)
	if err := writeTOML(path, data); err != nil {
		return "", err
	}
	return path, nil
}

func (c *tomlToolConfig) Unset(scope Scope, keyPath string) (bool, string, error) {
	parts, err := Parse(keyPath)
	if err != nil {
		return false, "", err
	}
	path := c.PathFor(scope)
	if path == "" {
		return false, "", fmt.Errorf("editorconfig: %s scope %s unsupported", c.spec.name, scope)
	}
	data, _, err := loadTOML(path)
	if err != nil {
		return false, "", err
	}
	found := Unset(data, parts)
	if !found {
		return false, path, nil
	}
	if err := writeTOML(path, data); err != nil {
		return false, "", err
	}
	return true, path, nil
}

func writeTOML(path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("editorconfig: mkdir %s: %w", filepath.Dir(path), err)
	}
	encoded, err := toml.Marshal(data)
	if err != nil {
		return fmt.Errorf("editorconfig: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("editorconfig: write %s: %w", path, err)
	}
	return nil
}
