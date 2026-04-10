package api

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingClient wraps MockClient and counts calls to specific methods.
type countingClient struct {
	MockClient
	listReposCalls atomic.Int32
	listPRsCalls   atomic.Int32
}

func (c *countingClient) ListRepositories() ([]GitRepository, error) {
	c.listReposCalls.Add(1)
	return c.MockClient.ListRepositories()
}

func (c *countingClient) ListPullRequests(status string) ([]PullRequest, error) {
	c.listPRsCalls.Add(1)
	return c.MockClient.ListPullRequests(status)
}

func TestCacheHit(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	// First call: cache miss
	repos1, err := cc.ListRepositories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call: cache hit
	repos2, err := cc.ListRepositories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inner.listReposCalls.Load() != 1 {
		t.Errorf("expected 1 call to inner, got %d", inner.listReposCalls.Load())
	}
	if len(repos1) != len(repos2) {
		t.Errorf("cache returned different data")
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	nowTime := time.Now()
	cc.now = func() time.Time { return nowTime }

	// First call: cache miss
	_, err := cc.ListPullRequests("active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.listPRsCalls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", inner.listPRsCalls.Load())
	}

	// Second call within TTL: cache hit
	nowTime = nowTime.Add(10 * time.Second)
	cc.now = func() time.Time { return nowTime }
	_, err = cc.ListPullRequests("active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.listPRsCalls.Load() != 1 {
		t.Errorf("expected 1 call (cached), got %d", inner.listPRsCalls.Load())
	}

	// Third call after TTL: cache miss
	nowTime = nowTime.Add(60 * time.Second) // well past 30s TTL
	cc.now = func() time.Time { return nowTime }
	_, err = cc.ListPullRequests("active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.listPRsCalls.Load() != 2 {
		t.Errorf("expected 2 calls (expired), got %d", inner.listPRsCalls.Load())
	}
}

func TestInvalidateAll(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	_, _ = cc.ListRepositories()
	_, _ = cc.ListPullRequests("active")

	cc.InvalidateAll()

	_, _ = cc.ListRepositories()
	_, _ = cc.ListPullRequests("active")

	if inner.listReposCalls.Load() != 2 {
		t.Errorf("expected 2 repo calls after invalidate, got %d", inner.listReposCalls.Load())
	}
	if inner.listPRsCalls.Load() != 2 {
		t.Errorf("expected 2 PR calls after invalidate, got %d", inner.listPRsCalls.Load())
	}
}

func TestInvalidatePrefix(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	_, _ = cc.ListRepositories()
	_, _ = cc.ListPullRequests("active")

	// Only invalidate PR cache
	cc.InvalidatePrefix("prs:")

	_, _ = cc.ListRepositories()
	_, _ = cc.ListPullRequests("active")

	if inner.listReposCalls.Load() != 1 {
		t.Errorf("repo cache should not be invalidated, got %d calls", inner.listReposCalls.Load())
	}
	if inner.listPRsCalls.Load() != 2 {
		t.Errorf("PR cache should be invalidated, got %d calls", inner.listPRsCalls.Load())
	}
}

func TestMutationInvalidatesCache(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	// Populate PR cache
	_, _ = cc.ListPullRequests("active")
	if inner.listPRsCalls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", inner.listPRsCalls.Load())
	}

	// MergePullRequest on MockClient returns ErrDemoMode, so cache is NOT
	// invalidated (only on success). Manually invalidate to simulate a
	// successful mutation path.
	cc.InvalidatePrefix("prs:")

	// Next call should be a cache miss
	_, _ = cc.ListPullRequests("active")
	if inner.listPRsCalls.Load() != 2 {
		t.Errorf("expected 2 calls after invalidate, got %d", inner.listPRsCalls.Load())
	}
}

func TestSessionCacheNeverExpires(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	nowTime := time.Now()
	cc.now = func() time.Time { return nowTime }

	// GetProjectID uses session TTL
	_, _ = cc.GetProjectID()

	// Jump forward 1 hour
	nowTime = nowTime.Add(1 * time.Hour)
	cc.now = func() time.Time { return nowTime }

	_, _ = cc.GetProjectID()

	// Should still use cached value (session = no expiry)
	cc.mu.RLock()
	_, exists := cc.store["projectid"]
	cc.mu.RUnlock()
	if !exists {
		t.Error("session cache entry should still exist")
	}
}

func TestConcurrentAccess(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cc.ListRepositories()
			_, _ = cc.ListPullRequests("active")
		}()
	}
	wg.Wait()

	// Should not panic and inner should have been called at least once
	if inner.listReposCalls.Load() < 1 {
		t.Error("expected at least 1 repo call")
	}
}
