// Package doctor runs CAM's diagnostic checks.
//
// Each Check is independently testable: the Reporter interface is the only
// seam between the check's logic and the user-facing renderer.  Tests can
// supply a fake Reporter to assert on the exact sequence of pass/warn/fail
// calls without needing to capture stdout.
package doctor

import (
	"context"
	"sort"
)

// Reporter receives the structured output of a Check.  Implementations
// typically format messages onto a *ui.Printer.
type Reporter interface {
	Header(string)
	Info(string)
	Pass(string)
	Warn(msg, hint string)
	Fail(msg, hint string)
}

// Result summarises a single Check's outcome.
type Result struct {
	Issues int
}

// Check is a single diagnostic step.  Implementations should report findings
// through r; the returned Result is used by the Run loop to compute the
// summary issue count.
type Check interface {
	Name() string
	Run(ctx context.Context, r Reporter) Result
}

// Run executes the supplied checks in order and returns the total issue
// count.  Each check is announced with its Name as a header so output reads
// like the Python doctor command.
func Run(ctx context.Context, r Reporter, checks []Check) int {
	total := 0
	for _, check := range checks {
		r.Header(check.Name())
		res := check.Run(ctx, r)
		total += res.Issues
		r.Info("")
	}
	return total
}

// SortedNames returns the names of a slice of checks in alphabetical order.
// Exposed primarily for tests that assert on the check inventory.
func SortedNames(checks []Check) []string {
	names := make([]string, 0, len(checks))
	for _, c := range checks {
		names = append(names, c.Name())
	}
	sort.Strings(names)
	return names
}
