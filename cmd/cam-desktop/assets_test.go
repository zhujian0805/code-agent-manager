package main

import (
	"io/fs"
	"testing"
)

func TestDesktopAssetsIncludesIndex(t *testing.T) {
	assets, err := desktopAssets()
	if err != nil {
		t.Fatalf("desktopAssets() error = %v", err)
	}

	content, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		t.Fatalf("read embedded index.html: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("embedded index.html is empty")
	}
}
