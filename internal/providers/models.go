package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

// modelDiscoveryTimeout caps how long ResolveModels will wait for a
// list_models_cmd to produce output. Hard ceiling; not configurable.
const modelDiscoveryTimeout = 15 * time.Second

// proxyVars are the env vars stripped before invoking list_models_cmd
// when KeepProxyConfig is false. Matches the Python CAM behaviour.
var proxyVars = []string{
	"http_proxy", "HTTP_PROXY",
	"https_proxy", "HTTPS_PROXY",
	"no_proxy", "NO_PROXY",
	"all_proxy", "ALL_PROXY",
}

// ErrEmptyModelList signals that the discovery command succeeded but
// returned no models. Wizard treats this the same as a non-zero exit:
// offer manual entry or step back to provider selection.
var ErrEmptyModelList = errors.New("providers: list_models_cmd returned no models")

// cacheEntry is the on-disk format under cacheDir/<epName>.json.
type cacheEntry struct {
	Models    []string  `json:"models"`
	FetchedAt time.Time `json:"fetched_at"`
}

// ResolveModels returns the model list to present for endpoint ep,
// identified by epName. Priority:
//
//  1. ep.Models when non-empty → returned verbatim, no cache I/O.
//  2. ep.ListModelsCmd when non-empty → run with env vars
//     endpoint=ep.Endpoint and api_key=ResolveAPIKey(ep, getenv),
//     proxies stripped unless ep.KeepProxyConfig. Hard timeout.
//     stdout split on \n; empty/whitespace lines dropped; result
//     cached for cacheTTL.
//  3. neither set → empty slice, nil error.
//
// On step-2 timeout, non-zero exit, or empty output, returns
// ([], err). Callers treat err as a recoverable signal.
//
// cacheDir defaults to pathutil.CacheDir()/models when empty.
func ResolveModels(
	ep Endpoint,
	epName string,
	cacheTTL time.Duration,
	cacheDir string,
	getenv func(string) string,
) ([]string, error) {
	if len(ep.Models) > 0 {
		return append([]string(nil), ep.Models...), nil
	}
	if ep.ListModelsCmd == "" {
		return nil, nil
	}

	if cacheDir == "" {
		cacheDir = filepath.Join(pathutil.CacheDir(), "models")
	}
	cachePath := filepath.Join(cacheDir, epName+".json")

	if cached, ok := readModelsCache(cachePath, cacheTTL); ok {
		return cached, nil
	}

	models, err := runListModelsCmd(ep, epName, getenv)
	if err != nil {
		return nil, err
	}

	if werr := writeModelsCache(cachePath, models); werr != nil {
		// Cache write failure must not break a successful discovery.
		// Surface a warning by wrapping; callers can ignore via the
		// models return value being valid.
		return models, fmt.Errorf("providers: cache write %s: %w", cachePath, werr)
	}
	return models, nil
}

func readModelsCache(path string, ttl time.Duration) ([]string, bool) {
	if ttl <= 0 {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entry cacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, false
	}
	if entry.FetchedAt.IsZero() || time.Since(entry.FetchedAt) > ttl {
		return nil, false
	}
	if len(entry.Models) == 0 {
		return nil, false
	}
	return append([]string(nil), entry.Models...), true
}

func writeModelsCache(path string, models []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	entry := cacheEntry{Models: models, FetchedAt: time.Now()}
	payload, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func runListModelsCmd(ep Endpoint, epName string, getenv func(string) string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), modelDiscoveryTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", ep.ListModelsCmd)
	cmd.Env = buildDiscoveryEnv(ep, getenv)

	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("providers: list_models_cmd for %s timed out after %s", epName, modelDiscoveryTimeout)
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("providers: list_models_cmd for %s exited %d: %s",
				epName, exit.ExitCode(), strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, fmt.Errorf("providers: list_models_cmd for %s: %w", epName, err)
	}

	models := parseModelsOutput(string(out))
	if len(models) == 0 {
		return nil, ErrEmptyModelList
	}
	return models, nil
}

// parseModelsOutput splits stdout on \n and drops empty/whitespace lines.
// Each surviving line is trimmed.
func parseModelsOutput(raw string) []string {
	var models []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		models = append(models, trimmed)
	}
	return models
}

// buildDiscoveryEnv constructs the env slice passed to list_models_cmd.
// Starts from os.Environ() (via getenv if injected via tests), strips
// proxy vars when KeepProxyConfig is false, then layers endpoint +
// api_key on top.
func buildDiscoveryEnv(ep Endpoint, getenv func(string) string) []string {
	if getenv == nil {
		getenv = os.Getenv
	}
	apiKey := ResolveAPIKey(ep, getenv)

	current := os.Environ()
	out := make([]string, 0, len(current)+2)
	for _, kv := range current {
		name, _, _ := strings.Cut(kv, "=")
		if !ep.KeepProxyConfig && isProxyVar(name) {
			continue
		}
		// Avoid letting an inherited endpoint/api_key override.
		if name == "endpoint" || name == "api_key" {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "endpoint="+ep.Endpoint)
	out = append(out, "api_key="+apiKey)
	return out
}

func isProxyVar(name string) bool {
	for _, p := range proxyVars {
		if name == p {
			return true
		}
	}
	return false
}
