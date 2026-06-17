# Metadata Index Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a shared Go metadata layer that ingests agent/skill/plugin repo configs into SQLite for local-first search, refresh, and install — used by both CLI and desktop sidecar.

**Architecture:** A new `internal/metadata` package owns the SQLite schema (tables in the existing `cam.db`), refresh logic (leveraging `internal/repoconfig`), search, and install orchestration (delegating to `internal/entities`). CLI gets a `cam metadata` command group; sidecar gets `/api/metadata/*` endpoints.

**Tech Stack:** Go, modernc.org/sqlite (already in go.mod), cobra CLI, net/http sidecar

---

### Task 1: Metadata SQLite schema and store

**Files:**
- Create: `internal/metadata/store.go`
- Create: `internal/metadata/store_test.go`

- [ ] **Step 1: Write the failing test for schema init**

```go
// internal/metadata/store_test.go
package metadata

import (
	"context"
	"path/filepath"
	"testing"
)

func TestInitCreatesTablesIdempotently(t *testing.T) {
	ctx := context.Background()
	s := NewStore(filepath.Join(t.TempDir(), "cam.db"))
	if err := s.Init(ctx); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	// Second call must not fail.
	if err := s.Init(ctx); err != nil {
		t.Fatalf("second Init: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metadata/ -run TestInitCreatesTablesIdempotently -v`
Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Write minimal store implementation**

```go
// internal/metadata/store.go
package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
	_ "modernc.org/sqlite"
)

// Store is the SQLite-backed metadata index.
type Store struct {
	path string
}

// NewStore returns a Store. If path is empty, uses the default cam.db location.
func NewStore(path string) *Store {
	if path == "" {
		if p := os.Getenv("CAM_DB_PATH"); p != "" {
			path = p
		} else {
			path = filepath.Join(pathutil.ConfigDir(), "cam.db")
		}
	}
	return &Store{path: path}
}

// Init creates metadata tables if they do not exist.
func (s *Store) Init(ctx context.Context) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("metadata: init schema: %w", err)
	}
	return nil
}

func (s *Store) open() (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return nil, fmt.Errorf("metadata: mkdir: %w", err)
	}
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return nil, fmt.Errorf("metadata: open %s: %w", s.path, err)
	}
	return db, nil
}

const schemaSQL = `
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS metadata_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL,
  source_key TEXT NOT NULL,
  owner TEXT NOT NULL DEFAULT '',
  repo TEXT NOT NULL DEFAULT '',
  branch TEXT NOT NULL DEFAULT 'main',
  path TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  source_file TEXT NOT NULL DEFAULT '',
  last_refreshed_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  UNIQUE(kind, source_key)
);

CREATE TABLE IF NOT EXISTS metadata_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  source_id INTEGER NOT NULL DEFAULT 0,
  repo_owner TEXT NOT NULL DEFAULT '',
  repo_name TEXT NOT NULL DEFAULT '',
  repo_branch TEXT NOT NULL DEFAULT 'main',
  item_path TEXT NOT NULL DEFAULT '',
  install_key TEXT NOT NULL DEFAULT '',
  target_apps TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  installed INTEGER NOT NULL DEFAULT 0,
  installed_targets TEXT NOT NULL DEFAULT '',
  last_seen_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  UNIQUE(kind, install_key)
);

CREATE INDEX IF NOT EXISTS idx_metadata_items_name ON metadata_items(name);
CREATE INDEX IF NOT EXISTS idx_metadata_items_kind ON metadata_items(kind);
CREATE INDEX IF NOT EXISTS idx_metadata_items_repo ON metadata_items(repo_owner, repo_name);
`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/metadata/ -run TestInitCreatesTablesIdempotently -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/store.go internal/metadata/store_test.go
git commit -m "feat(metadata): add SQLite schema and store init"
```

---

### Task 2: Upsert and search methods on Store

**Files:**
- Modify: `internal/metadata/store.go`
- Modify: `internal/metadata/store_test.go`
- Create: `internal/metadata/models.go`

- [ ] **Step 1: Write the failing test for upsert + search**

