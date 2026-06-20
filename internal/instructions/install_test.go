package instructions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// symlinksWork reports whether the OS lets this process create symlinks.
func symlinksWork(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "lnk")
	if err := os.Symlink(filepath.Join(dir, "src"), target); err != nil {
		return false
	}
	os.Remove(target)
	return true
}

func TestSymlinkInstall(t *testing.T) {
	if !symlinksWork(t) {
		t.Skip("symlinks unavailable on this platform/session")
	}
	ctx := context.Background()
	s := newTestStore(t)
	in, err := s.Create(ctx, "Sym", "", "body\n")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ins, err := s.Install(ctx, in.ID, "claude", "user", "")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if ins.LinkKind != "symlink" {
		t.Fatalf("expected symlink, got %s", ins.LinkKind)
	}
	resolved, err := os.Readlink(ins.TargetPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if filepath.Clean(resolved) != filepath.Clean(managedFilePath("Sym")) {
		t.Fatalf("symlink points at %s, want %s", resolved, managedFilePath("Sym"))
	}
}

func TestCopyFallback(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	in, err := s.Create(ctx, "Cpy", "", "body\n")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	old := symlinkFunc
	symlinkFunc = func(string, string) error { return privilegeErr{} }
	defer func() { symlinkFunc = old }()

	ins, err := s.Install(ctx, in.ID, "claude", "user", "")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if ins.LinkKind != "copy" {
		t.Fatalf("expected copy, got %s", ins.LinkKind)
	}
	data, err := os.ReadFile(ins.TargetPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "body\n" {
		t.Fatalf("copied content = %q", data)
	}
}

func TestConflictTargetExists(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	in, err := s.Create(ctx, "Conf", "", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	target, err := TargetPath("claude", "user", "")
	if err != nil {
		t.Fatalf("TargetPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(target, []byte("preexisting\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Install should succeed: the existing file is backed up, not deleted.
	ins, err := s.Install(ctx, in.ID, "claude", "user", "")
	if err != nil {
		t.Fatalf("Install should backup existing and succeed: %v", err)
	}
	if ins.TargetPath != target {
		t.Fatalf("target = %s, want %s", ins.TargetPath, target)
	}
	// The original file should be renamed to a backup.
	dir := filepath.Dir(target)
	matches, _ := filepath.Glob(filepath.Join(dir, filepath.Base(target)+".backup.*"))
	if len(matches) == 0 {
		t.Fatalf("expected a backup file in %s", dir)
	}
}

func TestConflictDifferentInstructionSymlink(t *testing.T) {
	if !symlinksWork(t) {
		t.Skip("symlinks unavailable on this platform/session")
	}
	ctx := context.Background()
	s := newTestStore(t)
	a, err := s.Create(ctx, "Alpha", "", "a")
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	if _, err := s.Install(ctx, a.ID, "claude", "user", ""); err != nil {
		t.Fatalf("Install A: %v", err)
	}
	// Simulate the DB losing the install row (or an external symlink) so the
	// UNIQUE constraint is not what triggers the conflict — the lstat path is.
	db, _ := s.open()
	_, _ = db.ExecContext(ctx, `DELETE FROM instruction_installs`)
	db.Close()

	b, err := s.Create(ctx, "Beta", "", "b")
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	_, err = s.Install(ctx, b.ID, "claude", "user", "")
	ce, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected ConflictError, got %v", err)
	}
	if want := "Alpha"; !strings.Contains(ce.Message, want) {
		t.Fatalf("conflict message %q should mention %q", ce.Message, want)
	}
}

func TestProjectLevelInstall(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	in, err := s.Create(ctx, "Proj", "", "p")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	projDir := t.TempDir()
	ins, err := s.Install(ctx, in.ID, "claude", "project", projDir)
	if err != nil {
		t.Fatalf("Install project: %v", err)
	}
	want := filepath.Join(projDir, "CLAUDE.md")
	if filepath.Clean(ins.TargetPath) != filepath.Clean(want) {
		t.Fatalf("project target = %s, want %s", ins.TargetPath, want)
	}
}

func TestProjectDirRequired(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	in, _ := s.Create(ctx, "NoDir", "", "x")
	if _, err := s.Install(ctx, in.ID, "claude", "project", ""); !isErr(err, ErrProjectDirRequired) {
		t.Fatalf("expected ErrProjectDirRequired, got %v", err)
	}
}

func TestUnsupportedLevel(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	in, _ := s.Create(ctx, "Cop", "", "x")
	// copilot has no user-level instruction path.
	if _, err := s.Install(ctx, in.ID, "copilot", "user", ""); !isErr(err, ErrUnsupportedTarget) {
		t.Fatalf("expected ErrUnsupportedTarget, got %v", err)
	}
}

func TestUninstallRemovesSymlinkLeavesEditedCopy(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if symlinksWork(t) {
		in, _ := s.Create(ctx, "SymU", "", "data\n")
		ins, err := s.Install(ctx, in.ID, "claude", "user", "")
		if err != nil {
			t.Fatalf("Install: %v", err)
		}
		if err := s.Uninstall(ctx, ins.ID); err != nil {
			t.Fatalf("Uninstall: %v", err)
		}
		if _, err := os.Lstat(ins.TargetPath); !os.IsNotExist(err) {
			t.Fatalf("symlink not removed: %v", err)
		}
	}

	// Copy install that the user has since edited must NOT be deleted.
	in2, _ := s.Create(ctx, "CpyU", "", "original\n")
	old := symlinkFunc
	symlinkFunc = func(string, string) error { return privilegeErr{} }
	ins2, err := s.Install(ctx, in2.ID, "claude", "user", "")
	symlinkFunc = old
	if err != nil {
		t.Fatalf("Install copy: %v", err)
	}
	// User edits the installed copy.
	if err := os.WriteFile(ins2.TargetPath, []byte("user edited\n"), 0o600); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := s.Uninstall(ctx, ins2.ID); err != nil {
		t.Fatalf("Uninstall copy: %v", err)
	}
	if _, err := os.Stat(ins2.TargetPath); err != nil {
		t.Fatalf("edited copy was destroyed: %v", err)
	}
}
