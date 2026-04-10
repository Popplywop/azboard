package api

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// CacheInvalidator allows the UI layer to clear cached data.
type CacheInvalidator interface {
	InvalidateAll()
	InvalidatePrefix(prefix string)
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// CachedClient wraps a Clienter with in-memory TTL-based caching.
type CachedClient struct {
	inner Clienter
	mu    sync.RWMutex
	store map[string]cacheEntry
	now   func() time.Time // injectable for testing
}

// NewCachedClient wraps a Clienter with caching.
func NewCachedClient(inner Clienter) *CachedClient {
	return &CachedClient{
		inner: inner,
		store: make(map[string]cacheEntry),
		now:   time.Now,
	}
}

// Cache TTL constants.
const (
	ttlSession = 0 // never expires during session
	ttlLong    = 5 * time.Minute
	ttlMedium  = 1 * time.Minute
	ttlShort   = 30 * time.Second
	ttlBrief   = 15 * time.Second
)

func (c *CachedClient) get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.store[key]
	if !ok {
		return nil, false
	}
	if entry.expiresAt.IsZero() {
		return entry.value, true // session-lifetime entry
	}
	if c.now().After(entry.expiresAt) {
		return nil, false // expired
	}
	return entry.value, true
}

func (c *CachedClient) set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = c.now().Add(ttl)
	}
	c.store[key] = cacheEntry{value: value, expiresAt: exp}
}

// InvalidateAll clears the entire cache.
func (c *CachedClient) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string]cacheEntry)
}

// InvalidatePrefix removes all entries whose key starts with prefix.
func (c *CachedClient) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.store {
		if strings.HasPrefix(k, prefix) {
			delete(c.store, k)
		}
	}
}

// --- Repositories & branches ---

func (c *CachedClient) GetProjectID() (string, error) {
	if v, ok := c.get("projectid"); ok {
		return v.(string), nil
	}
	val, err := c.inner.GetProjectID()
	if err != nil {
		return "", err
	}
	c.set("projectid", val, ttlSession)
	return val, nil
}

func (c *CachedClient) ListRepositories() ([]GitRepository, error) {
	if v, ok := c.get("repos"); ok {
		return v.([]GitRepository), nil
	}
	val, err := c.inner.ListRepositories()
	if err != nil {
		return nil, err
	}
	c.set("repos", val, ttlLong)
	return val, nil
}

func (c *CachedClient) ListBranches(repoName string) ([]GitBranch, error) {
	key := "branches:" + repoName
	if v, ok := c.get(key); ok {
		return v.([]GitBranch), nil
	}
	val, err := c.inner.ListBranches(repoName)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlMedium)
	return val, nil
}

// --- Pull requests ---

func (c *CachedClient) ListPullRequests(status string) ([]PullRequest, error) {
	key := "prs:" + status
	if v, ok := c.get(key); ok {
		return v.([]PullRequest), nil
	}
	val, err := c.inner.ListPullRequests(status)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlShort)
	return val, nil
}

func (c *CachedClient) ListPullRequestsForRepo(repoName, status string) ([]PullRequest, error) {
	key := fmt.Sprintf("prs:%s:%s", repoName, status)
	if v, ok := c.get(key); ok {
		return v.([]PullRequest), nil
	}
	val, err := c.inner.ListPullRequestsForRepo(repoName, status)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlShort)
	return val, nil
}

func (c *CachedClient) ListDraftPullRequests() ([]PullRequest, error) {
	key := "prs:draft"
	if v, ok := c.get(key); ok {
		return v.([]PullRequest), nil
	}
	val, err := c.inner.ListDraftPullRequests()
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlShort)
	return val, nil
}

func (c *CachedClient) ListDraftPullRequestsForRepo(repoName string) ([]PullRequest, error) {
	key := "prs:draft:" + repoName
	if v, ok := c.get(key); ok {
		return v.([]PullRequest), nil
	}
	val, err := c.inner.ListDraftPullRequestsForRepo(repoName)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlShort)
	return val, nil
}

func (c *CachedClient) GetPullRequestByID(prID int) (*PullRequest, error) {
	key := fmt.Sprintf("pr:id:%d", prID)
	if v, ok := c.get(key); ok {
		return v.(*PullRequest), nil
	}
	val, err := c.inner.GetPullRequestByID(prID)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlBrief)
	return val, nil
}

