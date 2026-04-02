package api

import (
	"fmt"
	"strings"
)

// ListPullRequests returns pull requests for the project.
// status can be "active", "completed", "abandoned", or "all".
func (c *Client) ListPullRequests(status string) ([]PullRequest, error) {
	path := fmt.Sprintf("/git/pullrequests?searchCriteria.status=%s", status)

	var resp ListResponse[PullRequest]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("list pull requests: %w", err)
	}

	return resp.Value, nil
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

// CreateThread creates a new comment thread on a pull request.
func (c *Client) CreateThread(repoID string, prID int, content string) error {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/threads", repoID, prID)

	body := CreateThreadRequest{
		Comments: []CreateCommentRequest{
			{Content: content, CommentType: "text"},
		},
		Status: "active",
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
