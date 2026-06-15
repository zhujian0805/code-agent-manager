// Package tools loads tools.yaml and provides launch + install primitives for
// the external AI assistant CLIs that CAM wraps (claude, codex, gemini, ...).
package tools

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
	"gopkg.in/yaml.v3"
)

//go:embed embed/tools.yaml
var bundledYAML []byte

// Tool represents a single tools.yaml entry — the subset CAM needs at launch
// and install time.  Unknown fields are tolerated (documentation only).
type Tool struct {
	Name          string    `yaml:"-"`
	Enabled       *bool     `yaml:"enabled"`
	InstallCmd    string    `yaml:"install_cmd"`
	CLICommand    string    `yaml:"cli_command"`
	Description   string    `yaml:"description"`
	Env           Env       `yaml:"env"`
	Configuration any       `yaml:"configuration"`
	CLIParameters CLIParams `yaml:"cli_parameters"`
}

// Env captures the env: block for a tool.
type Env struct {
	Exported    map[string]string `yaml:"exported"`
	Required    map[string]string `yaml:"required"`
	RequiredAny [][]string        `yaml:"required_any"`
	Optional    map[string]string `yaml:"optional"`
	Managed     map[string]string `yaml:"managed"`
	Removed     []string          `yaml:"removed"`
}

// CLIParams captures the cli_parameters block.
type CLIParams struct {
	Injected []string `yaml:"injected"`
}

// IsEnabled reports whether the tool participates in launch/upgrade.
func (t Tool) IsEnabled() bool {
	if t.Enabled == nil {
		return true
	}
	return *t.Enabled
}

// LaunchCommand returns the CLI binary name to launch.  Falls back to the
// tool key when cli_command is empty.
func (t Tool) LaunchCommand() string {
	if t.CLICommand != "" {
		return t.CLICommand
	}
	return t.Name
}

// Registry holds the loaded tools.yaml.
type Registry struct {
	Tools map[string]Tool
}

// LoadDefault returns a Registry from the user's tools.yaml when present, or
// the bundled defaults when not.
func LoadDefault() (*Registry, error) {
	userPath := filepath.Join(pathutil.ConfigDir(), "tools.yaml")
	if pathutil.Exists(userPath) {
		data, err := os.ReadFile(userPath)
		if err != nil {
			return nil, fmt.Errorf("tools: read %s: %w", userPath, err)
		}
		return parseRegistry(data)
	}
	return parseRegistry(bundledYAML)
}

func parseRegistry(data []byte) (*Registry, error) {
	var raw struct {
		Tools map[string]Tool `yaml:"tools"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("tools: parse: %w", err)
	}
	for name, t := range raw.Tools {
		t.Name = name
		raw.Tools[name] = t
	}
	return &Registry{Tools: raw.Tools}, nil
}

// Get returns a tool by its tools.yaml key.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.Tools[name]
	return t, ok
}

// ByCLICommand returns the tool whose cli_command matches the given binary
// name (e.g. "claude" returns the "claude-code" tool).
func (r *Registry) ByCLICommand(cli string) (Tool, bool) {
	for _, t := range r.Tools {
		if t.LaunchCommand() == cli {
			return t, true
		}
	}
	return Tool{}, false
}

// Names returns the registered tool names in alphabetical order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.Tools))
	for name := range r.Tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// EnabledNames returns the enabled tools sorted by name.
func (r *Registry) EnabledNames() []string {
	names := []string{}
	for name, t := range r.Tools {
		if t.IsEnabled() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// LaunchNames returns the enabled tools' CLI binary names (deduplicated and
// sorted).  This is the canonical list of binaries `cam launch` and the
// interactive menu present to the user.
func (r *Registry) LaunchNames() []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, name := range r.EnabledNames() {
		bin := r.Tools[name].LaunchCommand()
		if _, ok := seen[bin]; ok {
			continue
		}
		seen[bin] = struct{}{}
		out = append(out, bin)
	}
	sort.Strings(out)
	return out
}
