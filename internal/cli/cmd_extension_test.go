package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// `cam extension browse` parses the geminicli.com extensions.json payload and
// renders a numbered list.  We seed a fake HTTP server via the
// CAM_EXTENSION_BROWSE_URL override so the test never touches the network.
func TestExtensionBrowseRendersExtensionsFromCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"extensions": []map[string]any{
				{"extensionName": "alpha", "extensionDescription": "first", "fullName": "owner/alpha", "url": "https://example.com/alpha", "stars": 5},
				{"extensionName": "beta", "repoDescription": "second", "fullName": "owner/beta", "url": "https://example.com/beta", "stars": 99},
			},
		})
	}))
	defer srv.Close()
	t.Setenv("CAM_EXTENSION_BROWSE_URL", srv.URL)

	stdout, stderr, code := execute(t, "extension", "browse")
	if code != 0 {
		t.Fatalf("exit = %d; stderr=%s", code, stderr)
	}
	for _, want := range []string{
		"Available Gemini Extensions (2)",
		"alpha", "first", "owner/alpha", "★ 5",
		"beta", "second", "owner/beta", "★ 99",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("browse missing %q\nstdout:\n%s", want, stdout)
		}
	}
}

// Empty catalog payload prints the no-extensions notice.
func TestExtensionBrowseEmptyCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	t.Setenv("CAM_EXTENSION_BROWSE_URL", srv.URL)

	stdout, _, code := execute(t, "extension", "browse")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "No extensions found.") {
		t.Fatalf("missing empty notice:\n%s", stdout)
	}
}

// Catalog HTTP error surfaces as a non-zero exit.
func TestExtensionBrowseHTTPFailureReportsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("CAM_EXTENSION_BROWSE_URL", srv.URL)

	_, stderr, code := execute(t, "extension", "browse")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "HTTP 500") {
		t.Fatalf("stderr missing HTTP status: %s", stderr)
	}
}

// Passthrough subcommands invoke the `gemini` binary.  We can't test the
// happy path through `execute` because passthroughGemini calls os.Exit on
// success, which would terminate the test binary.  The negative path is
// exercised by TestExtensionPassthroughFailsWhenGeminiMissing below; that
// covers the only branch unique to our wiring (the rest is delegated to the
// system gemini binary).

// When the gemini binary is absent, passthrough subcommands report a clear
// error instead of silently exiting.
func TestExtensionPassthroughFailsWhenGeminiMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty PATH
	_, stderr, code := execute(t, "extension", "install", "some-source")
	if code == 0 {
		t.Fatal("expected non-zero exit when gemini missing")
	}
	if !strings.Contains(stderr, "gemini CLI is required") {
		t.Fatalf("stderr missing helpful message: %s", stderr)
	}
}

// Help output documents every passthrough subcommand.
func TestExtensionHelpListsAllSubcommands(t *testing.T) {
	stdout, _, code := execute(t, "extension", "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"browse", "install", "uninstall", "list", "update", "disable", "enable", "link", "new", "validate", "settings"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help missing %q:\n%s", want, stdout)
		}
	}
}

// `cam ext` and `cam extensions` aliases both work.
func TestExtensionAliases(t *testing.T) {
	for _, alias := range []string{"ext", "extensions"} {
		stdout, _, code := execute(t, alias, "--help")
		if code != 0 {
			t.Fatalf("alias %s exit = %d", alias, code)
		}
		if !strings.Contains(stdout, "browse") {
			t.Fatalf("alias %s help missing browse:\n%s", alias, stdout)
		}
	}
}
