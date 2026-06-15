package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `cam doctor` always runs to completion (exit 0) and prints both the legacy
// "Providers: N" summary block and the new per-section structured checks.
// We seed a providers.json and an env var so the summary picks them up.
func TestDoctorPrintsProviderSummaryAndAllSections(t *testing.T) {
	home := isolatedHome(t)
	providersFile := filepath.Join(t.TempDir(), "providers.json")
	payload := map[string]any{
		"endpoints": map[string]any{
			"test-endpoint": map[string]any{
				"endpoint":         "https://example.com/v1",
				"api_key_env":      "CAM_DOCTOR_KEY",
				"list_of_models":   []string{"model-a"},
				"supported_client": "claude,codex",
			},
		},
	}
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(providersFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CAM_DOCTOR_KEY", "secret")
	_ = home // ensure HOME is honored when doctor walks ~/.env etc

	stdout, stderr, code := execute(t, "--providers", providersFile, "doctor")
	if code != 0 {
		t.Fatalf("doctor exit = %d; stderr=%s", code, stderr)
	}
	for _, want := range []string{
		"Providers: 1",
		"test-endpoint",
		"Environment: CAM_DOCTOR_KEY set",
		"Installation Check",
		"Configuration Check",
		"Environment File Check",
		"Endpoint Format Check",
		"Cache Check",
		"Gemini / Vertex Authentication Check",
		"GitHub Copilot Authentication Check",
		"Tool Installation Check",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("doctor missing %q\nstdout:\n%s", want, stdout)
		}
	}
}

// --verbose enables the supported-clients line beneath each provider in the
// legacy block.
func TestDoctorVerboseShowsSupportedClients(t *testing.T) {
	isolatedHome(t)
	providersFile := filepath.Join(t.TempDir(), "providers.json")
	payload := `{"endpoints":{"test":{"endpoint":"https://x","supported_client":"claude,codex"}}}`
	if err := os.WriteFile(providersFile, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, _, code := execute(t, "--providers", providersFile, "doctor", "--verbose")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Supported clients: claude, codex") {
		t.Fatalf("verbose output missing supported clients line:\n%s", stdout)
	}
}

// When providers.json is missing entirely, doctor must still run all checks
// — it just notes the failure under the Configuration Check section.
func TestDoctorWithMissingProvidersStillRunsChecks(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "doctor")
	if code != 0 {
		t.Fatalf("exit = %d (doctor should always succeed without InstallationCheck failure)", code)
	}
	if !strings.Contains(stdout, "Installation Check") {
		t.Fatalf("missing installation check section:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Configuration file not found") &&
		!strings.Contains(stdout, "Providers config could not be loaded") {
		t.Fatalf("missing provider failure note:\n%s", stdout)
	}
}

// `cam d` alias works.
func TestDoctorAliasWorks(t *testing.T) {
	isolatedHome(t)
	stdout, _, code := execute(t, "d")
	if code != 0 {
		t.Fatalf("d exit = %d", code)
	}
	if !strings.Contains(stdout, "Installation Check") {
		t.Fatalf("d alias missing checks:\n%s", stdout)
	}
}
