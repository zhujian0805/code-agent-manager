package metadata

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestDetailReturnsManifestContent verifies that Detail decorates the indexed
// item with installed-app status and the on-demand manifest body fetched from
// the (fake) source repo.
func TestDetailReturnsManifestContent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("CAM_CACHE_DIR", filepath.Join(dir, "cache"))
	s := NewStore(filepath.Join(dir, "cam.db"))
	svc := NewService(s).WithFetcher(fakeFetcher{})

	if _, err := svc.RefreshAll(ctx); err != nil {
		t.Fatalf("RefreshAll: %v", err)
	}
	skills, _ := s.Search(ctx, SearchQuery{Query: "", Kind: "skill", Limit: 1})
	if len(skills) == 0 {
		t.Fatal("no skills indexed")
	}
	key := skills[0].InstallKey

	detail, err := svc.Detail(ctx, "skill", key)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if detail.Item.InstallKey != key {
		t.Fatalf("expected item %q, got %q", key, detail.Item.InstallKey)
	}
	// fakeFetcher writes a SKILL.md with frontmatter; the content must come back.
	if !strings.Contains(detail.Content, "name:") {
		t.Fatalf("expected manifest content, got %q", detail.Content)
	}
	if !strings.HasSuffix(detail.Manifest, "SKILL.md") {
		t.Fatalf("expected manifest path ending in SKILL.md, got %q", detail.Manifest)
	}
}

// TestDetailUnknownItem verifies Detail surfaces a lookup error for a missing
// install key rather than panicking or returning an empty success.
func TestDetailUnknownItem(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	s := NewStore(filepath.Join(dir, "cam.db"))
	svc := NewService(s).WithFetcher(fakeFetcher{})

	if _, err := svc.Detail(ctx, "skill", "nonexistent/repo:missing"); err == nil {
		t.Fatal("expected error for unknown install key")
	}
}
