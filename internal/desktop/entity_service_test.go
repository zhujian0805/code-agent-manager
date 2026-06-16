package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEntityServiceSaveSearchUninstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CAM_CONFIG_DIR", dir)
	service := NewEntityService()

	saved, err := service.Save("skill", EntityDTO{Name: "demo", Description: "Demo skill", Tags: []string{"test"}})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.Kind != "skill" || saved.Name != "demo" {
		t.Fatalf("unexpected entity: %+v", saved)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills.json")); err != nil {
		t.Fatalf("expected skills store: %v", err)
	}

	matches, err := service.Search("skill", "demo")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %+v", matches)
	}

	if _, err := service.Uninstall("skill", "demo"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
}

func TestEntityServiceInvalidKind(t *testing.T) {
	_, err := NewEntityService().List("unknown")
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
}