```go
// Append to internal/metadata/store_test.go

func TestUpsertAndSearch(t *testing.T) {
	ctx := context.Background()
	s := NewStore(filepath.Join(t.TempDir(), "cam.db"))
	if err := s.Init(ctx); err != nil {
		t.Fatal(err)
	}

	item := Item{
		Kind:       "skill",
		Name:       "deep-research",
		Description: "Deep research harness for multi-source reports",
		RepoOwner:  "obra",
		RepoName:   "superpowers",
		RepoBranch: "main",
		ItemPath:   "skills/deep-research",
		InstallKey: "obra/superpowers:deep-research",
		TargetApps: "claude,codex",
	}
	if err := s.UpsertItem(ctx, item); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}

	results, err := s.Search(ctx, SearchQuery{Query: "research"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "deep-research" {
		t.Fatalf("unexpected name: %s", results[0].Name)
	}
}

func TestSearchByKindFilter(t *testing.T) {
	ctx := context.Background()
	s := NewStore(filepath.Join(t.TempDir(), "cam.db"))
	if err := s.Init(ctx); err != nil {
		t.Fatal(err)
	}
	s.UpsertItem(ctx, Item{Kind: "skill", Name: "my-skill", InstallKey: "a/b:my-skill", Description: "a skill"})
	s.UpsertItem(ctx, Item{Kind: "agent", Name: "my-agent", InstallKey: "a/b:my-agent", Description: "an agent"})

	results, _ := s.Search(ctx, SearchQuery{Query: "my", Kind: "skill"})
	if len(results) != 1 || results[0].Name != "my-skill" {
		t.Fatalf("kind filter failed: got %+v", results)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metadata/ -run "TestUpsert|TestSearchByKind" -v`
Expected: FAIL — Item, UpsertItem, Search, SearchQuery not defined.

- [ ] **Step 3: Write models and store methods**

```go
// internal/metadata/models.go
package metadata

// Item represents a single searchable/installable metadata record.
type Item struct {
	ID               int64  `json:"id"`
	Kind             string `json:"kind"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	SourceID         int64  `json:"source_id"`
	RepoOwner        string `json:"repo_owner"`
	RepoName         string `json:"repo_name"`
	RepoBranch       string `json:"repo_branch"`
	ItemPath         string `json:"item_path"`
	InstallKey       string `json:"install_key"`
	TargetApps       string `json:"target_apps"`
	MetadataJSON     string `json:"metadata_json"`
	Installed        bool   `json:"installed"`
	InstalledTargets string `json:"installed_targets"`
	LastSeenAt       string `json:"last_seen_at"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// Source represents a configured metadata repository source.
type Source struct {
	ID              int64  `json:"id"`
	Kind            string `json:"kind"`
	SourceKey       string `json:"source_key"`
	Owner           string `json:"owner"`
	Repo            string `json:"repo"`
	Branch          string `json:"branch"`
	Path            string `json:"path"`
	Enabled         bool   `json:"enabled"`
	SourceFile      string `json:"source_file"`
	LastRefreshedAt string `json:"last_refreshed_at"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// SearchQuery configures a metadata search.
type SearchQuery struct {
	Query  string `json:"query"`
	Kind   string `json:"kind"`
	Limit  int    `json:"limit"`
}

