package cli

import (
	"strings"
	"testing"
)

func TestTextInputStep_TypeAndAdvance(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", false, "hint")
	for _, ch := range "hello" {
		action := s.update(keyEvent(string(ch)))
		if action != pickerActionNone {
			t.Fatalf("typing should return None, got %v", action)
		}
	}
	if s.value != "hello" {
		t.Fatalf("value = %q, want hello", s.value)
	}
	action := s.update(keyEvent("enter"))
	if action != pickerActionAdvance {
		t.Fatalf("enter should advance, got %v", action)
	}
	if s.selected() != "hello" {
		t.Fatalf("selected() = %q, want hello", s.selected())
	}
}

func TestTextInputStep_EscReturnsBack(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", false, "hint")
	action := s.update(keyEvent("esc"))
	if action != pickerActionBack {
		t.Fatalf("esc should return Back, got %v", action)
	}
}

func TestTextInputStep_CtrlCAborts(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", false, "hint")
	action := s.update(keyEvent("ctrl+c"))
	if action != pickerActionAbort {
		t.Fatalf("ctrl+c should abort, got %v", action)
	}
}

func TestTextInputStep_RequiredRejectsEmpty(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", true, "hint")
	action := s.update(keyEvent("enter"))
	if action != pickerActionNone {
		t.Fatalf("enter on empty required should return None, got %v", action)
	}
	if s.errMsg == "" {
		t.Fatal("expected errMsg set on empty required field")
	}
}

func TestTextInputStep_RequiredClearsErrOnKeystroke(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", true, "hint")
	s.update(keyEvent("enter")) // triggers error
	if s.errMsg == "" {
		t.Fatal("expected errMsg after empty enter")
	}
	s.update(keyEvent("a"))
	if s.errMsg != "" {
		t.Fatalf("errMsg should be cleared after keystroke, got %q", s.errMsg)
	}
}

func TestTextInputStep_Backspace(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", false, "hint")
	s.update(keyEvent("a"))
	s.update(keyEvent("b"))
	s.update(keyEvent("backspace"))
	if s.value != "a" {
		t.Fatalf("value = %q after backspace, want a", s.value)
	}
}

func TestTextInputStep_BackspaceOnEmpty(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", false, "hint")
	action := s.update(keyEvent("backspace"))
	if action != pickerActionNone {
		t.Fatalf("backspace on empty should be no-op, got %v", action)
	}
	if s.value != "" {
		t.Fatalf("value should remain empty, got %q", s.value)
	}
}

func TestTextInputStep_PlaceholderOnOptionalEmptyEnter(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "default-val", false, "hint")
	action := s.update(keyEvent("enter"))
	if action != pickerActionAdvance {
		t.Fatalf("enter on optional with placeholder should advance, got %v", action)
	}
	if s.selected() != "default-val" {
		t.Fatalf("selected() = %q, want default-val", s.selected())
	}
}

func TestTextInputStep_TypingOverridesPlaceholder(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "default-val", false, "hint")
	s.update(keyEvent("x"))
	s.update(keyEvent("enter"))
	if s.selected() != "x" {
		t.Fatalf("selected() = %q, want x", s.selected())
	}
}

func TestTextInputStep_ViewContainsTitleAndHint(t *testing.T) {
	s := newTextInputStep("My Title", "Prompt:", "", false, "my hint")
	view := s.view()
	if !strings.Contains(view, "My Title") {
		t.Fatalf("view missing title: %s", view)
	}
	if !strings.Contains(view, "my hint") {
		t.Fatalf("view missing hint: %s", view)
	}
}

func TestTextInputStep_ViewShowsPlaceholder(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "current-val", false, "hint")
	view := s.view()
	if !strings.Contains(view, "[current-val]") {
		t.Fatalf("view missing placeholder: %s", view)
	}
}

func TestTextInputStep_ViewShowsError(t *testing.T) {
	s := newTextInputStep("Title", "Name:", "", true, "hint")
	s.update(keyEvent("enter")) // triggers error
	view := s.view()
	if !strings.Contains(view, s.errMsg) {
		t.Fatalf("view missing error: %s", view)
	}
}