func (c *CachedClient) GetPullRequest(repoID string, prID int) (*PullRequest, error) {
	key := fmt.Sprintf("pr:%s:%d", repoID, prID)
	if v, ok := c.get(key); ok {
		return v.(*PullRequest), nil
	}
	val, err := c.inner.GetPullRequest(repoID, prID)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlBrief)
	return val, nil
}

func (c *CachedClient) GetPullRequestThreads(repoID string, prID int) ([]Thread, error) {
	key := fmt.Sprintf("threads:%s:%d", repoID, prID)
	if v, ok := c.get(key); ok {
		return v.([]Thread), nil
	}
	val, err := c.inner.GetPullRequestThreads(repoID, prID)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlBrief)
	return val, nil
}

func (c *CachedClient) GetCurrentUserID() (string, error) {
	if v, ok := c.get("userid"); ok {
		return v.(string), nil
	}
	val, err := c.inner.GetCurrentUserID()
	if err != nil {
		return "", err
	}
	c.set("userid", val, ttlSession)
	return val, nil
}

// --- Mutations: delegate then invalidate ---

func (c *CachedClient) CreatePullRequest(repoID, title, sourceBranch, targetBranch, description string, isDraft bool) (PullRequest, error) {
	val, err := c.inner.CreatePullRequest(repoID, title, sourceBranch, targetBranch, description, isDraft)
	if err == nil {
		c.InvalidatePrefix("prs:")
	}
	return val, err
}

func (c *CachedClient) MergePullRequest(repoID string, prID int, strategy, commitMsg string, deleteSourceBranch bool) error {
	err := c.inner.MergePullRequest(repoID, prID, strategy, commitMsg, deleteSourceBranch)
	if err == nil {
		c.InvalidatePrefix("prs:")
		c.InvalidatePrefix("pr:")
	}
	return err
}

func (c *CachedClient) AbandonPullRequest(repoID string, prID int) error {
	err := c.inner.AbandonPullRequest(repoID, prID)
	if err == nil {
		c.InvalidatePrefix("prs:")
		c.InvalidatePrefix("pr:")
	}
	return err
}

func (c *CachedClient) ToggleDraft(repoID string, prID int, isDraft bool) error {
	err := c.inner.ToggleDraft(repoID, prID, isDraft)
	if err == nil {
		c.InvalidatePrefix("prs:")
		c.InvalidatePrefix("pr:")
	}
	return err
}

func (c *CachedClient) CreateThread(repoID string, prID int, content string, threadCtx *ThreadContext) error {
	err := c.inner.CreateThread(repoID, prID, content, threadCtx)
	if err == nil {
		c.InvalidatePrefix(fmt.Sprintf("threads:%s:%d", repoID, prID))
	}
	return err
}

func (c *CachedClient) ReplyToThread(repoID string, prID, threadID int, content string) error {
	err := c.inner.ReplyToThread(repoID, prID, threadID, content)
	if err == nil {
		c.InvalidatePrefix(fmt.Sprintf("threads:%s:%d", repoID, prID))
	}
	return err
}

func (c *CachedClient) UpdateThreadStatus(repoID string, prID, threadID int, status string) error {
	err := c.inner.UpdateThreadStatus(repoID, prID, threadID, status)
	if err == nil {
		c.InvalidatePrefix(fmt.Sprintf("threads:%s:%d", repoID, prID))
	}
	return err
}

func (c *CachedClient) SetVote(repoID string, prID int, reviewerID string, vote int) error {
	err := c.inner.SetVote(repoID, prID, reviewerID, vote)
	if err == nil {
		c.InvalidatePrefix(fmt.Sprintf("pr:%s:%d", repoID, prID))
		c.InvalidatePrefix("prs:")
	}
	return err
}

// --- Iterations & diffs ---

func (c *CachedClient) GetPullRequestIterations(repoID string, prID int) ([]Iteration, error) {
	key := fmt.Sprintf("iterations:%s:%d", repoID, prID)
	if v, ok := c.get(key); ok {
		return v.([]Iteration), nil
	}
	val, err := c.inner.GetPullRequestIterations(repoID, prID)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlShort)
	return val, nil
}

