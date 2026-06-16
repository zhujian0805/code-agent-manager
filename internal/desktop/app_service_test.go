package desktop

import "testing"

func TestAppServiceVersionAndPlatform(t *testing.T) {
	service := NewAppService("1.2.3")
	if service.Version() != "1.2.3" {
		t.Fatalf("unexpected version: %s", service.Version())
	}
	if service.Platform() == "" {
		t.Fatal("expected platform")
	}
	if !service.Quit().OK {
		t.Fatal("expected quit result ok")
	}
}
