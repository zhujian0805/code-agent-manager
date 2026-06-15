// Package ui provides output helpers for the CLI.
//
// The Printer type satisfies the doctor.Reporter interface so the diagnostic
// runner can be wired up without referencing any color library directly.
package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// Printer wraps Stdout/Stderr writers with optional ANSI coloring.  Color is
// auto-detected (TTY + no NO_COLOR + non-dumb TERM) but may be overridden
// directly for tests.
type Printer struct {
	Out   io.Writer
	Err   io.Writer
	Color bool
}

// New constructs a Printer whose Color flag follows the standard auto-detect
// rules.  nil writers fall back to os.Stdout / os.Stderr.
func New(out, err io.Writer) *Printer {
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	return &Printer{Out: out, Err: err, Color: detectColor(out)}
}

func detectColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if term := os.Getenv("TERM"); term == "dumb" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}

func (p *Printer) sprint(c *color.Color, msg string) string {
	if !p.Color {
		return msg
	}
	return c.Sprint(msg)
}

// Header writes a bold section heading to stdout.
func (p *Printer) Header(msg string) {
	bold := color.New(color.Bold)
	fmt.Fprintln(p.Out, p.sprint(bold, msg))
}

// Info writes an informational message (no prefix) to stdout.
func (p *Printer) Info(msg string) {
	fmt.Fprintln(p.Out, msg)
}

// Pass writes a green checkmark line to stdout.
func (p *Printer) Pass(msg string) {
	check := color.New(color.FgGreen).Sprint("✓")
	if !p.Color {
		check = "✓"
	}
	fmt.Fprintf(p.Out, "%s %s\n", check, msg)
}

// Warn writes a yellow warning line to stdout, optionally followed by a hint.
func (p *Printer) Warn(msg, hint string) {
	mark := color.New(color.FgYellow).Sprint("⚠")
	if !p.Color {
		mark = "⚠"
	}
	fmt.Fprintf(p.Out, "%s %s\n", mark, msg)
	if hint != "" {
		fmt.Fprintf(p.Out, "  Suggestion: %s\n", hint)
	}
}

// Fail writes a red failure line to stdout, optionally followed by a hint.
func (p *Printer) Fail(msg, hint string) {
	mark := color.New(color.FgRed).Sprint("✗")
	if !p.Color {
		mark = "✗"
	}
	fmt.Fprintf(p.Out, "%s %s\n", mark, msg)
	if hint != "" {
		fmt.Fprintf(p.Out, "  Suggestion: %s\n", hint)
	}
}
