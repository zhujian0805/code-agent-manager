package appapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/providers"
)

func TestProviderAPIInitListShow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	api := ProviderAPI{ProvidersPath: path, Env: os.Getenv}

	result, err := api.Init(context.Background())
	if err != nil {
		t.Fatalf("Init error = %v", err)
	}
	if !result.OK || result.Message == "" || result.Path != path+".db" {
		t.Fatalf("Init result = %+v, want ok message and db path", result)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("db file missing: %v", err)
	}

	file := providers.File{Endpoints: map[string]providers.Endpoint{
		"local": {
			Endpoint:        "http://localhost:4000/v1",
			APIKeyEnv:       "LOCAL_KEY",
			SupportedClient: "claude,codex",
			Models:          []string{"m1", "m2"},
			Enabled:         boolPtr(true),
		},
	}}
	if err := providers.Save(path, file); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	listed, err := api.List(context.Background())
	if err != nil {
		t.Fatalf("List error = %v", err)
	}
	if len(listed) != 1 || listed[0].Name != "local" || listed[0].Endpoint != "http://localhost:4000/v1" {
		t.Fatalf("List = %+v, want local provider", listed)
	}

	shown, err := api.Show(context.Background(), "local")
	if err != nil {
		t.Fatalf("Show error = %v", err)
	}
	if shown.Name != "local" || len(shown.Models) != 2 || len(shown.Clients) != 2 {
		t.Fatalf("Show = %+v, want local with models and clients", shown)
	}
}

func TestProviderAPIMutations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	api := ProviderAPI{ProvidersPath: path, Env: os.Getenv}
	enabled := true

	added, err := api.Add(context.Background(), ProviderInput{
		Name:            "alpha",
		Endpoint:        "https://alpha.example",
		APIKeyEnv:       "ALPHA_KEY",
		SupportedClient: "claude,codex",
		Models:          []string{"m1"},
		Enabled:         &enabled,
		Description:     "alpha provider",
	})
	if err != nil {
		t.Fatalf("Add error = %v", err)
	}
	if added.Name != "alpha" || !added.Enabled || len(added.Models) != 1 {
		t.Fatalf("Add = %+v", added)
	}

	endpoint := "https://updated.example"
	description := "updated provider"
	models := providers.ListPatch{Op: providers.ListOpReplace, Items: []string{"m2", "m3"}}
	updated, err := api.Update(context.Background(), "alpha", ProviderPatch{
		Endpoint:    &endpoint,
		Models:      &models,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if updated.Endpoint != endpoint || updated.Description != description || len(updated.Models) != 2 {
		t.Fatalf("Update = %+v", updated)
	}

	disabled, err := api.SetEnabled(context.Background(), "alpha", false)
	if err != nil {
		t.Fatalf("SetEnabled false error = %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("SetEnabled false = %+v", disabled)
	}

	renamed, err := api.Rename(context.Background(), "alpha", "beta")
	if err != nil {
		t.Fatalf("Rename error = %v", err)
	}
	if renamed.Name != "beta" {
		t.Fatalf("Rename = %+v", renamed)
	}
	if _, err := api.Show(context.Background(), "alpha"); err == nil {
		t.Fatal("old provider name should be missing")
	}

	result, err := api.Remove(context.Background(), "beta")
	if err != nil {
		t.Fatalf("Remove error = %v", err)
	}
	if !result.OK {
		t.Fatalf("Remove result = %+v", result)
	}
	listed, err := api.List(context.Background())
	if err != nil {
		t.Fatalf("List after remove error = %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("List after remove = %+v", listed)
	}
}

func TestProviderAPIResolveModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	api := ProviderAPI{ProvidersPath: path, Env: os.Getenv}
	enabled := true
	_, err := api.Add(context.Background(), ProviderInput{
		Name:            "alpha",
		Endpoint:        "https://alpha.example",
		SupportedClient: "claude",
		Models:          []string{"static-a", "static-b"},
		Enabled:         &enabled,
	})
	if err != nil {
		t.Fatalf("Add error = %v", err)
	}
	models, err := api.ResolveModels(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("ResolveModels error = %v", err)
	}
	if len(models) < 2 || models[0] != "static-a" || models[1] != "static-b" {
		t.Fatalf("models = %v", models)
	}
}

func boolPtr(v bool) *bool { return &v }
