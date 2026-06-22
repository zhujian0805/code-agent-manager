package prompts

import (
	"context"
	"strings"
	"testing"
)

// TestClaudeLibraryEmbedded verifies the bundled Claude prompt library loads,
// has the expected size, and that every entry carries real content (the old
// implementation shipped placeholders with empty Content).
func TestClaudeLibraryEmbedded(t *testing.T) {
	svc := NewService()
	prompts, err := svc.FetchClaudePrompts(context.Background())
	if err != nil {
		t.Fatalf("FetchClaudePrompts: %v", err)
	}
	if len(prompts) != 52 {
		t.Fatalf("expected 52 Claude prompts, got %d", len(prompts))
	}
	for _, p := range prompts {
		if strings.TrimSpace(p.Title) == "" {
			t.Errorf("prompt with empty title: %+v", p)
		}
		if strings.TrimSpace(p.Content) == "" {
			t.Errorf("prompt %q has empty content", p.Title)
		}
	}
}

// TestSyncClaudeLibraryStoresAll is the regression test for the "Claude Prompt
// Library shows 0" bug: syncing must store every prompt (unique source_url means
// no collisions on the (source, source_url) index) and SyncAll must include it.
func TestSyncClaudeLibraryStoresAll(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CAM_CONFIG_DIR", dir)
	ctx := context.Background()
	svc := NewService()

	n, err := svc.syncClaudeLibrary(ctx)
	if err != nil {
		t.Fatalf("syncClaudeLibrary: %v", err)
	}
	if n != 52 {
		t.Fatalf("expected 52 prompts synced, got %d", n)
	}

	count, err := svc.store.CountPrompts(ctx, "claude", "")
	if err != nil {
		t.Fatalf("CountPrompts: %v", err)
	}
	if count != 52 {
		t.Fatalf("expected 52 claude prompts stored, got %d (source_url collision?)", count)
	}
}

// TestSlugifyUnique guards the root cause of the collisions: distinct titles must
// produce distinct slugs so their source_urls don't clash.
func TestSlugifyUnique(t *testing.T) {
	cases := map[string]string{
		"Get oriented in a new repository": "get-oriented-in-a-new-repository",
		"Fix a precise visual bug":         "fix-a-precise-visual-bug",
		"  Trim  me  ":                     "trim-me",
		"Already-slugged":                  "already-slugged",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestParsePromptsFromMarkdownStripsFences verifies the markdown parser no longer
// leaves stray ``` fences in extracted prompt content.
func TestParsePromptsFromMarkdownStripsFences(t *testing.T) {
	md := "## Text Summarization\nIntro text.\n\n*Prompt:*\n```\nExplain antibiotics\n\nA:\n```\n\n*Output:*\n```\nsome output\n```\n"
	got := parsePromptsFromMarkdown(md)
	if len(got) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(got))
	}
	if strings.Contains(got[0].Content, "```") {
		t.Errorf("content still contains code fence: %q", got[0].Content)
	}
	if !strings.Contains(got[0].Content, "Explain antibiotics") {
		t.Errorf("content missing prompt body: %q", got[0].Content)
	}
}
