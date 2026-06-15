package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type toolMenuModel struct {
	tools    []string
	cursor   int
	filter   string
	selected string
	quitting bool
}

func newToolMenuModel(tools []string) toolMenuModel {
	return toolMenuModel{tools: append([]string(nil), tools...)}
}

func (m toolMenuModel) Init() tea.Cmd {
	return nil
}

func (m toolMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.visibleTools())-1 {
				m.cursor++
			}
		case "enter":
			visible := m.visibleTools()
			if len(visible) > 0 {
				m.selected = visible[m.cursor]
			}
			return m, tea.Quit
		case "backspace":
			if m.filter != "" {
				m.filter = m.filter[:len(m.filter)-1]
				m.clampCursor()
			}
		default:
			text := msg.String()
			if len(text) == 1 && text != "/" {
				m.filter += text
				m.clampCursor()
			}
		}
	}
	return m, nil
}

func (m toolMenuModel) View() string {
	var b strings.Builder
	b.WriteString("Manage tools\n\n")
	if m.filter != "" {
		fmt.Fprintf(&b, "Filter: %s\n\n", m.filter)
	}
	visible := m.visibleTools()
	if len(visible) == 0 {
		b.WriteString("No tools match your filter.\n")
	} else {
		for i, tool := range visible {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			fmt.Fprintf(&b, "%s %s\n", cursor, tool)
		}
	}
	b.WriteString("\nUse ↑/↓ or j/k to move, type to filter, Enter to select, q to quit.\n")
	return b.String()
}

func (m toolMenuModel) visibleTools() []string {
	if m.filter == "" {
		return m.tools
	}
	visible := make([]string, 0, len(m.tools))
	filter := strings.ToLower(m.filter)
	for _, tool := range m.tools {
		if strings.Contains(strings.ToLower(tool), filter) {
			visible = append(visible, tool)
		}
	}
	return visible
}

func (m *toolMenuModel) clampCursor() {
	visible := m.visibleTools()
	if len(visible) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
}
