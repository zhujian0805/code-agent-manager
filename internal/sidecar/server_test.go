package sidecar

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestSidecarRequiresToken(t *testing.T) {
	server := New(Options{Version: "test", Token: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/api/app/version", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSidecarVersion(t *testing.T) {
	server := New(Options{Version: "test-version", Token: "secret"})
	req := httptest.NewRequest(http.MethodGet, "/api/app/version", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	payload := map[string]string{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if payload["version"] != "test-version" {
		t.Fatalf("version = %q", payload["version"])
	}
}

func TestSidecarProviderLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.json")
	server := New(Options{Version: "test", ProvidersPath: path, Token: "secret"})
	handler := server.Handler()

	body := bytes.NewBufferString(`{"name":"local","endpoint":"http://localhost:4000/v1","supportedClient":"claude","models":["m1"],"enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/providers", body)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("add status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var listed []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("list json: %v", err)
	}
	if len(listed) != 1 || listed[0]["name"] != "local" {
		t.Fatalf("listed = %+v", listed)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/providers/local/disable", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/providers/local", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}
