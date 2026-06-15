package entities

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

// AppPaths describes where a given app stores a kind of entity on disk.  All
// paths are user-level; project-level support can be added by callers if
// needed (Python only supports user level for skills/agents/plugins).
type AppPaths struct {
	App  string
	Path string
}

// Apps returns the install destinations for an entity kind keyed by app name.
// Paths use `~` (resolved at call time) so tests can swap HOME.
var promptApps = map[string]string{
	"claude":    "~/.claude/CLAUDE.md",
	"codex":     "~/.codex/AGENTS.md",
	"gemini":    "~/.gemini/GEMINI.md",
	"copilot":   "~/.copilot/COPILOT.md",
	"codebuddy": "~/.codebuddy/CODEBUDDY.md",
	"opencode":  "~/.config/opencode/AGENTS.md",
}

var skillApps = map[string]string{
	"claude":    "~/.claude/skills",
	"codex":     "~/.codex/skills",
	"gemini":    "~/.gemini/skills",
	"copilot":   "~/.copilot/skills",
	"codebuddy": "~/.codebuddy/skills",
	"droid":     "~/.droid/skills",
	"qwen":      "~/.qwen/skills",
}

var agentApps = map[string]string{
	"claude":    "~/.claude/agents",
	"codex":     "~/.codex/agents",
	"gemini":    "~/.gemini/agents",
	"copilot":   "~/.copilot/agents",
	"codebuddy": "~/.codebuddy/agents",
	"droid":     "~/.droid/agents",
}

var pluginApps = map[string]string{
	"claude":    "~/.claude/plugins",
	"codex":     "~/.codex/plugins",
	"gemini":    "~/.gemini/plugins",
	"copilot":   "~/.copilot/plugins",
	"codebuddy": "~/.codebuddy/plugins",
	"droid":     "~/.droid/plugins",
}

// AppPathsFor returns the destinations for a Kind keyed by app name.
func AppPathsFor(kind Kind) map[string]string {
	switch kind {
	case KindPrompt:
		return promptApps
	case KindSkill:
		return skillApps
	case KindAgent:
		return agentApps
	case KindPlugin:
		return pluginApps
	}
	return nil
}

// SupportedApps returns the supported app names for the kind, sorted.
func SupportedApps(kind Kind) []string {
	apps := AppPathsFor(kind)
	out := make([]string, 0, len(apps))
	for a := range apps {
		out = append(out, a)
	}
	sortStrings(out)
	return out
}

// InstallToApp writes the entity's content to the resolved location for app.
// For prompts: writes Content as a single file.  For skills/agents/plugins:
// creates a directory named entity.Name containing a SKILL.md/AGENT.md/manifest.json
// — minimal but matches the Python tree shape.
func InstallToApp(entity Entity, kind Kind, app string) (string, error) {
	apps := AppPathsFor(kind)
	dest, ok := apps[app]
	if !ok {
		return "", fmt.Errorf("entities: app %s does not support %s", app, kind)
	}
	resolved := pathutil.Expand(dest)
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return "", err
	}
	switch kind {
	case KindPrompt:
		return resolved, writeFile(resolved, []byte(entity.Content), 0o600)
	default:
		dir := filepath.Join(resolved, entity.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		switch kind {
		case KindSkill:
			path := filepath.Join(dir, "SKILL.md")
			return dir, writeFile(path, []byte(entity.Content), 0o600)
		case KindAgent:
			path := filepath.Join(dir, "AGENT.md")
			return dir, writeFile(path, []byte(entity.Content), 0o600)
		case KindPlugin:
			path := filepath.Join(dir, "manifest.json")
			content := entity.Content
			if strings.TrimSpace(content) == "" {
				content = "{}\n"
			}
			return dir, writeFile(path, []byte(content), 0o600)
		}
	}
	return resolved, nil
}

// UninstallFromApp removes the entity's installation for app and reports
// whether anything was removed.
func UninstallFromApp(entityName string, kind Kind, app string) (string, bool, error) {
	apps := AppPathsFor(kind)
	dest, ok := apps[app]
	if !ok {
		return "", false, fmt.Errorf("entities: app %s does not support %s", app, kind)
	}
	resolved := pathutil.Expand(dest)
	switch kind {
	case KindPrompt:
		if !pathutil.Exists(resolved) {
			return resolved, false, nil
		}
		// Prompts are app-wide files; removing the file is opt-in only when
		// the file matches the entity's content marker.  We don't truncate
		// arbitrary user data — instead report "found" if the file exists.
		return resolved, false, nil
	default:
		dir := filepath.Join(resolved, entityName)
		if !pathutil.Exists(dir) {
			return dir, false, nil
		}
		if err := os.RemoveAll(dir); err != nil {
			return dir, false, err
		}
		return dir, true, nil
	}
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
