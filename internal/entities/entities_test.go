package entities_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

func TestStoreRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CAM_CONFIG_DIR", filepath.Join(home, "cfg"))

	store := entities.NewStore(entities.KindPrompt)
	if err := store.Put(entities.Entity{Name: "demo", Content: "Hello", Apps: []string{"claude"}}); err != nil {
		t.Fatalf("Put err = %v", err)
	}
	all, err := store.All()
	if err != nil {
		t.Fatalf("All err = %v", err)
	}
	if len(all) != 1 || all[0].Name != "demo" {
		t.Fatalf("All = %v", all)
	}
	got, err := store.Get("demo")
	if err != nil {
		t.Fatalf("Get err = %v", err)
	}
	if got.Content != "Hello" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.Kind != entities.KindPrompt {
		t.Fatalf("kind = %q", got.Kind)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt should be set")
	}
	removed, err := store.Delete("demo")
	if err != nil || !removed {
		t.Fatalf("Delete = %v, %v", removed, err)
	}
	if all, _ := store.All(); len(all) != 0 {
		t.Fatalf("post-delete all = %v", all)
	}
}

func TestInstallPromptWritesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := entities.InstallToApp(entities.Entity{Name: "demo", Content: "Hi"}, entities.KindPrompt, "claude")
	if err != nil {
		t.Fatalf("InstallToApp err = %v", err)
	}
	if path != filepath.Join(home, ".claude/CLAUDE.md") {
		t.Fatalf("path = %q", path)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "Hi" {
		t.Fatalf("written content = %q", data)
	}
}

func TestInstallSkillCreatesDirectoryWithMarkdown(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir, err := entities.InstallToApp(entities.Entity{Name: "deep-research", Content: "skill body"}, entities.KindSkill, "claude")
	if err != nil {
		t.Fatalf("InstallToApp err = %v", err)
	}
	want := filepath.Join(home, ".claude/skills/deep-research")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if string(data) != "skill body" {
		t.Fatalf("SKILL.md = %q", data)
	}
}

func TestUninstallSkillRemovesDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := entities.InstallToApp(entities.Entity{Name: "demo", Content: "body"}, entities.KindSkill, "claude"); err != nil {
		t.Fatal(err)
	}
	_, removed, err := entities.UninstallFromApp("demo", entities.KindSkill, "claude")
	if err != nil {
		t.Fatalf("Uninstall err = %v", err)
	}
	if !removed {
		t.Fatal("expected removed=true")
	}
	if _, removed, _ := entities.UninstallFromApp("demo", entities.KindSkill, "claude"); removed {
		t.Fatal("second uninstall should not report removed=true")
	}
}

func TestSupportedAppsContainsExpectedSets(t *testing.T) {
	for _, kind := range []entities.Kind{entities.KindPrompt, entities.KindSkill, entities.KindAgent, entities.KindPlugin} {
		apps := entities.SupportedApps(kind)
		if len(apps) == 0 {
			t.Fatalf("kind %s should have apps", kind)
		}
		// claude should be supported across all kinds.
		found := false
		for _, a := range apps {
			if a == "claude" {
				found = true
			}
		}
		if !found {
			t.Fatalf("kind %s missing claude: %v", kind, apps)
		}
	}
}
