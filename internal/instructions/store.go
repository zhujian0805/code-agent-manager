// Package instructions provides full CRUD over user-authored, local
// instruction files (CLAUDE.md, AGENTS.md, GEMINI.md, …) and installs a saved
// instruction into a coding-agent path via symlink (with a copy fallback).
//
// State lives in two SQLite tables inside cam.db (the same database the
// appstate package owns) and is mirrored to managed Markdown files under
// ~/.config/code-agent-manager/instructions/<safe-name>.md so that symlink
// installs reflect edits immediately.
package instructions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chat2anyllm/code-agent-manager/internal/appstate"
	_ "modernc.org/sqlite"
)

// Instruction is a user-authored local instruction record.
type Instruction struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Installs    []Install `json:"installs"`
}

// Install records one place an instruction is linked into a coding-agent path.
type Install struct {
	ID         int64     `json:"id"`
	App        string    `json:"app"`
	Level      string    `json:"level"`
	ProjectDir string    `json:"project_dir"`
	TargetPath string    `json:"target_path"`
	LinkKind   string    `json:"link_kind"`
	CreatedAt  time.Time `json:"created_at"`
}

// Sentinel errors surfaced to callers (mapped to HTTP status by the sidecar).
var (
	ErrDuplicateName       = errors.New("instructions: name already exists")
	ErrInvalidName         = errors.New("instructions: invalid name")
	ErrInstructionNotFound = errors.New("instructions: not found")
	ErrUnsupportedTarget   = errors.New("instructions: unsupported target")
	ErrProjectDirRequired  = errors.New("instructions: project directory required")
)

// ConflictError reports that an install target is already occupied.
type ConflictError struct{ Message string }

func (e *ConflictError) Error() string { return e.Message }

// Store persists instructions in cam.db and mirrors content to managed files.
type Store struct {
	dbPath string
}

// New returns a Store backed by the cam.db at dbPath, or the default path when
// dbPath is empty.
func New(dbPath string) *Store {
	if dbPath == "" {
		dbPath = appstate.DefaultPath()
	}
	return &Store{dbPath: dbPath}
}

// Init ensures the schema exists and the managed directory is present.
func (s *Store) Init(ctx context.Context) error {
	if err := appstate.New(s.dbPath).Init(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(managedDir(), 0o755); err != nil {
		return fmt.Errorf("instructions: mkdir managed dir: %w", err)
	}
	return nil
}

func (s *Store) open() (*sql.DB, error) {
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return nil, fmt.Errorf("instructions: open %s: %w", s.dbPath, err)
	}
	return db, nil
}

const rfc = time.RFC3339Nano

func parseTime(v string) time.Time {
	t, _ := time.Parse(rfc, v)
	return t
}

// validateName rejects empty names and names containing path separators.
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is empty", ErrInvalidName)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("%w: name %q contains a path separator", ErrInvalidName, name)
	}
	return nil
}

// List returns all instructions (without installs) ordered by name.
func (s *Store) List(ctx context.Context) ([]Instruction, error) {
	if err := s.Init(ctx); err != nil {
		return nil, err
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, name, description, content, created_at, updated_at FROM instructions ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("instructions: list: %w", err)
	}
	defer rows.Close()
	var out []Instruction
	for rows.Next() {
		var in Instruction
		var created, updated string
		if err := rows.Scan(&in.ID, &in.Name, &in.Description, &in.Content, &created, &updated); err != nil {
			return nil, fmt.Errorf("instructions: scan: %w", err)
		}
		in.CreatedAt = parseTime(created)
		in.UpdatedAt = parseTime(updated)
		out = append(out, in)
	}
	return out, rows.Err()
}

// ListWithInstalls returns all instructions with their installs populated.
func (s *Store) ListWithInstalls(ctx context.Context) ([]Instruction, error) {
	list, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	for i := range list {
		installs, err := installsFor(ctx, db, list[i].ID)
		if err != nil {
			return nil, err
		}
		list[i].Installs = installs
	}
	return list, nil
}

// Get returns one instruction (with installs) by id.
func (s *Store) Get(ctx context.Context, id int64) (Instruction, error) {
	if err := s.Init(ctx); err != nil {
		return Instruction{}, err
	}
	db, err := s.open()
	if err != nil {
		return Instruction{}, err
	}
	defer db.Close()
	return getTx(ctx, db, id)
}

func getTx(ctx context.Context, q queryer, id int64) (Instruction, error) {
	var in Instruction
	var created, updated string
	err := q.QueryRowContext(ctx, `SELECT id, name, description, content, created_at, updated_at FROM instructions WHERE id = ?`, id).
		Scan(&in.ID, &in.Name, &in.Description, &in.Content, &created, &updated)
	if err == sql.ErrNoRows {
		return Instruction{}, ErrInstructionNotFound
	}
	if err != nil {
		return Instruction{}, fmt.Errorf("instructions: get: %w", err)
	}
	in.CreatedAt = parseTime(created)
	in.UpdatedAt = parseTime(updated)
	installs, err := installsFor(ctx, q, id)
	if err != nil {
		return Instruction{}, err
	}
	in.Installs = installs
	return in, nil
}

