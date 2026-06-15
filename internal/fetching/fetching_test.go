package fetching_test

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/fetching"
)

func TestDownloadGitHubZipExtractsAllEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/archive/refs/heads/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		zw := zip.NewWriter(w)
		entry := func(name, contents string) {
			f, err := zw.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = f.Write([]byte(contents))
		}
		entry("repo-main/", "")
		entry("repo-main/README.md", "hello")
		entry("repo-main/skills/foo/SKILL.md", "skill")
		_ = zw.Close()
	}))
	defer srv.Close()

	client := fetching.New()
	// Override the URL by hijacking the HTTPClient with a transport.
	client.HTTPClient.Transport = &rewriteTransport{base: srv.URL}

	dest := t.TempDir()
	root, err := client.DownloadGitHubZip("owner", "repo", "main", dest)
	if err != nil {
		t.Fatalf("DownloadGitHubZip err = %v", err)
	}
	if filepath.Base(root) != "repo-main" {
		t.Fatalf("root = %q", root)
	}
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("README content = %q", data)
	}
	if _, err := os.Stat(filepath.Join(root, "skills/foo/SKILL.md")); err != nil {
		t.Fatalf("nested file missing: %v", err)
	}
}

func TestDownloadGitHubZipRejectsZipSlip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		zw := zip.NewWriter(w)
		f, _ := zw.Create("../escape.txt")
		_, _ = f.Write([]byte("nope"))
		_ = zw.Close()
	}))
	defer srv.Close()
	client := fetching.New()
	client.HTTPClient.Transport = &rewriteTransport{base: srv.URL}
	if _, err := client.DownloadGitHubZip("owner", "repo", "main", t.TempDir()); err == nil {
		t.Fatal("expected zip-slip rejection")
	}
}

func TestFetchFileWritesContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("body"))
	}))
	defer srv.Close()
	c := fetching.New()
	dest := filepath.Join(t.TempDir(), "out.txt")
	if err := c.FetchFile(srv.URL+"/file", dest); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "body" {
		t.Fatalf("body = %q", data)
	}
}

// rewriteTransport rewrites any github.com URL to the test server.
type rewriteTransport struct{ base string }

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := rt.base + req.URL.Path
	newReq, err := http.NewRequest(req.Method, url, bytes.NewReader(nil))
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}
