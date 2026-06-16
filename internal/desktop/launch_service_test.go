package desktop

import (
	"path/filepath"
	"testing"
)

func TestLaunchServiceDryRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	providerService := NewProviderService(path)
	_, _ = providerService.Init()
	enabled := true
	_, err := providerService.Add(ProviderInput{
		Name: "local", Endpoint: "http://localhost:4000/v1", Clients: []string{"claude"}, Models: []string{"demo-model"}, Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("add provider: %v", err)
	}

	launch := NewLaunchService(path)
	providers, err := launch.ListProvidersForTool("claude")
	if err != nil {
		t.Fatalf("providers for tool: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected one provider, got %+v", providers)
	}

	plan, err := launch.DryRun("claude", "local", "demo-model", []string{"--help"})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if plan.Provider.Name != "local" || plan.Model != "demo-model" || plan.Command == "" {
		t.Fatalf("unexpected launch plan: %+v", plan)
	}
}
