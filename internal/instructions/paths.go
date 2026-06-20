package instructions

import (
	"path/filepath"
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

// managedDir returns the directory holding CAM-managed instruction files.
// It honors CAM_CONFIG_DIR (via pathutil.ConfigDir) so tests can isolate it.
func managedDir() string {
	return filepath.Join(pathutil.ConfigDir(), "instructions")
}

// safeName maps a human instruction name to a filesystem-safe basename by
// replacing every character outside [A-Za-z0-9._-] with '_'.
func safeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// managedFilePath returns the absolute path of the managed file for name.
func managedFilePath(name string) string {
	return filepath.Join(managedDir(), safeName(name)+".md")
}

// isUnderManagedDir reports whether path resolves inside the managed dir.
func isUnderManagedDir(path string) bool {
	dir := managedDir()
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// TargetPath delegates to entities.InstructionPath to resolve the concrete
// install path for an instruction on a given app/level.
func TargetPath(app string, level entities.InstallLevel, projectDir string) (string, error) {
	return entities.InstructionPath(app, level, projectDir)
}
