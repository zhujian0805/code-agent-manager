package pathutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/pathutil"
)

func TestExpand(t *testing.T) {
	t.Setenv("HOME", "/home/test")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "absolute", in: "/etc/passwd", want: "/etc/passwd"},
		{name: "relative", in: "foo/bar", want: "foo/bar"},
		{name: "tilde alone", in: "~", want: "/home/test"},
		{name: "tilde slash", in: "~/", want: "/home/test"},
		{name: "tilde subdir", in: "~/cfg/file", want: "/home/test/cfg/file"},
		{name: "literal tilde mid-path", in: "/var/~/foo", want: "/var/~/foo"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := pathutil.Expand(tc.in)
			if got != tc.want {
				t.Fatalf("Expand(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestConfigDirHonorsEnvironment(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("CAM_CONFIG_DIR", "")
	if got, want := pathutil.ConfigDir(), filepath.Join("/home/test", ".config", "code-agent-manager"); got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}

	t.Setenv("CAM_CONFIG_DIR", "/tmp/override")
	if got, want := pathutil.ConfigDir(), "/tmp/override"; got != want {
		t.Fatalf("ConfigDir() = %q, want %q with override", got, want)
	}
}

func TestCacheDirHonorsEnvironment(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("CAM_CACHE_DIR", "")
	if got, want := pathutil.CacheDir(), filepath.Join("/home/test", ".cache", "code-agent-manager"); got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}

	t.Setenv("CAM_CACHE_DIR", "/tmp/cache-override")
	if got, want := pathutil.CacheDir(), "/tmp/cache-override"; got != want {
		t.Fatalf("CacheDir() = %q, want %q with override", got, want)
	}
}

func TestExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if pathutil.Exists("") {
		t.Fatal("Exists(\"\") should be false")
	}
	if pathutil.Exists(filepath.Join(dir, "missing")) {
		t.Fatal("Exists on missing path should be false")
	}

	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !pathutil.Exists(file) {
		t.Fatal("Exists on real file should be true")
	}
}
