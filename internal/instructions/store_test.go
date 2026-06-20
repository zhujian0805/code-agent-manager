package instructions

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newTestStore returns a Store backed by a temp cam.db with CAM_CONFIG_DIR and
// HOME isolated to temp directories so managed files and user-level install
// paths never touch the real home.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	cfg := t.TempDir()
	home := t.TempDir()
	t.Setenv("CAM_CONFIG_DIR", cfg)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := New(filepath.Join(cfg, "cam.db"))
	if err := s.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return s
}

func TestCRUDRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	created, err := s.Create(ctx, "Instruction01", "first", "# hello\n")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.Name != "Instruction01" {
		t.Fatalf("created = %+v", created)
	}
	// Managed file written.
	if _, err := os.Stat(managedFilePath("Instruction01")); err != nil {
		t.Fatalf("managed file missing: %v", err)
	}

	list, err := s.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %+v err=%v", list, err)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil || got.Content != "# hello\n" {
		t.Fatalf("Get = %+v err=%v", got, err)
	}

	updated, err := s.Update(ctx, created.ID, "Instruction01", "edited", "# changed\n")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Description != "edited" || updated.Content != "# changed\n" {
		t.Fatalf("updated = %+v", updated)
	}
	data, _ := os.ReadFile(managedFilePath("Instruction01"))
	if string(data) != "# changed\n" {
		t.Fatalf("managed file content = %q", data)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(managedFilePath("Instruction01")); !os.IsNotExist(err) {
		t.Fatalf("managed file not removed: %v", err)
	}
	list, _ = s.List(ctx)
	if len(list) != 0 {
		t.Fatalf("list after delete = %+v", list)
	}
}

func TestRenameRetargetsSymlinkLeavesCopyStale(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	in, err := s.Create(ctx, "Old", "", "content\n")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Symlink install to claude user-level.
	sym, err := s.Install(ctx, in.ID, "claude", "user", "")
	if err != nil {
		t.Fatalf("Install symlink: %v", err)
	}
	if sym.LinkKind != "symlink" {
		t.Skipf("platform produced %s install; symlink retarget not exercised", sym.LinkKind)
	}

	// Force a copy install to gemini user-level by injecting a failing symlink.
	old := symlinkFunc
	symlinkFunc = func(string, string) error { return privilegeErr{} }
	copyIns, err := s.Install(ctx, in.ID, "gemini", "user", "")
	symlinkFunc = old
	if err != nil {
		t.Fatalf("Install copy: %v", err)
	}
	if copyIns.LinkKind != "copy" {
		t.Fatalf("expected copy install, got %s", copyIns.LinkKind)
	}

	if _, err := s.Update(ctx, in.ID, "New", "", "content\n"); err != nil {
		t.Fatalf("Update rename: %v", err)
	}

	// Old managed file gone, new one present.
	if _, err := os.Stat(managedFilePath("Old")); !os.IsNotExist(err) {
		t.Fatalf("old managed file still present: %v", err)
	}
	newFile := managedFilePath("New")
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("new managed file missing: %v", err)
	}

	// Symlink install now resolves to the new managed file.
	resolved, err := os.Readlink(sym.TargetPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if filepath.Clean(resolved) != filepath.Clean(newFile) {
		t.Fatalf("symlink not retargeted: %s -> %s", sym.TargetPath, resolved)
	}

	// Copy install left untouched (still points at the file on disk; stale).
	if _, err := os.Lstat(copyIns.TargetPath); err != nil {
		t.Fatalf("copy install removed unexpectedly: %v", err)
	}
}

func TestDuplicateName(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	if _, err := s.Create(ctx, "Dup", "", "a"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err := s.Create(ctx, "Dup", "", "b")
	if !isErr(err, ErrDuplicateName) {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestInvalidName(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	for _, bad := range []string{"", "  ", "a/b", `a\b`} {
		if _, err := s.Create(ctx, bad, "", "x"); !isErr(err, ErrInvalidName) {
			t.Fatalf("name %q: expected ErrInvalidName, got %v", bad, err)
		}
	}
}

func TestDeleteCascadesInstalls(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	in, err := s.Create(ctx, "Casc", "", "x")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := s.Install(ctx, in.ID, "claude", "user", ""); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if err := s.Delete(ctx, in.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	db, _ := s.open()
	defer db.Close()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM instruction_installs WHERE instruction_id = ?`, in.ID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("install rows remain after delete: %d", count)
	}
}

func TestGetNotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	if _, err := s.Get(ctx, 999); !isErr(err, ErrInstructionNotFound) {
		t.Fatalf("expected ErrInstructionNotFound, got %v", err)
	}
}

// privilegeErr is a synthetic error that isPrivilegeError recognizes so the
// copy fallback is portable across platforms.
type privilegeErr struct{}

func (privilegeErr) Error() string { return "symlink: a required privilege is not held by the client" }

func isErr(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