// RefreshSummary reports the result of a refresh operation.
type RefreshSummary struct {
	SourcesScanned int      `json:"sources_scanned"`
	ItemsAdded     int      `json:"items_added"`
	ItemsUpdated   int      `json:"items_updated"`
	ItemsStale     int      `json:"items_stale"`
	FailedSources  []string `json:"failed_sources"`
}
```

Then add to `internal/metadata/store.go`:

```go
// UpsertItem inserts or updates a metadata item by (kind, install_key).
func (s *Store) UpsertItem(ctx context.Context, item Item) error {
	if err := s.Init(ctx); err != nil {
		return err
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	now := timeNow()
	_, err = db.ExecContext(ctx, `
		INSERT INTO metadata_items(kind, name, description, source_id, repo_owner, repo_name, repo_branch, item_path, install_key, target_apps, metadata_json, installed, installed_targets, last_seen_at, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(kind, install_key) DO UPDATE SET
			name=excluded.name, description=excluded.description, source_id=excluded.source_id,
			repo_owner=excluded.repo_owner, repo_name=excluded.repo_name, repo_branch=excluded.repo_branch,
			item_path=excluded.item_path, target_apps=excluded.target_apps, metadata_json=excluded.metadata_json,
			installed=excluded.installed, installed_targets=excluded.installed_targets,
			last_seen_at=excluded.last_seen_at, updated_at=excluded.updated_at`,
		item.Kind, item.Name, item.Description, item.SourceID,
		item.RepoOwner, item.RepoName, item.RepoBranch, item.ItemPath,
		item.InstallKey, item.TargetApps, coalesce(item.MetadataJSON, "{}"),
		boolToInt(item.Installed), item.InstalledTargets, now, now, now)
	if err != nil {
		return fmt.Errorf("metadata: upsert item: %w", err)
	}
	return nil
}

// Search queries the metadata_items table with LIKE matching.
func (s *Store) Search(ctx context.Context, q SearchQuery) ([]Item, error) {
	if err := s.Init(ctx); err != nil {
		return nil, err
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	pattern := "%" + q.Query + "%"

	var rows *sql.Rows
	if q.Kind != "" {
		rows, err = db.QueryContext(ctx, `
			SELECT id, kind, name, description, source_id, repo_owner, repo_name, repo_branch,
			       item_path, install_key, target_apps, metadata_json, installed, installed_targets,
			       last_seen_at, created_at, updated_at
			FROM metadata_items
			WHERE kind = ? AND (name LIKE ? OR description LIKE ? OR repo_owner LIKE ? OR repo_name LIKE ? OR install_key LIKE ?)
			ORDER BY name LIMIT ?`,
			q.Kind, pattern, pattern, pattern, pattern, pattern, limit)
	} else {
		rows, err = db.QueryContext(ctx, `
			SELECT id, kind, name, description, source_id, repo_owner, repo_name, repo_branch,
			       item_path, install_key, target_apps, metadata_json, installed, installed_targets,
			       last_seen_at, created_at, updated_at
			FROM metadata_items
			WHERE name LIKE ? OR description LIKE ? OR repo_owner LIKE ? OR repo_name LIKE ? OR install_key LIKE ?
			ORDER BY name LIMIT ?`,
			pattern, pattern, pattern, pattern, pattern, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("metadata: search: %w", err)
	}
	defer rows.Close()
	return scanItems(rows)
}

func scanItems(rows *sql.Rows) ([]Item, error) {
	var items []Item
	for rows.Next() {
		var it Item
		var installed int
		if err := rows.Scan(&it.ID, &it.Kind, &it.Name, &it.Description, &it.SourceID,
			&it.RepoOwner, &it.RepoName, &it.RepoBranch, &it.ItemPath, &it.InstallKey,
			&it.TargetApps, &it.MetadataJSON, &installed, &it.InstalledTargets,
			&it.LastSeenAt, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("metadata: scan item: %w", err)
		}
		it.Installed = installed != 0
		items = append(items, it)
	}
	return items, rows.Err()
}

func timeNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func coalesce(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
```

Add `"time"` to the import list of store.go.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/metadata/ -run "TestUpsert|TestSearchByKind" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/
git commit -m "feat(metadata): add upsert and search methods"
```

---

### Task 3: Refresh logic — load repo config into SQLite

**Files:**
- Create: `internal/metadata/refresh.go`
- Create: `internal/metadata/refresh_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/metadata/refresh_test.go
package metadata

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRefreshFromLocalRepoFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cam.db")
	s := NewStore(dbPath)

	// Create a fake skill_repos.json
	cfgDir := filepath.Join(dir, "config")
	os.MkdirAll(cfgDir, 0o755)

	skillRepos := map[string]any{
		"obra/superpowers": map[string]any{
			"owner":      "obra",
			"name":       "superpowers",
			"branch":     "main",
			"enabled":    true,
			"skillsPath": "skills",
		},
	}
	raw, _ := json.MarshalIndent(skillRepos, "", "  ")
	os.WriteFile(filepath.Join(cfgDir, "skill_repos.json"), raw, 0o644)

	agentRepos := map[string]any{
		"iannuttall/claude-agents": map[string]any{
			"owner":      "iannuttall",
			"name":       "claude-agents",
			"branch":     "main",
			"enabled":    true,
			"agentsPath": "agents",
		},
	}
	raw2, _ := json.MarshalIndent(agentRepos, "", "  ")
	os.WriteFile(filepath.Join(cfgDir, "agent_repos.json"), raw2, 0o644)

	svc := NewService(s)
	summary, err := svc.RefreshFromFiles(ctx, cfgDir)
	if err != nil {
		t.Fatalf("RefreshFromFiles: %v", err)
	}
	if summary.SourcesScanned != 2 {
		t.Fatalf("expected 2 sources scanned, got %d", summary.SourcesScanned)
	}
	if summary.ItemsAdded < 2 {
		t.Fatalf("expected at least 2 items, got %d added", summary.ItemsAdded)
	}

	// Search should find the indexed items.
	results, err := s.Search(ctx, SearchQuery{Query: "superpowers"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for 'superpowers'")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metadata/ -run TestRefreshFromLocalRepoFiles -v`
Expected: FAIL — NewService, RefreshFromFiles not defined.

- [ ] **Step 3: Write the Service and refresh logic**

```go
// internal/metadata/refresh.go
package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Service orchestrates metadata operations.
type Service struct {
	store *Store
}

// NewService constructs a metadata Service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// RefreshFromFiles reads *_repos.json files from cfgDir and indexes entries into SQLite.
func (svc *Service) RefreshFromFiles(ctx context.Context, cfgDir string) (RefreshSummary, error) {
	if err := svc.store.Init(ctx); err != nil {
		return RefreshSummary{}, err
	}

	summary := RefreshSummary{}
	files := []struct {
		filename string
		kind     string
	}{
		{"skill_repos.json", "skill"},
		{"agent_repos.json", "agent"},
		{"plugin_repos.json", "plugin"},
	}

	for _, f := range files {
		path := filepath.Join(cfgDir, f.filename)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			summary.FailedSources = append(summary.FailedSources, f.filename)
			continue
		}

		var repos map[string]json.RawMessage
		if err := json.Unmarshal(data, &repos); err != nil {
			summary.FailedSources = append(summary.FailedSources, f.filename)
			continue
		}

		summary.SourcesScanned++
		for key, raw := range repos {
			var entry repoEntry
			if err := json.Unmarshal(raw, &entry); err != nil {
				continue
			}
			if entry.Enabled != nil && !*entry.Enabled {
				continue
			}

			item := Item{
				Kind:        f.kind,
				Name:        entryName(entry, key),
				Description: entry.Description,
				RepoOwner:  effectiveOwner(entry),
				RepoName:   effectiveName(entry),
				RepoBranch: effectiveBranch(entry),
				ItemPath:    entryPath(entry, f.kind),
				InstallKey:  key,
				TargetApps: strings.Join(defaultTargetApps(f.kind), ","),
			}
			if err := svc.store.UpsertItem(ctx, item); err != nil {
				continue
			}
			summary.ItemsAdded++
		}
	}

	return summary, nil
}

// repoEntry mirrors the JSON shape used in *_repos.json files.
type repoEntry struct {
	Owner       string   `json:"owner,omitempty"`
	Name        string   `json:"name,omitempty"`
	Branch      string   `json:"branch,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
	SkillsPath  string   `json:"skillsPath,omitempty"`
	AgentsPath  string   `json:"agentsPath,omitempty"`
	PluginPath  string   `json:"pluginPath,omitempty"`
	Description string   `json:"description,omitempty"`
	RepoOwner   string   `json:"repoOwner,omitempty"`
	RepoName    string   `json:"repoName,omitempty"`
	RepoBranch  string   `json:"repoBranch,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
}

func effectiveOwner(e repoEntry) string {
	if e.RepoOwner != "" {
		return e.RepoOwner
	}
	return e.Owner
}

func effectiveName(e repoEntry) string {
	if e.RepoName != "" {
		return e.RepoName
	}
	return e.Name
}

func effectiveBranch(e repoEntry) string {
	if e.RepoBranch != "" {
		return e.RepoBranch
	}
	if e.Branch != "" {
		return e.Branch
	}
	return "main"
}

func entryName(e repoEntry, key string) string {
	if e.Name != "" {
		return e.Name
	}
	parts := strings.Split(key, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return key
}

func entryPath(e repoEntry, kind string) string {
	switch kind {
	case "skill":
		return e.SkillsPath
	case "agent":
		return e.AgentsPath
	case "plugin":
		return e.PluginPath
	}
	return ""
}

func defaultTargetApps(kind string) []string {
	switch kind {
	case "skill":
		return []string{"claude", "codex", "gemini", "copilot"}
	case "agent":
		return []string{"claude", "codex", "gemini", "copilot"}
	case "plugin":
		return []string{"claude", "codebuddy"}
	}
	return []string{"claude"}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/metadata/ -run TestRefreshFromLocalRepoFiles -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/refresh.go internal/metadata/refresh_test.go
git commit -m "feat(metadata): refresh from local repo JSON files"
```

---

### Task 4: Integrate repoconfig.LoadAll for full refresh

**Files:**
- Modify: `internal/metadata/refresh.go`
- Modify: `internal/metadata/refresh_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Append to internal/metadata/refresh_test.go

func TestRefreshAll(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "cam.db"))

	svc := NewService(s)
	// RefreshAll uses repoconfig.LoadEnabled internally which reads bundled repos.
	summary, err := svc.RefreshAll(ctx)
	if err != nil {
		t.Fatalf("RefreshAll: %v", err)
	}
	// Should have indexed items from bundled repos.
	if summary.ItemsAdded == 0 {
		t.Fatal("expected at least some items from bundled repos")
	}

	// Search should return something.
	results, _ := s.Search(ctx, SearchQuery{Query: ""})
	if len(results) == 0 {
		t.Fatal("expected results after refresh")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metadata/ -run TestRefreshAll -v`
Expected: FAIL — RefreshAll not defined.

- [ ] **Step 3: Implement RefreshAll using repoconfig**

Add to `internal/metadata/refresh.go`:

```go
import (
	"github.com/chat2anyllm/code-agent-manager/internal/entities"
	"github.com/chat2anyllm/code-agent-manager/internal/repoconfig"
)

// RefreshAll loads all enabled repo entries from repoconfig (bundled + config.yaml + local sources)
// and indexes them into SQLite.
func (svc *Service) RefreshAll(ctx context.Context) (RefreshSummary, error) {
	if err := svc.store.Init(ctx); err != nil {
		return RefreshSummary{}, err
	}
	summary := RefreshSummary{}

	kinds := []struct {
		kind       string
		entityKind entities.Kind
	}{
		{"skill", entities.KindSkill},
		{"agent", entities.KindAgent},
		{"plugin", entities.KindPlugin},
	}

	for _, k := range kinds {
		repos, err := repoconfig.LoadEnabled(k.entityKind)
		if err != nil {
			summary.FailedSources = append(summary.FailedSources, fmt.Sprintf("%s: %v", k.kind, err))
			continue
		}
		summary.SourcesScanned += len(repos)

		for key, entry := range repos {
			item := Item{
				Kind:        k.kind,
				Name:        repoEntryName(entry, key),
				Description: entry.Description,
				RepoOwner:  entry.EffectiveOwner(),
				RepoName:   entry.EffectiveName(),
				RepoBranch: entry.EffectiveBranch(),
				ItemPath:    entry.SubPath(k.entityKind),
				InstallKey:  key,
				TargetApps: strings.Join(defaultTargetApps(k.kind), ","),
			}
			if err := svc.store.UpsertItem(ctx, item); err != nil {
				continue
			}
			summary.ItemsAdded++
		}
	}

	return summary, nil
}

func repoEntryName(e repoconfig.RepoEntry, key string) string {
	if e.Name != "" {
		return e.Name
	}
	parts := strings.Split(key, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return key
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/metadata/ -run TestRefreshAll -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/refresh.go internal/metadata/refresh_test.go
git commit -m "feat(metadata): RefreshAll using repoconfig for full source loading"
```

---

### Task 5: GetItem and MarkInstalled methods

**Files:**
- Modify: `internal/metadata/store.go`
- Modify: `internal/metadata/store_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// Append to internal/metadata/store_test.go

func TestGetItem(t *testing.T) {
	ctx := context.Background()
	s := NewStore(filepath.Join(t.TempDir(), "cam.db"))
	s.Init(ctx)
	s.UpsertItem(ctx, Item{Kind: "skill", Name: "test-skill", InstallKey: "a/b:test-skill"})

	item, err := s.GetItem(ctx, "skill", "a/b:test-skill")
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if item.Name != "test-skill" {
		t.Fatalf("unexpected name: %s", item.Name)
	}
}

func TestMarkInstalled(t *testing.T) {
	ctx := context.Background()
	s := NewStore(filepath.Join(t.TempDir(), "cam.db"))
	s.Init(ctx)
	s.UpsertItem(ctx, Item{Kind: "skill", Name: "x", InstallKey: "a/b:x"})

	if err := s.MarkInstalled(ctx, "skill", "a/b:x", "claude"); err != nil {
		t.Fatalf("MarkInstalled: %v", err)
	}
	item, _ := s.GetItem(ctx, "skill", "a/b:x")
	if !item.Installed {
		t.Fatal("expected installed=true")
	}
	if item.InstalledTargets != "claude" {
		t.Fatalf("unexpected targets: %s", item.InstalledTargets)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metadata/ -run "TestGetItem|TestMarkInstalled" -v`
Expected: FAIL — methods not defined.

- [ ] **Step 3: Implement GetItem and MarkInstalled**

Add to `internal/metadata/store.go`:

```go
// GetItem returns a single item by kind and install_key.
func (s *Store) GetItem(ctx context.Context, kind, installKey string) (Item, error) {
	if err := s.Init(ctx); err != nil {
		return Item{}, err
	}
	db, err := s.open()
	if err != nil {
		return Item{}, err
	}
	defer db.Close()

	var it Item
	var installed int
	err = db.QueryRowContext(ctx, `
		SELECT id, kind, name, description, source_id, repo_owner, repo_name, repo_branch,
		       item_path, install_key, target_apps, metadata_json, installed, installed_targets,
		       last_seen_at, created_at, updated_at
		FROM metadata_items WHERE kind = ? AND install_key = ?`, kind, installKey).
		Scan(&it.ID, &it.Kind, &it.Name, &it.Description, &it.SourceID,
			&it.RepoOwner, &it.RepoName, &it.RepoBranch, &it.ItemPath, &it.InstallKey,
			&it.TargetApps, &it.MetadataJSON, &installed, &it.InstalledTargets,
			&it.LastSeenAt, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		return Item{}, fmt.Errorf("metadata: get item: %w", err)
	}
	it.Installed = installed != 0
	return it, nil
}

// MarkInstalled sets the installed flag and records the target app.
func (s *Store) MarkInstalled(ctx context.Context, kind, installKey, targetApp string) error {
	if err := s.Init(ctx); err != nil {
		return err
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	now := timeNow()
	_, err = db.ExecContext(ctx, `
		UPDATE metadata_items SET installed = 1, installed_targets = ?, updated_at = ?
		WHERE kind = ? AND install_key = ?`, targetApp, now, kind, installKey)
	if err != nil {
		return fmt.Errorf("metadata: mark installed: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/metadata/ -run "TestGetItem|TestMarkInstalled" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/store.go internal/metadata/store_test.go
git commit -m "feat(metadata): add GetItem and MarkInstalled store methods"
```

---

### Task 6: CLI metadata commands

**Files:**
- Create: `internal/cli/metadata.go`
- Create: `internal/cli/cmd_metadata_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/cli/cmd_metadata_test.go
package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMetadataRefreshCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CAM_CONFIG_DIR", filepath.Join(home, ".config", "code-agent-manager"))
	t.Setenv("CAM_DB_PATH", filepath.Join(home, "cam.db"))

	stdout, _, code := runApp(t, "metadata", "refresh")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, "Refreshed metadata") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
}

func TestMetadataSearchCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CAM_CONFIG_DIR", filepath.Join(home, ".config", "code-agent-manager"))
	t.Setenv("CAM_DB_PATH", filepath.Join(home, "cam.db"))

	// Refresh first to populate DB.
	runApp(t, "metadata", "refresh")

	stdout, _, code := runApp(t, "metadata", "search", "superpowers")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	// Should find something from bundled repos.
	if !strings.Contains(stdout, "superpowers") {
		t.Fatalf("expected 'superpowers' in output:\n%s", stdout)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run "TestMetadataRefresh|TestMetadataSearch" -v`
Expected: FAIL — no metadata command registered.

- [ ] **Step 3: Implement the metadata CLI commands**

```go
// internal/cli/metadata.go
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chat2anyllm/code-agent-manager/internal/metadata"
)

func (a *App) metadataCommand(state *globalState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metadata",
		Aliases: []string{"md"},
		Short:   "Manage the metadata index for agents, skills, and plugins",
	}
	cmd.AddCommand(a.metadataRefreshCommand(state))
	cmd.AddCommand(a.metadataSearchCommand(state))
	return cmd
}

func (a *App) metadataRefreshCommand(state *globalState) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh metadata index from configured repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			store := metadata.NewStore(state.storePath)
			svc := metadata.NewService(store)

			summary, err := svc.RefreshAll(context.Background())
			if err != nil {
				return fmt.Errorf("refresh failed: %w", err)
			}
			fmt.Fprintf(out, "✓ Refreshed metadata\n")
			fmt.Fprintf(out, "  Sources scanned: %d\n", summary.SourcesScanned)
			fmt.Fprintf(out, "  Items indexed:   %d\n", summary.ItemsAdded)
			if len(summary.FailedSources) > 0 {
				fmt.Fprintf(out, "  Failed sources:  %d\n", len(summary.FailedSources))
				for _, f := range summary.FailedSources {
					fmt.Fprintf(out, "    - %s\n", f)
				}
			}
			return nil
		},
	}
}

func (a *App) metadataSearchCommand(state *globalState) *cobra.Command {
	var kindFilter string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the metadata index",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			store := metadata.NewStore(state.storePath)
			query := strings.Join(args, " ")

			results, err := store.Search(context.Background(), metadata.SearchQuery{
				Query: query,
				Kind:  kindFilter,
			})
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}
			if len(results) == 0 {
				fmt.Fprintf(out, "No results for %q\n", query)
				return nil
			}
			fmt.Fprintf(out, "Found %d result(s) for %q:\n\n", len(results), query)
			fmt.Fprintf(out, "  %-8s %-30s %-25s %s\n", "KIND", "NAME", "REPO", "DESCRIPTION")
			fmt.Fprintf(out, "  %-8s %-30s %-25s %s\n",
				strings.Repeat("─", 8), strings.Repeat("─", 30),
				strings.Repeat("─", 25), strings.Repeat("─", 40))
			for _, item := range results {
				repo := item.RepoOwner + "/" + item.RepoName
				desc := item.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				fmt.Fprintf(out, "  %-8s %-30s %-25s %s\n", item.Kind, item.Name, repo, desc)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&kindFilter, "type", "", "Filter by kind (skill, agent, plugin)")
	return cmd
}
```

Register in `internal/cli/app.go` by adding `root.AddCommand(a.metadataCommand(state))` after the plugin command line.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run "TestMetadataRefresh|TestMetadataSearch" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/metadata.go internal/cli/cmd_metadata_test.go internal/cli/app.go
git commit -m "feat(cli): add cam metadata refresh and search commands"
```

---

### Task 7: Sidecar metadata API endpoints

**Files:**
- Modify: `internal/sidecar/server.go`
- Create: `internal/sidecar/metadata_handler.go`
- Create: `internal/sidecar/metadata_handler_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/sidecar/metadata_handler_test.go
package sidecar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataSearchEndpoint(t *testing.T) {
	srv := New(Options{Version: "test"})
	handler := srv.Handler()

	// Refresh first.
	req := httptest.NewRequest(http.MethodPost, "/api/metadata/refresh", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: status %d, body %s", w.Code, w.Body.String())
	}

	// Search.
	req = httptest.NewRequest(http.MethodGet, "/api/metadata/search?q=superpowers", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("search: status %d, body %s", w.Code, w.Body.String())
	}

	var results []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Should have at least one result from bundled repos.
	if len(results) == 0 {
		t.Fatal("expected at least one search result")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sidecar/ -run TestMetadataSearchEndpoint -v`
Expected: FAIL — no handler registered for /api/metadata/*.

- [ ] **Step 3: Implement the metadata handler**

```go
// internal/sidecar/metadata_handler.go
package sidecar

import (
	"context"
	"net/http"

	"github.com/chat2anyllm/code-agent-manager/internal/metadata"
)

func (s *Server) handleMetadataRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	store := metadata.NewStore("")
	svc := metadata.NewService(store)
	summary, err := svc.RefreshAll(context.Background())
	writeResult(w, summary, err)
}

func (s *Server) handleMetadataSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	store := metadata.NewStore("")
	q := r.URL.Query().Get("q")
	kind := r.URL.Query().Get("type")

	results, err := store.Search(context.Background(), metadata.SearchQuery{
		Query: q,
		Kind:  kind,
	})
	writeResult(w, results, err)
}
```

Register in `internal/sidecar/server.go` Handler() method:

```go
mux.HandleFunc("/api/metadata/refresh", s.handleMetadataRefresh)
mux.HandleFunc("/api/metadata/search", s.handleMetadataSearch)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/sidecar/ -run TestMetadataSearchEndpoint -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/sidecar/metadata_handler.go internal/sidecar/metadata_handler_test.go internal/sidecar/server.go
git commit -m "feat(sidecar): add /api/metadata/refresh and /api/metadata/search endpoints"
```

---

### Task 8: Install via metadata service

**Files:**
- Modify: `internal/metadata/refresh.go`
- Modify: `internal/metadata/refresh_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Append to internal/metadata/refresh_test.go

func TestInstallItem(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	s := NewStore(filepath.Join(dir, "cam.db"))
	svc := NewService(s)

	// Insert an item.
	s.Init(ctx)
	s.UpsertItem(ctx, Item{
		Kind:       "skill",
		Name:       "test-install-skill",
		InstallKey: "test/repo:test-install-skill",
		RepoOwner:  "test",
		RepoName:   "repo",
		RepoBranch: "main",
		ItemPath:   "skills",
		TargetApps: "claude",
	})

	err := svc.Install(ctx, "skill", "test/repo:test-install-skill", "claude")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Check marked installed.
	item, _ := s.GetItem(ctx, "skill", "test/repo:test-install-skill")
	if !item.Installed {
		t.Fatal("expected installed=true after Install")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metadata/ -run TestInstallItem -v`
Expected: FAIL — Install method not defined.

- [ ] **Step 3: Implement Install on Service**

Add to `internal/metadata/refresh.go`:

```go
import (
	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

// Install installs an item from the metadata index into the target app.
func (svc *Service) Install(ctx context.Context, kind, installKey, targetApp string) error {
	item, err := svc.store.GetItem(ctx, kind, installKey)
	if err != nil {
		return fmt.Errorf("metadata: item not found: %w", err)
	}

	entityKind := entities.Kind(item.Kind)
	entity := entities.Entity{
		Kind:        entityKind,
		Name:        item.Name,
		Description: item.Description,
		Content:     "", // Content is fetched from repo or placeholder for now.
		Repo: &entities.RepoRef{
			Owner:  item.RepoOwner,
			Name:   item.RepoName,
			Branch: item.RepoBranch,
			Path:   item.ItemPath,
		},
	}

	if _, err := entities.InstallToApp(entity, entityKind, targetApp); err != nil {
		return fmt.Errorf("metadata: install to %s: %w", targetApp, err)
	}

	return svc.store.MarkInstalled(ctx, kind, installKey, targetApp)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/metadata/ -run TestInstallItem -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/refresh.go internal/metadata/refresh_test.go
git commit -m "feat(metadata): add Install method for shared install path"
```

---

### Task 9: CLI install command and sidecar endpoint

**Files:**
- Modify: `internal/cli/metadata.go`
- Modify: `internal/cli/cmd_metadata_test.go`
- Modify: `internal/sidecar/metadata_handler.go`

- [ ] **Step 1: Write the failing test for CLI install**

```go
// Append to internal/cli/cmd_metadata_test.go

func TestMetadataInstallCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CAM_CONFIG_DIR", filepath.Join(home, ".config", "code-agent-manager"))
	t.Setenv("CAM_DB_PATH", filepath.Join(home, "cam.db"))

	// Refresh to populate.
	runApp(t, "metadata", "refresh")

	stdout, _, code := runApp(t, "metadata", "install", "obra/superpowers", "--target", "claude")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, "Installed") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestMetadataInstall -v`
Expected: FAIL — no install subcommand.

- [ ] **Step 3: Implement install subcommand and sidecar endpoint**

Add to `internal/cli/metadata.go`:

```go
func (a *App) metadataInstallCommand(state *globalState) *cobra.Command {
	var targetApp string
	cmd := &cobra.Command{
		Use:   "install <install-key>",
		Short: "Install a metadata item to a target coding agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			store := metadata.NewStore(state.storePath)
			svc := metadata.NewService(store)
			installKey := args[0]

			// Try each kind until found.
			var err error
			for _, kind := range []string{"skill", "agent", "plugin"} {
				err = svc.Install(context.Background(), kind, installKey, targetApp)
				if err == nil {
					fmt.Fprintf(out, "✓ Installed %s to %s\n", installKey, targetApp)
					return nil
				}
			}
			return fmt.Errorf("install failed: %w", err)
		},
	}
	cmd.Flags().StringVar(&targetApp, "target", "claude", "Target coding agent (claude, codex, gemini, etc.)")
	return cmd
}
```

Register in `metadataCommand`: `cmd.AddCommand(a.metadataInstallCommand(state))`

Add to `internal/sidecar/metadata_handler.go`:

```go
func (s *Server) handleMetadataInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		Kind       string `json:"kind"`
		InstallKey string `json:"install_key"`
		TargetApp  string `json:"target_app"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	store := metadata.NewStore("")
	svc := metadata.NewService(store)
	err := svc.Install(context.Background(), input.Kind, input.InstallKey, input.TargetApp)
	writeResult(w, map[string]string{"status": "installed", "install_key": input.InstallKey, "target": input.TargetApp}, err)
}
```

Register in server.go: `mux.HandleFunc("/api/metadata/install", s.handleMetadataInstall)`

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestMetadataInstall -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/metadata.go internal/cli/cmd_metadata_test.go internal/sidecar/metadata_handler.go internal/sidecar/server.go
git commit -m "feat: add metadata install CLI command and sidecar endpoint"
```

---

### Task 10: Full integration test and final verification

**Files:**
- All metadata files

- [ ] **Step 1: Run all metadata tests**

Run: `go test ./internal/metadata/ -v`
Expected: All PASS

- [ ] **Step 2: Run all CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: All PASS

- [ ] **Step 3: Run all sidecar tests**

Run: `go test ./internal/sidecar/ -v`
Expected: All PASS

- [ ] **Step 4: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 5: Verify CLI end-to-end**

Run manually:
```bash
dist/cam metadata refresh
dist/cam metadata search superpowers
dist/cam metadata search claude-agents --type agent
```

- [ ] **Step 6: Final commit (if any fixups needed)**

```bash
git add -A
git commit -m "test: verify metadata index integration"
```
