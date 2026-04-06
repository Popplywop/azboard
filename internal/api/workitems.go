package api

import (
	"fmt"
	"net/url"
	"strings"
)

// ListWorkItems queries work items via WIQL and fetches their details.
// types is a list of work item types to include.
// assignedTo is "@me" for current-user scoped queries; empty = no filter.
// areaPath restricts results to items under the given area path; empty = no filter.
// activeOnly filters to non-terminal states.
func (c *Client) ListWorkItems(types []string, assignedTo, areaPath string, activeOnly bool) ([]WorkItem, error) {
	// Build WIQL WHERE clauses
	var clauses []string
	clauses = append(clauses, fmt.Sprintf("[System.TeamProject] = '%s'", c.project))

	if len(types) > 0 {
		quotedTypes := make([]string, len(types))
		for i, t := range types {
			quotedTypes[i] = "'" + t + "'"
		}
		clauses = append(clauses, fmt.Sprintf("[System.WorkItemType] IN (%s)", strings.Join(quotedTypes, ", ")))
	}

	if assignedTo != "" {
		clauses = append(clauses, fmt.Sprintf("[System.AssignedTo] = %s", assignedTo))
	}

	if areaPath != "" {
		clauses = append(clauses, fmt.Sprintf("[System.AreaPath] UNDER '%s'", areaPath))
	}

	if activeOnly {
		clauses = append(clauses, "[System.State] NOT IN ('Closed', 'Done', 'Resolved', 'Removed')")
	}

	query := fmt.Sprintf(
		"SELECT [System.Id] FROM WorkItems WHERE %s ORDER BY [System.ChangedDate] DESC",
		strings.Join(clauses, " AND "),
	)

	// Use $top=200 to prevent hitting ADO's 20,000-item hard limit
	wiqlPath := "/wit/wiql?$top=200"
	var result WIQLResult
	if err := c.post(wiqlPath, WIQLRequest{Query: query}, &result); err != nil {
		return nil, fmt.Errorf("wiql query: %w", err)
	}

	if len(result.WorkItems) == 0 {
		return nil, nil
	}

	// Build comma-separated IDs (max 200 per request per ADO limits)
	ids := make([]string, 0, len(result.WorkItems))
	for _, ref := range result.WorkItems {
		ids = append(ids, fmt.Sprintf("%d", ref.ID))
		if len(ids) == 200 {
			break
		}
	}

	path := fmt.Sprintf("/wit/workitems?ids=%s&$expand=fields", strings.Join(ids, ","))
	var resp ListResponse[WorkItem]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("get work items: %w", err)
	}

	return resp.Value, nil
}

// GetWorkItem returns a single work item with all fields expanded.
func (c *Client) GetWorkItem(id int) (WorkItem, error) {
	path := fmt.Sprintf("/wit/workitems/%d?$expand=all", id)
	var wi WorkItem
	if err := c.get(path, &wi); err != nil {
		return WorkItem{}, fmt.Errorf("get work item %d: %w", id, err)
	}
	return wi, nil
}

// GetWorkItemComments returns comments for a work item.
func (c *Client) GetWorkItemComments(id int) ([]WorkItemComment, error) {
	path := fmt.Sprintf("/wit/workitems/%d/comments", id)
	var resp WorkItemCommentsResult
	if err := c.getPreview(path, "7.1-preview.4", &resp); err != nil {
		return nil, fmt.Errorf("get work item comments %d: %w", id, err)
	}
	return resp.Comments, nil
}

// UpdateWorkItemState updates a work item's state.
func (c *Client) UpdateWorkItemState(id int, state string) error {
	path := fmt.Sprintf("/wit/workitems/%d", id)
	body := []WorkItemPatchOp{
		{Op: "add", Path: "/fields/System.State", Value: state},
	}
	return c.patchJSONPatch(path, body, nil)
}

// AddWorkItemComment adds a comment to a work item.
func (c *Client) AddWorkItemComment(id int, text string) error {
	path := fmt.Sprintf("/wit/workitems/%d/comments", id)
	body := AddWorkItemCommentRequest{Text: text}
	return c.postPreview(path, "7.1-preview.4", body, nil)
}

// GetWorkItemTypeStates fetches the valid states for a work item type from ADO.
// Falls back to the hardcoded WorkItemStates map if the API call fails.
func (c *Client) GetWorkItemTypeStates(workItemType string) ([]string, error) {
	path := fmt.Sprintf("/wit/workitemtypes/%s/states", url.PathEscape(workItemType))
	var resp ListResponse[WorkItemTypeState]
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("get work item type states: %w", err)
	}
	names := make([]string, len(resp.Value))
	for i, s := range resp.Value {
		names[i] = s.Name
	}
	return names, nil
}

// LinkWorkItemToPR links a work item to a pull request via an artifact link.
// prArtifactURL format: vstfs:///Git/PullRequestId/{projectID}/{repoID}/{prID}
func (c *Client) LinkWorkItemToPR(workItemID int, prArtifactURL string) error {
	path := fmt.Sprintf("/wit/workitems/%d", workItemID)
	body := []WorkItemPatchOp{
		{
			Op:   "add",
			Path: "/relations/-",
			Value: WorkItemLinkValue{
				Rel: "ArtifactLink",
				URL: prArtifactURL,
				Attributes: map[string]interface{}{
					"name": "Pull Request",
				},
			},
		},
	}
	return c.patchJSONPatch(path, body, nil)
}