func (c *CachedClient) GetPullRequestIterationChanges(repoID string, prID, iterationID int) ([]IterationChange, error) {
	key := fmt.Sprintf("iterchanges:%s:%d:%d", repoID, prID, iterationID)
	if v, ok := c.get(key); ok {
		return v.([]IterationChange), nil
	}
	val, err := c.inner.GetPullRequestIterationChanges(repoID, prID, iterationID)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlSession) // immutable per iteration
	return val, nil
}

func (c *CachedClient) GetFileContentAtCommit(repoID, filePath, commitID string) (string, error) {
	key := fmt.Sprintf("file:%s:%s:%s", repoID, filePath, commitID)
	if v, ok := c.get(key); ok {
		return v.(string), nil
	}
	val, err := c.inner.GetFileContentAtCommit(repoID, filePath, commitID)
	if err != nil {
		return "", err
	}
	c.set(key, val, ttlSession) // immutable per commit
	return val, nil
}

func (c *CachedClient) BuildUnifiedDiff(repoID string, change IterationChange, oldCommitID, newCommitID string) (string, error) {
	key := fmt.Sprintf("diff:%s:%s:%s:%s", repoID, change.Item.Path, oldCommitID, newCommitID)
	if v, ok := c.get(key); ok {
		return v.(string), nil
	}
	val, err := c.inner.BuildUnifiedDiff(repoID, change, oldCommitID, newCommitID)
	if err != nil {
		return "", err
	}
	c.set(key, val, ttlSession) // immutable per commit pair
	return val, nil
}

// --- Work items ---

func (c *CachedClient) ListWorkItems(types []string, assignedTo, areaPath string, activeOnly bool) ([]WorkItem, error) {
	key := fmt.Sprintf("wi:%s:%s:%s:%v", strings.Join(types, ","), assignedTo, areaPath, activeOnly)
	if v, ok := c.get(key); ok {
		return v.([]WorkItem), nil
	}
	val, err := c.inner.ListWorkItems(types, assignedTo, areaPath, activeOnly)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlShort)
	return val, nil
}

func (c *CachedClient) GetWorkItem(id int) (WorkItem, error) {
	key := fmt.Sprintf("wi:%d", id)
	if v, ok := c.get(key); ok {
		return v.(WorkItem), nil
	}
	val, err := c.inner.GetWorkItem(id)
	if err != nil {
		return WorkItem{}, err
	}
	c.set(key, val, ttlBrief)
	return val, nil
}

func (c *CachedClient) GetWorkItemComments(id int) ([]WorkItemComment, error) {
	key := fmt.Sprintf("wicomments:%d", id)
	if v, ok := c.get(key); ok {
		return v.([]WorkItemComment), nil
	}
	val, err := c.inner.GetWorkItemComments(id)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlBrief)
	return val, nil
}

func (c *CachedClient) GetWorkItemTypeStates(workItemType string) ([]string, error) {
	key := "wistates:" + workItemType
	if v, ok := c.get(key); ok {
		return v.([]string), nil
	}
	val, err := c.inner.GetWorkItemTypeStates(workItemType)
	if err != nil {
		return nil, err
	}
	c.set(key, val, ttlSession)
	return val, nil
}

// --- Work item mutations ---

func (c *CachedClient) UpdateWorkItemState(id int, state string) error {
	err := c.inner.UpdateWorkItemState(id, state)
	if err == nil {
		c.InvalidatePrefix("wi:")
	}
	return err
}

func (c *CachedClient) AddWorkItemComment(id int, text string) error {
	err := c.inner.AddWorkItemComment(id, text)
	if err == nil {
		c.InvalidatePrefix(fmt.Sprintf("wicomments:%d", id))
		c.InvalidatePrefix(fmt.Sprintf("wi:%d", id))
	}
	return err
}

func (c *CachedClient) LinkWorkItemToPR(workItemID int, prArtifactURL string) error {
	err := c.inner.LinkWorkItemToPR(workItemID, prArtifactURL)
	if err == nil {
		c.InvalidatePrefix(fmt.Sprintf("wi:%d", workItemID))
	}
	return err
}
