package appstate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/providers"
)

func TestProviderCRUD(t *testing.T) {
	ctx := context.Background()
	store := New(filepath.Join(t.TempDir(), "cam.db"))
	enabled := true

	if err := store.AddProvider(ctx, "alpha", providers.Endpoint{Endpoint: "https://alpha.example", SupportedClient: "claude,codex", Models: []string{"m1"}, Enabled: &enabled}); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	file, err := store.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(file.Endpoints) != 1 || file.Endpoints["alpha"].Endpoint != "https://alpha.example" {
		t.Fatalf("providers = %+v", file.Endpoints)
	}

	endpoint := "https://updated.example"
	models := providers.ListPatch{Op: providers.ListOpReplace, Items: []string{"m2", "m3"}}
	if err := store.UpdateProvider(ctx, "alpha", providers.Patch{Endpoint: &endpoint, Models: &models}); err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	file, _ = store.ListProviders(ctx)
	if file.Endpoints["alpha"].Endpoint != endpoint || len(file.Endpoints["alpha"].Models) != 2 {
		t.Fatalf("updated provider = %+v", file.Endpoints["alpha"])
	}

	if err := store.SetProviderEnabled(ctx, "alpha", false); err != nil {
		t.Fatalf("SetProviderEnabled: %v", err)
	}
	file, _ = store.ListProviders(ctx)
	if file.Endpoints["alpha"].IsEnabled() {
		t.Fatal("provider should be disabled")
	}

	if err := store.RenameProvider(ctx, "alpha", "beta"); err != nil {
		t.Fatalf("RenameProvider: %v", err)
	}
	file, _ = store.ListProviders(ctx)
	if _, ok := file.Endpoints["alpha"]; ok {
		t.Fatal("old provider name still exists")
	}
	if _, ok := file.Endpoints["beta"]; !ok {
		t.Fatal("new provider name missing")
	}

	if !store.RemoveProvider(ctx, "beta") {
		t.Fatal("RemoveProvider returned false")
	}
	file, _ = store.ListProviders(ctx)
	if len(file.Endpoints) != 0 {
		t.Fatalf("providers after remove = %+v", file.Endpoints)
	}
}

func TestImportProvidersJSON(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "providers.json")
	if err := providers.Save(jsonPath, providers.File{Endpoints: map[string]providers.Endpoint{"local": {Endpoint: "http://localhost:4000/v1", Models: []string{"m1"}}}}); err != nil {
		t.Fatalf("save providers json: %v", err)
	}
	store := New(filepath.Join(dir, "cam.db"))
	if err := store.ImportProvidersJSON(ctx, jsonPath); err != nil {
		t.Fatalf("ImportProvidersJSON: %v", err)
	}
	file, err := store.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if got := file.Endpoints["local"].Endpoint; got != "http://localhost:4000/v1" {
		t.Fatalf("imported endpoint = %q", got)
	}
}

func TestAppStateKeyValue(t *testing.T) {
	ctx := context.Background()
	store := New(filepath.Join(t.TempDir(), "cam.db"))
	want := map[string]any{"sidebar": "wide"}
	if err := store.SetState(ctx, "ui", want); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	var got map[string]any
	ok, err := store.GetState(ctx, "ui", &got)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if !ok || got["sidebar"] != "wide" {
		t.Fatalf("state = ok:%v value:%+v", ok, got)
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("db file missing: %v", err)
	}
}
