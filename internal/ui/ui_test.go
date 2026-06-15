package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chat2anyllm/code-agent-manager/internal/ui"
)

func TestPrinterPlainOutputWhenColorDisabled(t *testing.T) {
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	p := ui.New(out, err)
	p.Color = false

	p.Header("Section")
	p.Info("informational")
	p.Pass("works")
	p.Warn("warning", "fix it")
	p.Fail("broken", "try again")

	got := out.String()
	for _, want := range []string{
		"Section\n",
		"informational\n",
		"✓ works\n",
		"⚠ warning\n",
		"  Suggestion: fix it\n",
		"✗ broken\n",
		"  Suggestion: try again\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plain output missing %q\noutput:\n%s", want, got)
		}
	}
	for _, banned := range []string{"\x1b[", "\033["} {
		if strings.Contains(got, banned) {
			t.Fatalf("non-color writer received ANSI escape %q\noutput:\n%s", banned, got)
		}
	}
}

func TestPrinterRespectsNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	p := ui.New(&bytes.Buffer{}, &bytes.Buffer{})
	if p.Color {
		t.Fatal("NO_COLOR should disable color")
	}
}

func TestPrinterDefaultsToFalseColorForBuffer(t *testing.T) {
	p := ui.New(&bytes.Buffer{}, &bytes.Buffer{})
	if p.Color {
		t.Fatal("bytes.Buffer is not a TTY; Color should be false")
	}
}

func TestPrinterWarnAndFailOmitHintWhenEmpty(t *testing.T) {
	out := &bytes.Buffer{}
	p := ui.New(out, &bytes.Buffer{})
	p.Color = false
	p.Warn("only warning", "")
	p.Fail("only failure", "")
	got := out.String()
	if strings.Contains(got, "Suggestion") {
		t.Fatalf("empty hint should not print Suggestion line:\n%s", got)
	}
}
