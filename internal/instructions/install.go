package instructions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

// symlinkFunc is the symlink primitive, overridable in tests to exercise the
// copy fallback on platforms where real symlinks always succeed.
var symlinkFunc = os.Symlink

// errPrivilegeNotHeld mirrors Windows ERROR_PRIVILEGE_NOT_HELD (1314), the
// failure raised when a process lacks the right to create a symlink.
const windowsPrivilegeNotHeld = 1314

// isPrivilegeError reports whether err indicates symlink creation was denied
// for lack of privilege (the Windows non-developer-mode case).
func isPrivilegeError(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(strings.ToLower(err.Error()), "privilege") {
		return true
	}
	// syscall.Errno on Windows compares equal to the numeric code.
	type numbered interface{ Errno() uintptr }
	var n numbered
	if errors.As(err, &n) && n.Errno() == windowsPrivilegeNotHeld {
		return true
	}
	return false
}

// Install links the instruction identified by id into the coding-agent path
// for (app, level, projectDir), using a symlink with a copy fallback.
func (s *Store) Install(ctx context.Context, id int64, app, level, projectDir string) (Install, error) {
	if err := s.Init(ctx); err != nil {
		return Install{}, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return Install{}, err
	}

	lvl := entities.InstallLevel(level)
	if lvl == entities.InstallLevelProject && strings.TrimSpace(projectDir) == "" {
		return Install{}, ErrProjectDirRequired
	}
	if !levelSupported(app, lvl) {
		return Install{}, fmt.Errorf("%w: %s does not support %s-level installs", ErrUnsupportedTarget, app, level)
	}

	target, err := entities.InstructionPath(app, lvl, projectDir)
	if err != nil {
		// Distinguish "unsupported" (programmer/input error) from filesystem
		// problems like a missing project directory.
		return Install{}, err
	}

	src := managedFilePath(current.Name)

	if err := s.checkConflict(ctx, target, id); err != nil {
		return Install{}, err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Install{}, fmt.Errorf("instructions: mkdir target dir: %w", err)
	}

	linkKind := "symlink"
	if err := symlinkFunc(src, target); err != nil {
		if isPrivilegeError(err) {
			if cerr := copyFile(src, target); cerr != nil {
				return Install{}, fmt.Errorf("instructions: copy fallback: %w", cerr)
			}
			linkKind = "copy"
		} else {
			return Install{}, fmt.Errorf("instructions: symlink: %w", err)
		}
	}

	db, err := s.open()
	if err != nil {
		return Install{}, err
	}
	defer db.Close()
	// Remove any prior install row for this (app, level, project_dir) so a
	// re-install replaces the old one cleanly.  The actual file was already
	// backed up by checkConflict above.
	_, _ = db.ExecContext(ctx, `DELETE FROM instruction_installs WHERE app = ? AND level = ? AND project_dir = ?`, app, level, projectDir)

	now := time.Now().UTC().Format(rfc)
	res, err := db.ExecContext(ctx, `INSERT INTO instruction_installs(instruction_id, app, level, project_dir, target_path, link_kind, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		id, app, level, projectDir, target, linkKind, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Install{}, &ConflictError{Message: fmt.Sprintf("an instruction is already installed at %s; uninstall it first", target)}
		}
		return Install{}, fmt.Errorf("instructions: insert install: %w", err)
	}
	insID, _ := res.LastInsertId()
	return Install{
		ID:         insID,
		App:        app,
		Level:      level,
		ProjectDir: projectDir,
		TargetPath: target,
		LinkKind:   linkKind,
		CreatedAt:  parseTime(now),
	}, nil
}

// levelSupported reports whether app supports installs at level.
func levelSupported(app string, level entities.InstallLevel) bool {
	for _, l := range entities.InstructionAppLevels(app) {
		if l == level {
			return true
		}
	}
	return false
}

// checkConflict refuses the install when a CAM-owned symlink for a *different*
// instruction already occupies the target.  When the file belongs to the same
// instruction or is a user file, it backs it up with a timestamp so the
// install can proceed.
func (s *Store) checkConflict(ctx context.Context, target string, id int64) error {
	info, err := os.Lstat(target)
	if err != nil {
		return nil // nothing there — clear to install
	}
	// Hard conflict: another CAM instruction owns this symlink.
	if info.Mode()&os.ModeSymlink != 0 {
		if resolved, rerr := os.Readlink(target); rerr == nil && isUnderManagedDir(resolved) {
			if other, ok := s.instructionForFile(ctx, resolved); ok && other.ID != id {
				return &ConflictError{Message: fmt.Sprintf("%s is currently installed at %s; uninstall it first", other.Name, target)}
			}
		}
	}
	// File exists but is safe to replace — back it up first.
	if err := backupExistingFile(target); err != nil {
		return &ConflictError{Message: fmt.Sprintf("failed to backup %s: %v", target, err)}
	}
	return nil
}

// backupExistingFile renames target to target.backup.<YYYYMMDD-HHMMSS>.
func backupExistingFile(target string) error {
	ts := time.Now().Local().Format("20060102-150405")
	backup := target + ".backup." + ts
	return os.Rename(target, backup)
}

// instructionForFile returns the instruction whose managed file equals path.
func (s *Store) instructionForFile(ctx context.Context, path string) (Instruction, bool) {
	list, err := s.List(ctx)
	if err != nil {
		return Instruction{}, false
	}
	clean := filepath.Clean(path)
	for _, in := range list {
		if filepath.Clean(managedFilePath(in.Name)) == clean {
			return in, true
		}
	}
	return Instruction{}, false
}

// Uninstall removes the link/copy CAM placed (when safe) and deletes the row.
func (s *Store) Uninstall(ctx context.Context, installID int64) error {
	if err := s.Init(ctx); err != nil {
		return err
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	var ins Install
	var instructionID int64
	var created string
	err = db.QueryRowContext(ctx, `SELECT id, instruction_id, app, level, project_dir, target_path, link_kind, created_at FROM instruction_installs WHERE id = ?`, installID).
		Scan(&ins.ID, &instructionID, &ins.App, &ins.Level, &ins.ProjectDir, &ins.TargetPath, &ins.LinkKind, &created)
	if err != nil {
		// Idempotent: nothing to remove.
		return nil
	}

	var name string
	if nerr := db.QueryRowContext(ctx, `SELECT name FROM instructions WHERE id = ?`, instructionID).Scan(&name); nerr == nil {
		removeInstalledFile(ins, managedFilePath(name))
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM instruction_installs WHERE id = ?`, installID); err != nil {
		return fmt.Errorf("instructions: delete install: %w", err)
	}
	return nil
}

// removeInstalledFile deletes the installed file only when CAM can prove it owns
// it: a symlink pointing at managedFile, or a copy whose content still matches.
func removeInstalledFile(ins Install, managedFile string) {
	info, err := os.Lstat(ins.TargetPath)
	if err != nil {
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if resolved, rerr := os.Readlink(ins.TargetPath); rerr == nil {
			if filepath.Clean(resolved) == filepath.Clean(managedFile) {
				os.Remove(ins.TargetPath)
			}
		}
		return
	}
	if ins.LinkKind == "copy" {
		if sameContent(ins.TargetPath, managedFile) {
			os.Remove(ins.TargetPath)
		}
	}
}

func sameContent(a, b string) bool {
	da, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	db, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return string(da) == string(db)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
