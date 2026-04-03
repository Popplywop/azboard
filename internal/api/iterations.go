package api

import "fmt"

// GetPullRequestIterations returns PR iterations.
func (c *Client) GetPullRequestIterations(repoID string, prID int) ([]Iteration, error) {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/iterations", repoID, prID)

	var resp ListResponse[Iteration]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("get PR iterations %d: %w", prID, err)
	}

	return resp.Value, nil
}

type iterationChangesResponse struct {
	ChangeEntries []struct {
		ChangeTrackingID int            `json:"changeTrackingId"`
		ChangeType       string         `json:"changeType"`
		Item             ChangeItem     `json:"item"`
		OriginalPath     string         `json:"originalPath"`
		AdditionalProps  map[string]any `json:"additionalProperties"`
	} `json:"changeEntries"`
}

// GetPullRequestIterationChanges returns file changes for a PR iteration.
func (c *Client) GetPullRequestIterationChanges(repoID string, prID, iterationID int) ([]IterationChange, error) {
	path := fmt.Sprintf("/git/repositories/%s/pullrequests/%d/iterations/%d/changes", repoID, prID, iterationID)

	var resp iterationChangesResponse
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("get PR iteration changes %d/%d: %w", prID, iterationID, err)
	}

	changes := make([]IterationChange, 0, len(resp.ChangeEntries))
	for _, e := range resp.ChangeEntries {
		changes = append(changes, IterationChange{
			ChangeID:     e.ChangeTrackingID,
			ChangeType:   e.ChangeType,
			Item:         e.Item,
			OriginalPath: e.OriginalPath,
		})
	}

	return changes, nil
}
