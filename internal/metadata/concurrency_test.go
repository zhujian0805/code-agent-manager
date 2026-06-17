package metadata

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// concurrentFetcher records the maximum number of in-flight Fetch calls so we
// can assert downloads actually run in parallel, and produces a deterministic
// one-skill repo so results are checkable.
type concurrentFetcher struct {
	inFlight atomic.Int32
	maxSeen  atomic.Int32
	calls    atomic.Int32
}

func (c *concurrentFetcher) Fetch(owner, repo, branch, dest string) (string, error) {
	c.calls.Add(1)
	cur := c.inFlight.Add(1)
	for {
		prev := c.maxSeen.Load()
		if cur <= prev || c.maxSeen.CompareAndSwap(prev, cur) {
			break
		}
	}
	// Hold the slot briefly so concurrent workers reliably overlap.
	time.Sleep(5 * time.Millisecond)
	defer c.inFlight.Add(-1)

	root := filepath.Join(dest, repo+"-"+branch)
	p := filepath.Join(root, "skills", repo+"-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte("---\nname: "+repo+"-skill\ndescription: x\n---\n"), 0o644); err != nil {
		return "", err
	}
	return root, nil
}

func TestRefreshAllRunsConcurrently(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("CAM_CACHE_DIR", filepath.Join(dir, "cache"))
	s := NewStore(filepath.Join(dir, "cam.db"))

	cf := &concurrentFetcher{}
	svc := NewService(s).WithFetcher(cf)

	summary, err := svc.RefreshAll(ctx)
	if err != nil {
		t.Fatalf("RefreshAll: %v", err)
	}
	if summary.ItemsAdded == 0 {
		t.Fatal("expected items added")
	}
	if cf.calls.Load() == 0 {
		t.Fatal("fetcher was never called")
	}
	// With many bundled repos and an 8-worker pool, at least 2 downloads must
	// have overlapped. (If the machine is extremely slow this could be flaky;
	// the barrier in Fetch makes overlap reliable.)
	if cf.maxSeen.Load() < 2 {
		t.Fatalf("expected concurrent downloads, max in-flight was %d", cf.maxSeen.Load())
	}
}
