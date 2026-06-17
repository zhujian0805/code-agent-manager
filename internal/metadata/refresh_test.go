package metadata

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

// writeCatalogFile writes content to root/rel, creating parent dirs. Used by
// fake fetchers that don't have a *testing.T in scope.
func writeCatalogFile(root, rel, content string) error {
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

// dedupFetcher serves two repos: an awesome-list "pointer" catalog whose single
// row links to a real source repo, and that source repo scanned directly with a
// concrete SKILL.md. Both should resolve to the same indexed item.
type dedupFetcher struct{}

func (dedupFetcher) Fetch(owner, repo, branch, dest string) (string, error) {
	root := filepath.Join(dest, repo+"-"+branch)
	switch owner + "/" + repo {
	case "Chat2AnyLLM/awesome-claude-skills":
		// Pointer catalog: one row linking to the real skill in obra/superpowers.
		_ = writeCatalogFile(root, "FULL-SKILLS.md",
			"| Skill | Description | Author |\n| --- | --- | --- |\n"+
				"| [golang-testing](https://github.com/obra/superpowers/tree/main/skills/golang-testing) | Go testing | obra |\n")
	case "obra/superpowers":
		// The real source repo, scanned directly.
		_ = writeCatalogFile(root, "skills/golang-testing/SKILL.md",
			"---\nname: golang-testing\ndescription: Go testing\n---\nbody")
	}
	return root, nil
}

func TestFetchAndDiscoverDedupesCatalogPointerWithDirectScan(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("CAM_CACHE_DIR", filepath.Join(dir, "cache"))

	store := NewStore(filepath.Join(dir, "cam.db"))
	svc := NewService(store).WithFetcher(dedupFetcher{})

	jobs := []repoJob{
		{owner: "Chat2AnyLLM", repo: "awesome-claude-skills", branch: "main", catalogFile: "FULL-SKILLS.md"},
		{owner: "obra", repo: "superpowers", branch: "main"},
	}

	var items []Item
	for r := range svc.fetchAndDiscover(ctx, "skill", entities.KindSkill, filepath.Join(dir, "metadata-repos"), jobs) {
		if r.err != "" {
			t.Fatalf("unexpected error: %s", r.err)
		}
		items = append(items, r.items...)
	}

	// Both jobs must resolve to the same source-attributed install key.
	var matched []Item
	for _, it := range items {
		if it.Name == "golang-testing" {
			matched = append(matched, it)
		}
	}
	if len(matched) != 2 {
		t.Fatalf("expected 2 candidate items (catalog + direct), got %d: %+v", len(matched), items)
	}
	for _, it := range matched {
		if it.InstallKey != "obra/superpowers:golang-testing" {
			t.Fatalf("catalog row not attributed to source repo; install key = %q", it.InstallKey)
		}
		if it.RepoOwner != "obra" || it.RepoName != "superpowers" {
			t.Fatalf("expected attribution to obra/superpowers, got %s/%s", it.RepoOwner, it.RepoName)
		}
	}

	// Upserting both collapses them to a single row via the unique constraint.
	if _, err := store.UpsertItems(ctx, items); err != nil {
		t.Fatalf("UpsertItems: %v", err)
	}
	got, err := store.GetItem(ctx, "skill", "obra/superpowers:golang-testing")
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if got.RepoOwner != "obra" || got.RepoName != "superpowers" {
		t.Fatalf("stored item mis-attributed: %+v", got)
	}
	if got.ItemPath != "skills/golang-testing" {
		t.Fatalf("expected item path skills/golang-testing, got %q", got.ItemPath)
	}
}
