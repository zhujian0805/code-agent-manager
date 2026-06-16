package desktop

import "testing"

func TestConfigServiceListAndValidate(t *testing.T) {
	service := NewConfigService()
	files := service.ListFiles()
	if len(files) == 0 {
		t.Fatal("expected config files")
	}
	if _, err := service.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}
