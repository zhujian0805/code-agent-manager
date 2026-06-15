package envfile_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/envfile"
)

func TestFindReturnsCustomPathWhenItExists(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, ".env")
	if err := os.WriteFile(custom, []byte("FOO=bar\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := envfile.Find(custom, false)
	if err != nil {
		t.Fatalf("Find unexpected error: %v", err)
	}
	if got != custom {
		t.Fatalf("Find = %q, want %q", got, custom)
	}
}

func TestFindStrictMissingCustomReturnsError(t *testing.T) {
	_, err := envfile.Find("/nonexistent/.env", true)
	if !errors.Is(err, envfile.ErrNotFound) {
		t.Fatalf("Find strict missing err = %v, want ErrNotFound", err)
	}
}

func TestFindWalksUpwardUntilGit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("FOO=bar\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, "nested", "deep")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	got, err := envfile.Find("", false)
	if err != nil {
		t.Fatalf("Find err = %v", err)
	}
	if got != filepath.Join(root, ".env") {
		t.Fatalf("Find = %q, want %q", got, filepath.Join(root, ".env"))
	}
}

func TestFindFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CAM_CONFIG_DIR", filepath.Join(home, "cfg"))
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("FOO=bar\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Walk-upward must not find a sibling .env: chdir to an isolated tempdir
	// with a .git marker so the walk stops immediately.
	other := t.TempDir()
	if err := os.MkdirAll(filepath.Join(other, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	got, err := envfile.Find("", false)
	if err != nil {
		t.Fatalf("Find err = %v", err)
	}
	if got != filepath.Join(home, ".env") {
		t.Fatalf("Find = %q, want %q", got, filepath.Join(home, ".env"))
	}
}

func TestLoadParsesKeyValuePairs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\n" +
		"\n" +
		"FOO=bar\n" +
		"QUOTED=\"with spaces\"\n" +
		"SINGLE='other'\n" +
		"export EXPORTED=value\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := envfile.Load(path)
	if err != nil {
		t.Fatalf("Load err = %v", err)
	}
	want := map[string]string{
		"FOO":      "bar",
		"QUOTED":   "with spaces",
		"SINGLE":   "other",
		"EXPORTED": "value",
	}
	if len(got) != len(want) {
		t.Fatalf("Load got %d entries, want %d (%v)", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("Load %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestLoadRejectsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("INVALID LINE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := envfile.Load(path); err == nil {
		t.Fatal("Load should reject malformed line")
	}
}

func TestApplyToProcessDoesNotOverwrite(t *testing.T) {
	t.Setenv("CAM_TEST_EXISTING", "existing")
	envfile.ApplyToProcess(map[string]string{
		"CAM_TEST_EXISTING": "new",
		"CAM_TEST_NEW":      "fresh",
	})
	if got := os.Getenv("CAM_TEST_EXISTING"); got != "existing" {
		t.Fatalf("existing var overwritten: %q", got)
	}
	if got := os.Getenv("CAM_TEST_NEW"); got != "fresh" {
		t.Fatalf("new var not set: %q", got)
	}
	_ = os.Unsetenv("CAM_TEST_NEW")
}

func TestLoadKeysAreReturnedInDeterministicOrderWhenSorted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("B=2\nA=1\nC=3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := envfile.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if want := []string{"A", "B", "C"}; !equalSlice(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
}

func equalSlice(a, b []string) bool {
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
