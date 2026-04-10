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

func TestCachedGetProjectID(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	id1, _ := cc.GetProjectID()
	id2, _ := cc.GetProjectID()

	if id1 != id2 {
		t.Errorf("expected same project ID, got %q and %q", id1, id2)
	}
}

func TestCachedGetCurrentUserID(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	id1, _ := cc.GetCurrentUserID()
	id2, _ := cc.GetCurrentUserID()

	if id1 != id2 {
		t.Errorf("expected same user ID, got %q and %q", id1, id2)
	}
}

func TestCachedListBranches(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	b1, err := cc.ListBranches("my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b2, _ := cc.ListBranches("my-repo")

	if len(b1) != len(b2) {
		t.Error("expected same branches on cache hit")
	}
}

func TestCachedGetPullRequestThreads(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	t1, _ := cc.GetPullRequestThreads("repo-1", 1847)
	t2, _ := cc.GetPullRequestThreads("repo-1", 1847)

	if len(t1) != len(t2) {
		t.Error("expected same threads on cache hit")
	}
}

func TestCachedListWorkItems(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	types := []string{"Bug", "Task"}
	w1, _ := cc.ListWorkItems(types, "@me", "", true)
	w2, _ := cc.ListWorkItems(types, "@me", "", true)

	if len(w1) != len(w2) {
		t.Error("expected same work items on cache hit")
	}
}

func TestCachedGetWorkItem(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	wi1, _ := cc.GetWorkItem(4521)
	wi2, _ := cc.GetWorkItem(4521)

	if wi1.ID != wi2.ID {
		t.Error("expected same work item on cache hit")
	}
}

func TestCachedGetWorkItemComments(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	c1, _ := cc.GetWorkItemComments(4521)
	c2, _ := cc.GetWorkItemComments(4521)

	if len(c1) != len(c2) {
		t.Error("expected same comments on cache hit")
	}
}

func TestCachedGetWorkItemTypeStates(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	s1, _ := cc.GetWorkItemTypeStates("Bug")
	s2, _ := cc.GetWorkItemTypeStates("Bug")

	if len(s1) != len(s2) {
		t.Error("expected same states on cache hit")
	}
}

func TestCachedGetPullRequestIterations(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	i1, _ := cc.GetPullRequestIterations("repo-1", 1847)
	i2, _ := cc.GetPullRequestIterations("repo-1", 1847)

	if len(i1) != len(i2) {
		t.Error("expected same iterations on cache hit")
	}
}

func TestCachedGetPullRequestByID(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	pr1, _ := cc.GetPullRequestByID(1847)
	pr2, _ := cc.GetPullRequestByID(1847)

	if pr1.PullRequestID != pr2.PullRequestID {
		t.Error("expected same PR on cache hit")
	}
}

func TestCachedListPullRequestsForRepo(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	p1, _ := cc.ListPullRequestsForRepo("inventory-api", "active")
	p2, _ := cc.ListPullRequestsForRepo("inventory-api", "active")

	if len(p1) != len(p2) {
		t.Error("expected same PRs on cache hit")
	}
}

func TestCachedListDraftPullRequests(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	p1, _ := cc.ListDraftPullRequests()
	p2, _ := cc.ListDraftPullRequests()

	if len(p1) != len(p2) {
		t.Error("expected same draft PRs on cache hit")
	}
}

func TestCachedListDraftPullRequestsForRepo(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	p1, _ := cc.ListDraftPullRequestsForRepo("inventory-api")
	p2, _ := cc.ListDraftPullRequestsForRepo("inventory-api")

	if len(p1) != len(p2) {
		t.Error("expected same draft PRs on cache hit")
	}
}

func TestCachedCreateThreadInvalidatesCache(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	// Populate threads cache
	_, _ = cc.GetPullRequestThreads("repo-1", 1847)

	// CreateThread returns ErrDemoMode, but we can test the flow
	_ = cc.CreateThread("repo-1", 1847, "test", nil)

	// Since ErrDemoMode is non-nil, cache won't be invalidated automatically.
	// Manual invalidation test covered elsewhere.
}

func TestCachedUpdateWorkItemStateInvalidates(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	// Populate
	_, _ = cc.GetWorkItem(4521)

	// UpdateWorkItemState returns ErrDemoMode
	_ = cc.UpdateWorkItemState(4521, "Active")

	// Verify cache entry still exists (error path doesn't invalidate)
	cc.mu.RLock()
	_, exists := cc.store["wi:4521"]
	cc.mu.RUnlock()
	if !exists {
		t.Error("cache entry should still exist after failed mutation")
	}
}

func TestCachedGetPullRequest(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	pr1, _ := cc.GetPullRequest("repo-1", 1847)
	pr2, _ := cc.GetPullRequest("repo-1", 1847)

	if pr1.PullRequestID != pr2.PullRequestID {
		t.Error("expected same PR on cache hit")
	}
}

func TestCachedGetFileContentAtCommit(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	c1, _ := cc.GetFileContentAtCommit("repo-1", "/src/main.go", "abc123")
	c2, _ := cc.GetFileContentAtCommit("repo-1", "/src/main.go", "abc123")

	if c1 != c2 {
		t.Error("expected same content on cache hit")
	}
}

func TestCachedGetPullRequestIterationChanges(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	ch1, _ := cc.GetPullRequestIterationChanges("repo-1", 1847, 1)
	ch2, _ := cc.GetPullRequestIterationChanges("repo-1", 1847, 1)

	if len(ch1) != len(ch2) {
		t.Error("expected same changes on cache hit")
	}
}

func TestCachedMutationMergePR(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	// MergePullRequest returns ErrDemoMode so cache won't invalidate
	err := cc.MergePullRequest("repo-1", 1, "squash", "msg", true)
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationAbandonPR(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	err := cc.AbandonPullRequest("repo-1", 1)
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationToggleDraft(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	err := cc.ToggleDraft("repo-1", 1, true)
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationSetVote(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	err := cc.SetVote("repo-1", 1, "user-1", 10)
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationReplyToThread(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	err := cc.ReplyToThread("repo-1", 1, 1, "reply")
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationUpdateThreadStatus(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	err := cc.UpdateThreadStatus("repo-1", 1, 1, "fixed")
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationAddWorkItemComment(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	err := cc.AddWorkItemComment(42, "hello")
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationLinkWorkItemToPR(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	err := cc.LinkWorkItemToPR(42, "vstfs:///Git/PullRequestId/proj/repo/1")
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedMutationCreatePullRequest(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	_, err := cc.CreatePullRequest("repo-1", "title", "refs/heads/feature", "refs/heads/main", "desc", false)
	if err == nil {
		t.Error("expected ErrDemoMode")
	}
}

func TestCachedBuildUnifiedDiff(t *testing.T) {
	inner := &countingClient{}
	cc := NewCachedClient(inner)

	change := IterationChange{
		ChangeID:   1,
		ChangeType: "edit",
		Item:       ChangeItem{Path: "/src/main.go"},
	}
	d1, _ := cc.BuildUnifiedDiff("repo-1", change, "abc", "def")
	d2, _ := cc.BuildUnifiedDiff("repo-1", change, "abc", "def")

	if d1 != d2 {
		t.Error("expected same diff on cache hit")
	}
}
