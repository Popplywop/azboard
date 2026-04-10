package api

import (
	"fmt"
	"strings"
)

const prPageSize = 50

// ListPullRequests returns pull requests for the project.
// status can be "active", "completed", "abandoned", or "all".
func (c *Client) ListPullRequests(status string) ([]PullRequest, error) {
	path := fmt.Sprintf("/git/pullrequests?searchCriteria.status=%s&$top=%d", status, prPageSize)

	var resp ListResponse[PullRequest]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("list pull requests: %w", err)
	}

	return resp.Value, nil
}

// ListPullRequestsForRepo returns pull requests for a specific repository.
func (c *Client) ListPullRequestsForRepo(repoName, status string) ([]PullRequest, error) {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests?searchCriteria.status=%s&$top=%d", repoName, status, prPageSize)

	var resp ListResponse[PullRequest]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("list pull requests for repo %s: %w", repoName, err)
	}

	return resp.Value, nil
}

// ListDraftPullRequests returns only draft (active) pull requests, using the
// native ADO isDraft filter so the server filters rather than the client.
func (c *Client) ListDraftPullRequests() ([]PullRequest, error) {
	path := "/git/pullrequests?searchCriteria.status=active&searchCriteria.isDraft=true&$top=" + fmt.Sprint(prPageSize)

	var resp ListResponse[PullRequest]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("list draft pull requests: %w", err)
	}

	return resp.Value, nil
}

// ListDraftPullRequestsForRepo returns only draft PRs for a specific repo.
func (c *Client) ListDraftPullRequestsForRepo(repoName string) ([]PullRequest, error) {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests?searchCriteria.status=active&searchCriteria.isDraft=true&$top=%d", repoName, prPageSize)

	var resp ListResponse[PullRequest]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("list draft pull requests for repo %s: %w", repoName, err)
	}

	return resp.Value, nil
}

// GetPullRequestByID returns a single pull request by project-wide PR ID,
// without needing to know the repository up front.
//
// The ADO searchCriteria.pullRequestId filter is ignored by some org
// configurations (returns all PRs regardless), so we fetch all PRs with
// status=all and client-side filter by PullRequestID.
func (c *Client) GetPullRequestByID(prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/git/pullrequests?searchCriteria.status=all&searchCriteria.pullRequestId=%d", prID)

	var resp ListResponse[PullRequest]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("get pull request %d: %w", prID, err)
	}
	for i := range resp.Value {
		if resp.Value[i].PullRequestID == prID {
			return &resp.Value[i], nil
		}
	}
	return nil, fmt.Errorf("pull request %d not found", prID)
}

// GetPullRequest returns a single pull request by repository and PR ID.
func (c *Client) GetPullRequest(repoID string, prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d", repoID, prID)

	var pr PullRequest
	if err := c.get(path, &pr); err != nil {
		return nil, fmt.Errorf("get pull request %d: %w", prID, err)
	}

	return &pr, nil
}

// GetPullRequestThreads returns comment threads for a pull request.
func (c *Client) GetPullRequestThreads(repoID string, prID int) ([]Thread, error) {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/threads", repoID, prID)

	var resp ListResponse[Thread]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("get PR threads %d: %w", prID, err)
	}

	return resp.Value, nil
}

// CreatePullRequest creates a new pull request in the given repository.
func (c *Client) CreatePullRequest(repoID, title, sourceBranch, targetBranch, description string, isDraft bool) (PullRequest, error) {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests", repoID)

	body := CreatePullRequestRequest{
		Title:         title,
		Description:   description,
		SourceRefName: sourceBranch,
		TargetRefName: targetBranch,
		IsDraft:       isDraft,
	}

	var pr PullRequest
	if err := c.post(path, body, &pr); err != nil {
		return PullRequest{}, fmt.Errorf("create pull request: %w", err)
	}
	return pr, nil
}

// MergePullRequest completes (merges) a pull request.
// strategy must be one of: "squash", "noFastForward", "rebase", "rebaseMerge".
func (c *Client) MergePullRequest(repoID string, prID int, strategy, commitMsg string, deleteSourceBranch bool) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d", repoID, prID)

	body := CompletePullRequestRequest{
		Status: "completed",
		CompletionOptions: CompletionOptions{
			MergeStrategy:      strategy,
			DeleteSourceBranch: deleteSourceBranch,
			MergeCommitMessage: commitMsg,
		},
	}

	return c.patch(path, body, nil)
}

// AbandonPullRequest abandons a pull request.
func (c *Client) AbandonPullRequest(repoID string, prID int) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d", repoID, prID)
	body := StatusUpdateRequest{Status: "abandoned"}
	return c.patch(path, body, nil)
}

// ToggleDraft sets a pull request's draft state.
func (c *Client) ToggleDraft(repoID string, prID int, isDraft bool) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d", repoID, prID)
	body := StatusUpdateRequest{IsDraft: &isDraft}
	return c.patch(path, body, nil)
}

// CreateThread creates a new comment thread on a pull request.
// Pass a non-nil threadCtx to anchor the comment to a specific file/line.
func (c *Client) CreateThread(repoID string, prID int, content string, threadCtx *ThreadContext) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/threads", repoID, prID)

	body := CreateThreadRequest{
		Comments: []CreateCommentRequest{
			{Content: content, CommentType: "text"},
		},
		Status:        "active",
		ThreadContext: threadCtx,
	}

	return c.post(path, body, nil)
}

// ReplyToThread adds a comment to an existing thread.
func (c *Client) ReplyToThread(repoID string, prID, threadID int, content string) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/threads/%d/comments", repoID, prID, threadID)

	body := CreateCommentRequest{
		Content:     content,
		CommentType: "text",
	}

	return c.post(path, body, nil)
}

// UpdateThreadStatus updates a thread's status (e.g. "active", "fixed", "closed").
func (c *Client) UpdateThreadStatus(repoID string, prID, threadID int, status string) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/threads/%d", repoID, prID, threadID)

	body := UpdateThreadRequest{Status: status}

	return c.patch(path, body, nil)
}

// SetVote sets the current user's vote on a pull request.
func (c *Client) SetVote(repoID string, prID int, reviewerID string, vote int) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/reviewers/%s", repoID, prID, reviewerID)

	body := SetVoteRequest{Vote: vote}

	return c.put(path, body, nil)
}

// GetCurrentUserID returns the authenticated user's ID.
// Tries connectionData first, falls back to the profile endpoint.
func (c *Client) GetCurrentUserID() (string, error) {
	// Try connectionData (org-level endpoint)
	var data ConnectionData
	if err := c.getOrg("/connectionData", &data); err == nil && data.AuthenticatedUser.ID != "" {
		return data.AuthenticatedUser.ID, nil
	}

	// Fallback: try the VSSPS profile endpoint
	// https://vssps.dev.azure.com/{org}/_apis/profile/profiles/me
	var profile struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	}
	vsspsURL := strings.Replace(c.orgURL, "dev.azure.com", "vssps.dev.azure.com", 1)
	// Also handle visualstudio.com — vssps uses app.vssps.visualstudio.com
	vsspsURL = strings.Replace(vsspsURL, ".visualstudio.com", ".vssps.visualstudio.com", 1)
	if err := c.doRequest("GET", vsspsURL+"/profile/profiles/me", nil, &profile); err == nil && profile.ID != "" {
		return profile.ID, nil
	}

	return "", fmt.Errorf("could not determine user ID — try setting AZURE_DEVOPS_ORG_URL in your config")
}
