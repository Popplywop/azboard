package api

// Clienter is the interface for Azure DevOps API operations.
// Both the real Client and MockClient implement this.
type Clienter interface {
	// Repositories & branches
	GetProjectID() (string, error)
	ListRepositories() ([]GitRepository, error)
	ListBranches(repoName string) ([]GitBranch, error)

	// Pull requests
	ListPullRequests(status string) ([]PullRequest, error)
	ListPullRequestsForRepo(repoName, status string) ([]PullRequest, error)
	ListDraftPullRequests() ([]PullRequest, error)
	ListDraftPullRequestsForRepo(repoName string) ([]PullRequest, error)
	GetPullRequestByID(prID int) (*PullRequest, error)
	GetPullRequest(repoID string, prID int) (*PullRequest, error)
	GetPullRequestThreads(repoID string, prID int) ([]Thread, error)
	CreatePullRequest(repoID, title, sourceBranch, targetBranch, description string, isDraft bool) (PullRequest, error)
	MergePullRequest(repoID string, prID int, strategy, commitMsg string, deleteSourceBranch bool) error
	AbandonPullRequest(repoID string, prID int) error
	ToggleDraft(repoID string, prID int, isDraft bool) error
	CreateThread(repoID string, prID int, content string, threadCtx *ThreadContext) error
	ReplyToThread(repoID string, prID, threadID int, content string) error
	UpdateThreadStatus(repoID string, prID, threadID int, status string) error
	SetVote(repoID string, prID int, reviewerID string, vote int) error
	GetCurrentUserID() (string, error)

	// Iterations & diffs
	GetPullRequestIterations(repoID string, prID int) ([]Iteration, error)
	GetPullRequestIterationChanges(repoID string, prID, iterationID int) ([]IterationChange, error)
	GetFileContentAtCommit(repoID, filePath, commitID string) (string, error)
	BuildUnifiedDiff(repoID string, change IterationChange, oldCommitID, newCommitID string) (string, error)

	// Work items
	ListWorkItems(types []string, assignedTo, areaPath string, activeOnly bool) ([]WorkItem, error)
	GetWorkItem(id int) (WorkItem, error)
	GetWorkItemComments(id int) ([]WorkItemComment, error)
	UpdateWorkItemState(id int, state string) error
	AddWorkItemComment(id int, text string) error
	GetWorkItemTypeStates(workItemType string) ([]string, error)
	LinkWorkItemToPR(workItemID int, prArtifactURL string) error
}