// queryer abstracts *sql.DB and *sql.Tx for shared read helpers.
type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func installsFor(ctx context.Context, q queryer, id int64) ([]Install, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, app, level, project_dir, target_path, link_kind, created_at FROM instruction_installs WHERE instruction_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("instructions: list installs: %w", err)
	}
	defer rows.Close()
	var out []Install
	for rows.Next() {
		var ins Install
		var created string
		if err := rows.Scan(&ins.ID, &ins.App, &ins.Level, &ins.ProjectDir, &ins.TargetPath, &ins.LinkKind, &created); err != nil {
			return nil, fmt.Errorf("instructions: scan install: %w", err)
		}
		ins.CreatedAt = parseTime(created)
		out = append(out, ins)
	}
	return out, rows.Err()
}

// nameExists reports whether a row with name exists, optionally excluding an id.
func nameExists(ctx context.Context, q queryer, name string, excludeID int64) (bool, error) {
	var existingID int64
	err := q.QueryRowContext(ctx, `SELECT id FROM instructions WHERE name = ?`, name).Scan(&existingID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return existingID != excludeID, nil
}

// writeManagedFile atomically writes content to the managed file for name.
func writeManagedFile(name, content string) error {
	if err := os.MkdirAll(managedDir(), 0o755); err != nil {
		return fmt.Errorf("instructions: mkdir managed dir: %w", err)
	}
	dest := managedFilePath(name)
	tmp, err := os.CreateTemp(managedDir(), ".tmp-*")
	if err != nil {
		return fmt.Errorf("instructions: tmp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("instructions: write tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("instructions: close tmp: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("instructions: rename managed file: %w", err)
	}
	return nil
}

// Create persists a new instruction and writes its managed file.
func (s *Store) Create(ctx context.Context, name, desc, content string) (Instruction, error) {
	if err := validateName(name); err != nil {
		return Instruction{}, err
	}
	if err := s.Init(ctx); err != nil {
		return Instruction{}, err
	}
	db, err := s.open()
	if err != nil {
		return Instruction{}, err
	}
	defer db.Close()

	exists, err := nameExists(ctx, db, name, 0)
	if err != nil {
		return Instruction{}, fmt.Errorf("instructions: check name: %w", err)
	}
	if exists {
		return Instruction{}, fmt.Errorf("%w: %q", ErrDuplicateName, name)
	}

	if err := writeManagedFile(name, content); err != nil {
		return Instruction{}, err
	}

	now := time.Now().UTC().Format(rfc)
	res, err := db.ExecContext(ctx, `INSERT INTO instructions(name, description, content, created_at, updated_at) VALUES(?, ?, ?, ?, ?)`, name, desc, content, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Instruction{}, fmt.Errorf("%w: %q", ErrDuplicateName, name)
		}
		return Instruction{}, fmt.Errorf("instructions: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

// Update changes an instruction's name/description/content, keeps the managed
// file in sync, and re-targets symlink installs when the safe-name changes.
func (s *Store) Update(ctx context.Context, id int64, name, desc, content string) (Instruction, error) {
	if err := validateName(name); err != nil {
		return Instruction{}, err
	}
	if err := s.Init(ctx); err != nil {
		return Instruction{}, err
	}
	db, err := s.open()
	if err != nil {
		return Instruction{}, err
	}
	defer db.Close()

	current, err := getTx(ctx, db, id)
	if err != nil {
		return Instruction{}, err
	}

	exists, err := nameExists(ctx, db, name, id)
	if err != nil {
		return Instruction{}, fmt.Errorf("instructions: check name: %w", err)
	}
	if exists {
		return Instruction{}, fmt.Errorf("%w: %q", ErrDuplicateName, name)
	}

	oldFile := managedFilePath(current.Name)
	newFile := managedFilePath(name)

	if err := writeManagedFile(name, content); err != nil {
		return Instruction{}, err
	}

	now := time.Now().UTC().Format(rfc)
	if _, err := db.ExecContext(ctx, `UPDATE instructions SET name = ?, description = ?, content = ?, updated_at = ? WHERE id = ?`, name, desc, content, now, id); err != nil {
		// Best-effort rollback of the managed file content.
		_ = writeManagedFile(current.Name, current.Content)
		if strings.Contains(err.Error(), "UNIQUE") {
			return Instruction{}, fmt.Errorf("%w: %q", ErrDuplicateName, name)
		}
		return Instruction{}, fmt.Errorf("instructions: update: %w", err)
	}

	if oldFile != newFile {
		// Re-target every symlink install that pointed at the old managed file.
		installs, err := installsFor(ctx, db, id)
		if err != nil {
			return Instruction{}, err
		}
		for _, ins := range installs {
			if ins.LinkKind != "symlink" {
				continue // copy installs go stale until re-install
			}
			retargetSymlink(ins.TargetPath, newFile)
		}
		os.Remove(oldFile)
	}

	return s.Get(ctx, id)
}

// Delete uninstalls every install, removes the managed file, and deletes rows.
func (s *Store) Delete(ctx context.Context, id int64) error {
	if err := s.Init(ctx); err != nil {
		return err
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	current, err := getTx(ctx, db, id)
	if err != nil {
		return err
	}

	for _, ins := range current.Installs {
		removeInstalledFile(ins, managedFilePath(current.Name))
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM instruction_installs WHERE instruction_id = ?`, id); err != nil {
		return fmt.Errorf("instructions: delete installs: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM instructions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("instructions: delete: %w", err)
	}
	os.Remove(managedFilePath(current.Name))
	return nil
}

// retargetSymlink replaces the symlink at target so it points at newSrc.
func retargetSymlink(target, newSrc string) {
	info, err := os.Lstat(target)
	if err != nil {
		return
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return
	}
	os.Remove(target)
	_ = os.Symlink(newSrc, target)
}
