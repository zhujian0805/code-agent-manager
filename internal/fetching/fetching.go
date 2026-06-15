// Package fetching downloads upstream skill/agent/plugin repositories.
//
// The current implementation is HTTP-only: it pulls a GitHub repository's
// zip archive, extracts it to a temp directory, and returns the path.  This
// is sufficient for skills/agents/plugins which all live as plain files in a
// public GitHub repo subdirectory.
package fetching

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client downloads and caches GitHub archives.
type Client struct {
	HTTPClient *http.Client
	Timeout    time.Duration
	UserAgent  string
}

// New returns a Client with sane defaults.
func New() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		Timeout:    60 * time.Second,
		UserAgent:  "code-agent-manager-go",
	}
}

// DownloadGitHubZip fetches an owner/repo/branch zip and extracts it under
// destRoot.  The returned path is the root of the extracted tree (which
// GitHub generates as "<repo>-<branch>/").
func (c *Client) DownloadGitHubZip(owner, repo, branch, destRoot string) (string, error) {
	if owner == "" || repo == "" {
		return "", errors.New("fetching: owner and repo are required")
	}
	if branch == "" {
		branch = "main"
	}
	url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, branch)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching: get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching: get %s: HTTP %d", url, resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "cam-fetch-*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return "", err
	}
	zr, err := zip.OpenReader(tmp.Name())
	if err != nil {
		return "", fmt.Errorf("fetching: open zip: %w", err)
	}
	defer zr.Close()
	var topDir string
	for _, f := range zr.File {
		if topDir == "" {
			parts := strings.SplitN(f.Name, "/", 2)
			topDir = parts[0]
		}
		path := filepath.Join(destRoot, f.Name)
		// Guard against zip-slip.
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(destRoot)+string(os.PathSeparator)) {
			return "", fmt.Errorf("fetching: zip entry escapes dest: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		mode := f.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		dst, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return "", err
		}
		src, err := f.Open()
		if err != nil {
			dst.Close()
			return "", err
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			src.Close()
			return "", err
		}
		dst.Close()
		src.Close()
	}
	return filepath.Join(destRoot, topDir), nil
}

// FetchFile fetches a single URL into dest.  Used for raw repo content
// (e.g. https://raw.githubusercontent.com/.../README.md).
func (c *Client) FetchFile(url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching: get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching: get %s: HTTP %d", url, resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}
