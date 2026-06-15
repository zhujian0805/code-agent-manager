package providers

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveModels_StaticList(t *testing.T) {
	ep := Endpoint{Models: []string{"m1", "m2"}, ListModelsCmd: "echo never"}
	got, err := ResolveModels(ep, "ep", time.Hour, t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "m1" || got[1] != "m2" {
		t.Fatalf("expected static list, got %v", got)
	}
}

func TestResolveModels_StaticListIsCopy(t *testing.T) {
	ep := Endpoint{Models: []string{"m1", "m2"}}
	got, _ := ResolveModels(ep, "ep", time.Hour, t.TempDir(), os.Getenv)
	got[0] = "mutated"
	if ep.Models[0] != "m1" {
		t.Fatalf("ResolveModels returned a shared slice; backing array mutated")
	}
}

func TestResolveModels_NeitherStaticNorCmd(t *testing.T) {
	got, err := ResolveModels(Endpoint{}, "ep", time.Hour, t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestResolveModels_DynamicCacheMiss(t *testing.T) {
	cacheDir := t.TempDir()
	ep := Endpoint{ListModelsCmd: "printf 'alpha\\nbeta\\n'"}
	got, err := ResolveModels(ep, "ep", time.Hour, cacheDir, os.Getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"alpha", "beta"}
	if !equalSlices(got, want) {
		t.Fatalf("models = %v, want %v", got, want)
	}
	// Cache file written.
	raw, err := os.ReadFile(filepath.Join(cacheDir, "ep.json"))
	if err != nil {
		t.Fatalf("cache not written: %v", err)
	}
	var entry cacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		t.Fatalf("cache parse: %v", err)
	}
	if !equalSlices(entry.Models, want) {
		t.Fatalf("cached models = %v, want %v", entry.Models, want)
	}
	if entry.FetchedAt.IsZero() {
		t.Fatalf("cached FetchedAt is zero")
	}
}

func TestResolveModels_DynamicCacheHit(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "ep.json")
	entry := cacheEntry{Models: []string{"cached-m"}, FetchedAt: time.Now()}
	payload, _ := json.Marshal(entry)
	if err := os.WriteFile(cachePath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	// Command would fail if run; cache hit must short-circuit.
	ep := Endpoint{ListModelsCmd: "exit 7"}
	got, err := ResolveModels(ep, "ep", time.Hour, cacheDir, os.Getenv)
	if err != nil {
		t.Fatalf("cache hit should not invoke cmd: %v", err)
	}
	if len(got) != 1 || got[0] != "cached-m" {
		t.Fatalf("expected cached models, got %v", got)
	}
}

func TestResolveModels_DynamicStale(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "ep.json")
	stale := cacheEntry{Models: []string{"old"}, FetchedAt: time.Now().Add(-2 * time.Hour)}
	payload, _ := json.Marshal(stale)
	_ = os.WriteFile(cachePath, payload, 0o600)

	ep := Endpoint{ListModelsCmd: "echo fresh"}
	got, err := ResolveModels(ep, "ep", time.Hour, cacheDir, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "fresh" {
		t.Fatalf("expected fresh exec, got %v", got)
	}
}

func TestResolveModels_NonZeroExit(t *testing.T) {
	cacheDir := t.TempDir()
	ep := Endpoint{ListModelsCmd: "exit 3"}
	_, err := ResolveModels(ep, "ep", time.Hour, cacheDir, os.Getenv)
	if err == nil {
		t.Fatalf("expected error on non-zero exit")
	}
	if !strings.Contains(err.Error(), "exited 3") {
		t.Fatalf("error %v should mention exit code", err)
	}
	if _, statErr := os.Stat(filepath.Join(cacheDir, "ep.json")); statErr == nil {
		t.Fatalf("cache should not be written on failure")
	}
}

func TestResolveModels_EmptyStdout(t *testing.T) {
	ep := Endpoint{ListModelsCmd: "true"} // no output, exit 0
	_, err := ResolveModels(ep, "ep", time.Hour, t.TempDir(), os.Getenv)
	if !errors.Is(err, ErrEmptyModelList) {
		t.Fatalf("expected ErrEmptyModelList, got %v", err)
	}
}

func TestResolveModels_StripsProxiesByDefault(t *testing.T) {
	t.Setenv("http_proxy", "should-be-stripped")
	t.Setenv("HTTPS_PROXY", "should-be-stripped")
	ep := Endpoint{ListModelsCmd: `printf '%s\n%s\n' "${http_proxy:-EMPTY}" "${HTTPS_PROXY:-EMPTY}"`}
	got, err := ResolveModels(ep, "ep", time.Hour, t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range got {
		if line != "EMPTY" {
			t.Fatalf("proxy leaked into discovery env: %v", got)
		}
	}
}

func TestResolveModels_KeepsProxiesWhenRequested(t *testing.T) {
	t.Setenv("http_proxy", "kept-value")
	ep := Endpoint{
		ListModelsCmd:   `printf '%s\n' "${http_proxy:-EMPTY}"`,
		KeepProxyConfig: true,
	}
	got, err := ResolveModels(ep, "ep", time.Hour, t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "kept-value" {
		t.Fatalf("expected proxy to pass through, got %v", got)
	}
}

func TestResolveModels_ExposesEndpointAndApiKey(t *testing.T) {
	t.Setenv("CAM_TEST_KEY", "secret-123")
	ep := Endpoint{
		Endpoint:      "https://x.example",
		APIKeyEnv:     "CAM_TEST_KEY",
		ListModelsCmd: `printf '%s\n%s\n' "$endpoint" "$api_key"`,
	}
	got, err := ResolveModels(ep, "ep", time.Hour, t.TempDir(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://x.example", "secret-123"}
	if !equalSlices(got, want) {
		t.Fatalf("models = %v, want %v", got, want)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
