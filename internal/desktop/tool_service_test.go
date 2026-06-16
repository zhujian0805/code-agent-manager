package desktop

import "testing"

func TestToolServiceList(t *testing.T) {
	tools, err := NewToolService().List()
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("expected bundled tools")
	}
}
