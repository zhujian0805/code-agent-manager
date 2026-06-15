// Package envfile loads dotenv files used by the CLI for sensitive defaults.
//
// The implementation intentionally uses only the standard library: the parser
// supports KEY=VALUE pairs, quoted values, comments and blank lines, which is
// enough for the limited set of variables CAM ships.
package envfile

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

// ErrNotFound is returned by Find when no .env file is located.
var ErrNotFound = errors.New("envfile: no .env file found")

// Find returns the first .env file CAM should load.
//
// When custom is non-empty and exists, it is returned.  When custom is set but
// missing and strict is true, ErrNotFound is returned.  When custom is empty,
// Find walks the current working directory upward until it locates a .env file
// or hits a directory that contains a .git entry (matching Python's
// dotenv.find_dotenv behaviour), then falls back to ~/.env and
// ~/.config/code-agent-manager/.env.
func Find(custom string, strict bool) (string, error) {
	if custom != "" {
		expanded := pathutil.Expand(custom)
		if pathutil.Exists(expanded) {
			return expanded, nil
		}
		if strict {
			return "", ErrNotFound
		}
	}

	if dir, err := os.Getwd(); err == nil {
		if found := walkUpward(dir); found != "" {
			return found, nil
		}
	}

	for _, candidate := range []string{
		filepath.Join(pathutil.Home(), ".env"),
		filepath.Join(pathutil.ConfigDir(), ".env"),
	} {
		if pathutil.Exists(candidate) {
			return candidate, nil
		}
	}
	return "", ErrNotFound
}

func walkUpward(start string) string {
	dir := start
	for {
		candidate := filepath.Join(dir, ".env")
		if pathutil.Exists(candidate) {
			return candidate
		}
		if pathutil.Exists(filepath.Join(dir, ".git")) {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Load reads path and returns the parsed key/value pairs.  An error from Load
// indicates a syntactically invalid file; missing files are surfaced as the
// standard os.ErrNotExist wrapped value so callers can treat them as optional.
func Load(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("envfile: open %s: %w", path, err)
	}
	defer file.Close()

	out := map[string]string{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		if strings.HasPrefix(raw, "export ") {
			raw = strings.TrimSpace(strings.TrimPrefix(raw, "export "))
		}
		idx := strings.IndexByte(raw, '=')
		if idx <= 0 {
			return nil, fmt.Errorf("envfile: %s:%d: missing '='", path, line)
		}
		key := strings.TrimSpace(raw[:idx])
		value := strings.TrimSpace(raw[idx+1:])
		if !validKey(key) {
			return nil, fmt.Errorf("envfile: %s:%d: invalid key %q", path, line, key)
		}
		value = unquote(value)
		out[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("envfile: read %s: %w", path, err)
	}
	return out, nil
}

func validKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func unquote(value string) string {
	if len(value) < 2 {
		return value
	}
	first, last := value[0], value[len(value)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

// ApplyToProcess sets every key/value in vars on the current process
// environment, leaving existing values intact when the same key is already
// present.  This mirrors Python's dotenv.load_dotenv default behaviour.
func ApplyToProcess(vars map[string]string) {
	for key, value := range vars {
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}
