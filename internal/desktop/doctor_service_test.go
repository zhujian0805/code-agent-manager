package desktop

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDoctorServiceRunChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	_, _ = NewProviderService(path).Init()
	service := NewDoctorService("test", path)

	checks, err := service.RunChecks(context.Background())
	if err != nil {
		t.Fatalf("run checks: %v", err)
	}
	if len(checks) == 0 {
		t.Fatal("expected checks")
	}
	if len(service.ListChecks()) == 0 {
		t.Fatal("expected check names")
	}
}
