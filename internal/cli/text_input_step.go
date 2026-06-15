package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// textInputStep is a reusable free-text input component that follows the
// same update/view/selected contract as pickerStep. It handles typing,
// backspace, Enter (advance), Esc (back), and Ctrl+C (abort).
type textInputStep struct {
	title       string
	prompt      string
	value       string
	placeholder string // shown when value is empty (e.g. current value for update)
	required    bool
	hint        string
	errMsg      string // validation error, cleared on next keystroke
}

// newTextInputStep constructs a textInputStep. placeholder is the default
// value shown/returned when value is empty (used for update mode pre-population).
func newTextInputStep(title, prompt, placeholder string, required bool, hint string) textInputStep {
	return textInputStep{
		title:       title,
		prompt:      prompt,
		placeholder: placeholder,
		required:    required,
		hint:        hint,
	}
}

// update processes a key event and returns the action the wrapping model
// should take. Same contract as pickerStep.update.
func (s *textInputStep) update(msg tea.KeyMsg) pickerAction {
	switch msg.Type {
	case tea.KeyCtrlC:
		return pickerActionAbort
	case tea.KeyEsc:
		return pickerActionBack
	case tea.KeyEnter:
		if s.value == "" && s.required && s.placeholder == "" {
			s.errMsg = "This field is required"
			return pickerActionNone
		}
		return pickerActionAdvance
	case tea.KeyBackspace:
		if s.value != "" {
			s.value = s.value[:len(s.value)-1]
		}
		return pickerActionNone
	default:
		text := msg.String()
		if len(text) == 1 {
			s.errMsg = ""
			s.value += text
		}
		return pickerActionNone
	}
}

// selected returns the current value, falling back to placeholder when the
// user typed nothing and the field is optional.
func (s textInputStep) selected() string {
	if s.value != "" {
		return s.value
	}
	return s.placeholder
}

// view renders the step for display.
func (s textInputStep) view() string {
	var b strings.Builder
	b.WriteString(s.title)
	b.WriteString("\n\n")
	if s.errMsg != "" {
		fmt.Fprintf(&b, "  ⚠ %s\n\n", s.errMsg)
	}
	if s.value != "" {
		fmt.Fprintf(&b, "  %s %s_\n", s.prompt, s.value)
	} else if s.placeholder != "" {
		fmt.Fprintf(&b, "  %s [%s] _\n", s.prompt, s.placeholder)
	} else {
		fmt.Fprintf(&b, "  %s _\n", s.prompt)
	}
	if s.hint != "" {
		b.WriteString("\n")
		b.WriteString(s.hint)
		b.WriteString("\n")
	}
	return b.String()
}
