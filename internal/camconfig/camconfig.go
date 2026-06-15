// Package camconfig loads the top-level CAM config.yaml file, which describes
// where CAM should fetch repository definitions and how to cache them.
//
// User files override the bundled defaults shipped with the binary; a missing
// user file silently falls back to the bundled config so a fresh installation
// works without any setup.
package camconfig

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
	"gopkg.in/yaml.v3"
)

//go:embed embed/config.yaml
var bundledYAML []byte

// CamConfig models the relevant subset of config.yaml.  Unknown fields are
// ignored so future Python additions don't break parsing.
type CamConfig struct {
	Cache        CacheConfig            `yaml:"cache"`
	Repositories map[string]RepoSources `yaml:"repositories"`
}

// CacheConfig models the cache section.
type CacheConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Directory  string `yaml:"directory"`
	TTLSeconds int    `yaml:"ttl_seconds"`
}

// RepoSources wraps a list of repository sources.
type RepoSources struct {
	Sources []RepoSource `yaml:"sources"`
}

// RepoSource describes a single source (local file or remote URL).
type RepoSource struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}

// DefaultPath returns the canonical config.yaml location under CAM's config
// directory.
func DefaultPath() string {
	return filepath.Join(pathutil.ConfigDir(), "config.yaml")
}

// Load returns the parsed config.yaml.  When path is empty the canonical
// location is consulted; when that file is missing the bundled defaults are
// returned instead so the CLI keeps working on a fresh install.
func Load(path string) (CamConfig, error) {
	if path == "" {
		path = DefaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Bundled()
		}
		return CamConfig{}, fmt.Errorf("camconfig: read %s: %w", path, err)
	}
	cfg, parseErr := parse(data)
	if parseErr != nil {
		return CamConfig{}, fmt.Errorf("camconfig: parse %s: %w", path, parseErr)
	}
	cfg.Cache.Directory = pathutil.Expand(cfg.Cache.Directory)
	return cfg, nil
}

// Bundled returns the parsed bundled config.yaml shipped with the binary.
// Used as a fallback when no user file is present and exposed so tests can
// assert on the bundled defaults directly.
func Bundled() (CamConfig, error) {
	cfg, err := parse(bundledYAML)
	if err != nil {
		return CamConfig{}, fmt.Errorf("camconfig: bundled parse: %w", err)
	}
	cfg.Cache.Directory = pathutil.Expand(cfg.Cache.Directory)
	return cfg, nil
}

func parse(data []byte) (CamConfig, error) {
	var cfg CamConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return CamConfig{}, err
	}
	if cfg.Repositories == nil {
		cfg.Repositories = map[string]RepoSources{}
	}
	return cfg, nil
}
