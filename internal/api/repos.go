package api

import "fmt"

// ListRepositories returns all git repositories in the project.
func (c *Client) ListRepositories() ([]GitRepository, error) {
	var resp ListResponse[GitRepository]
	if err := c.get("/git/repositories", &resp); err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	return resp.Value, nil
}

// ListBranches returns all branches for the given repository (up to 1000).
func (c *Client) ListBranches(repoName string) ([]GitBranch, error) {
	path := fmt.Sprintf("/git/repositories/%s/refs?filter=heads&$top=1000&peelTags=false", repoName)
	var resp ListResponse[GitBranch]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("list branches for %s: %w", repoName, err)
	}
	return resp.Value, nil
}
