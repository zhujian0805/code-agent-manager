package metadata

import (
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

func TestParseCatalogMarkdownBasic(t *testing.T) {
	content := `# Catalog

| Skill | Description | Repository |
| --- | --- | --- |
| deep-research | Multi-source research | obra/superpowers |
| code-review | Reviews code | tools/review |
`

	res := parseCatalogMarkdown(content, "FULL-SKILLS.md", entities.KindSkill)
	if len(res) != 2 {
		t.Fatalf("expected 2 catalog rows, got %d: %+v", len(res), res)
	}
	if res[0].Name != "deep-research" || res[0].Description != "Multi-source research" {
		t.Fatalf("unexpected first resource: %+v", res[0])
	}
	if res[0].RelPath != "FULL-SKILLS.md" || res[0].ManifestRel != "FULL-SKILLS.md" {
		t.Fatalf("expected catalog rel paths, got %+v", res[0])
	}
}

func TestParseCatalogMarkdownCleansLinksAndCode(t *testing.T) {
	content := "| Agent | Description |\n" +
		"| --- | --- |\n" +
		"| [`security-auditor`](https://github.com/example/repo/blob/main/security-auditor.md) | [Audits code](https://example.com) |\n" +
		"| `test-runner` | Runs tests |\n"

	res := parseCatalogMarkdown(content, "README.md", entities.KindAgent)
	if len(res) != 2 {
		t.Fatalf("expected 2 catalog rows, got %d: %+v", len(res), res)
	}
	if res[0].Name != "security-auditor" {
		t.Fatalf("expected cleaned link/code name, got %q", res[0].Name)
	}
	if res[0].Description != "Audits code" {
		t.Fatalf("expected cleaned link description, got %q", res[0].Description)
	}
	if res[1].Name != "test-runner" {
		t.Fatalf("expected cleaned code name, got %q", res[1].Name)
	}
}

func TestParseCatalogMarkdownDeduplicatesSameSourceNames(t *testing.T) {
	content := `| Plugin | Description |
| --- | --- |
| [lint](https://example.com/a) | First |
| lint | Duplicate |
`

	res := parseCatalogMarkdown(content, "README.md", entities.KindPlugin)
	if len(res) != 1 {
		t.Fatalf("expected duplicate plugin names to collapse, got %d: %+v", len(res), res)
	}
	if res[0].Description != "First" {
		t.Fatalf("expected first duplicate to win, got %q", res[0].Description)
	}
}

func TestParseCatalogMarkdownKeepsSameNameFromDifferentSources(t *testing.T) {
	content := `| Plugin | Description |
| --- | --- |
| [lint](https://github.com/one/repo/tree/main/plugins/lint) | First |
| [lint](https://github.com/two/repo/tree/main/plugins/lint) | Second |
`

	res := parseCatalogMarkdown(content, "README.md", entities.KindPlugin)
	if len(res) != 2 {
		t.Fatalf("expected same plugin name from two sources to be preserved, got %d: %+v", len(res), res)
	}
	if res[0].InstallKeyName != "one/repo:lint" || res[1].InstallKeyName != "two/repo:lint" {
		t.Fatalf("unexpected install key names: %+v", res)
	}
}

func TestParseCatalogMarkdownIgnoresProse(t *testing.T) {
	res := parseCatalogMarkdown("# No tables here\n\nJust prose.", "README.md", entities.KindSkill)
	if len(res) != 0 {
		t.Fatalf("expected no resources from prose, got %+v", res)
	}
}

func TestParseCatalogMarkdownAttributesRowToSourceRepo(t *testing.T) {
	// awesome-list catalogs list skills as links to their real source repo. The
	// row must be attributed to that source repo (with branch + in-repo path)
	// so it merges with a direct scan instead of duplicating.
	content := "| Skill | Description | Author |\n" +
		"| --- | --- | --- |\n" +
		"| [golang-testing](https://github.com/obra/superpowers/tree/main/skills/golang-testing) | Go testing | obra |\n"

	res := parseCatalogMarkdown(content, "FULL-SKILLS.md", entities.KindSkill)
	if len(res) != 1 {
		t.Fatalf("expected 1 catalog row, got %d: %+v", len(res), res)
	}
	r := res[0]
	if r.SourceOwner != "obra" || r.SourceRepo != "superpowers" {
		t.Fatalf("expected source obra/superpowers, got %s/%s", r.SourceOwner, r.SourceRepo)
	}
	if r.SourceBranch != "main" {
		t.Fatalf("expected source branch main, got %q", r.SourceBranch)
	}
	if r.SourcePath != "skills/golang-testing" {
		t.Fatalf("expected source path skills/golang-testing, got %q", r.SourcePath)
	}
	if r.InstallKeyName != "obra/superpowers:golang-testing" {
		t.Fatalf("unexpected install key name: %q", r.InstallKeyName)
	}
}

func TestParseCatalogMarkdownIgnoresNonGithubLinks(t *testing.T) {
	// A link to a non-GitHub host must not be mistaken for a source repo.
	content := "| Plugin | Description |\n" +
		"| --- | --- |\n" +
		"| [lint](https://example.com/a) | First |\n"

	res := parseCatalogMarkdown(content, "README.md", entities.KindPlugin)
	if len(res) != 1 {
		t.Fatalf("expected 1 row, got %d: %+v", len(res), res)
	}
	if res[0].SourceOwner != "" || res[0].SourceRepo != "" {
		t.Fatalf("non-GitHub link should not set source attribution: %+v", res[0])
	}
}

func TestDiscoverCatalogResourcesIntegration(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "FULL-AGENTS.md"), `| Agent | Description |
| --- | --- |
| planner | Plans work |
`)

	res := DiscoverCatalogResources(root, "FULL-AGENTS.md", entities.KindAgent)
	if len(res) != 1 {
		t.Fatalf("expected 1 catalog agent, got %d: %+v", len(res), res)
	}
	if res[0].Name != "planner" || res[0].Description != "Plans work" {
		t.Fatalf("unexpected catalog resource: %+v", res[0])
	}
}

func TestInferCatalogFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "FULL-SKILLS.md"), `| Skill | Description |
| --- | --- |
| deploy | Deploys apps |
`)

	if got := inferCatalogFile(root, entities.KindSkill); got != "FULL-SKILLS.md" {
		t.Fatalf("inferCatalogFile() = %q, want FULL-SKILLS.md", got)
	}
}

func TestInferCatalogFileIgnoresReadme(t *testing.T) {
	// A README with a table must NOT be mistaken for a resource catalog — that
	// is how documentation tables (feature lists, command tables) leaked in as
	// fake skills/plugins. Only generated FULL-*.md files are auto-discovered.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), `# Plugins

| Plugin | Description |
| --- | --- |
| docs | Writes docs |
`)

	if got := inferCatalogFile(root, entities.KindPlugin); got != "" {
		t.Fatalf("inferCatalogFile() = %q, want \"\" (README must not be inferred)", got)
	}
}
