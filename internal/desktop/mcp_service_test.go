package desktop

import "testing"

func TestMCPServiceClientsAndRegistry(t *testing.T) {
	service := NewMCPService()
	clients := service.ListClients()
	if len(clients) == 0 {
		t.Fatal("expected supported clients")
	}

	matches, err := service.SearchRegistry("github")
	if err != nil {
		t.Fatalf("search registry: %v", err)
	}
	if matches == nil {
		t.Fatal("expected non-nil registry result slice")
	}
}
