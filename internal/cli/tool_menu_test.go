package cli

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestToolMenuFiltersMovesAndSelectsTool(t *testing.T) {
	model := newToolMenuModel([]string{"claude", "codex", "gemini"})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	model = updated.(toolMenuModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(toolMenuModel)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(toolMenuModel)

	if cmd == nil {
		t.Fatal("enter should quit after selecting a tool")
	}
	if model.selected != "codex" {
		t.Fatalf("selected = %q, want codex", model.selected)
	}
}

func TestToolMenuViewShowsManagementInstructions(t *testing.T) {
	view := newToolMenuModel([]string{"claude"}).View()
	for _, want := range []string{"Manage tools", "> claude", "Use ↑/↓ or j/k", "Enter to select"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q\nview:\n%s", want, view)
		}
	}
}
