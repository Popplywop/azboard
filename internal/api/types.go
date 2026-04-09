package api

import (
	"strings"
	"time"
)

// ListResponse is the generic wrapper for Azure DevOps list API responses.
type ListResponse[T any] struct {
	Count int `json:"count"`
	Value []T `json:"value"`
}

// PullRequest represents an Azure DevOps pull request.
type PullRequest struct {
	PullRequestID int           `json:"pullRequestId"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Status        string        `json:"status"`
	CreationDate  time.Time     `json:"creationDate"`
	ClosedDate    *time.Time    `json:"closedDate,omitempty"`
	SourceRefName string        `json:"sourceRefName"`
	TargetRefName string        `json:"targetRefName"`
	MergeStatus   string        `json:"mergeStatus"`
	IsDraft       bool          `json:"isDraft"`
	CreatedBy     IdentityRef   `json:"createdBy"`
	Repository    GitRepository `json:"repository"`
	Reviewers     []Reviewer    `json:"reviewers"`
}

// SourceBranch returns the short branch name (strips refs/heads/).
func (pr *PullRequest) SourceBranch() string {
	return stripRefsPrefix(pr.SourceRefName)
}

// TargetBranch returns the short branch name (strips refs/heads/).
func (pr *PullRequest) TargetBranch() string {
	return stripRefsPrefix(pr.TargetRefName)
}

func stripRefsPrefix(ref string) string {
	after, found := strings.CutPrefix(ref, "refs/heads/")
	if found {
		return after
	}
	return ref
}

// IdentityRef represents a user identity in Azure DevOps.
type IdentityRef struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	ID          string `json:"id"`
}

// GitRepository represents a git repository in Azure DevOps.
type GitRepository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Reviewer represents a pull request reviewer.
type Reviewer struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	ID          string `json:"id"`
	Vote        int    `json:"vote"`
	IsRequired  bool   `json:"isRequired"`
}

// VoteString returns a human-readable vote status.
func (r *Reviewer) VoteString() string {
	switch r.Vote {
	case 10:
		return "Approved"
	case 5:
		return "Approved with suggestions"
	case 0:
		return "No vote"
	case -5:
		return "Waiting for author"
	case -10:
		return "Rejected"
	default:
		return "Unknown"
	}
}

// VoteIcon returns a compact icon for the vote status.
func (r *Reviewer) VoteIcon() string {
	switch r.Vote {
	case 10:
		return "✓"
	case 5:
		return "~"
	case 0:
		return "·"
	case -5:
		return "⏳"
	case -10:
		return "✗"
	default:
		return "?"
	}
}

// Thread represents a pull request comment thread.
type Thread struct {
	ID              int            `json:"id"`
	Status          string         `json:"status"`
	Comments        []Comment      `json:"comments"`
	IsDeleted       bool           `json:"isDeleted"`
	ThreadContext   *ThreadContext `json:"threadContext,omitempty"`
	PublishedDate   time.Time      `json:"publishedDate"`
	LastUpdatedDate time.Time      `json:"lastUpdatedDate"`
}

// ThreadContext contains file path info for inline comments.
type ThreadContext struct {
	FilePath       string     `json:"filePath"`
	RightFileStart *LineRange `json:"rightFileStart,omitempty"`
	RightFileEnd   *LineRange `json:"rightFileEnd,omitempty"`
}

// LineRange identifies a line/offset position in a file.
type LineRange struct {
	Line   int `json:"line"`
	Offset int `json:"offset"`
}

// Comment represents a single comment within a thread.
type Comment struct {
	ID            int         `json:"id"`
	Author        IdentityRef `json:"author"`
	Content       string      `json:"content"`
	PublishedDate time.Time   `json:"publishedDate"`
	CommentType   string      `json:"commentType"`
	IsDeleted     bool        `json:"isDeleted"`
}

// --- Request body types ---

// CreatePullRequestRequest is the body for creating a new pull request.
type CreatePullRequestRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	SourceRefName string `json:"sourceRefName"` // "refs/heads/branch-name"
	TargetRefName string `json:"targetRefName"`
	IsDraft       bool   `json:"isDraft"`
}

// CompletePullRequestRequest is the body for completing (merging) a PR.
type CompletePullRequestRequest struct {
	Status            string            `json:"status"`
	CompletionOptions CompletionOptions `json:"completionOptions"`
}

// CompletionOptions specifies how a PR should be completed.
type CompletionOptions struct {
	MergeStrategy      string `json:"mergeStrategy"`
	DeleteSourceBranch bool   `json:"deleteSourceBranch"`
	MergeCommitMessage string `json:"mergeCommitMessage,omitempty"`
}

// StatusUpdateRequest is the body for updating a PR's status (abandon, etc.).
type StatusUpdateRequest struct {
	Status  string `json:"status"`
	IsDraft *bool  `json:"isDraft,omitempty"`
}

// CreateThreadRequest is the body for creating a new comment thread.
type CreateThreadRequest struct {
	Comments      []CreateCommentRequest `json:"comments"`
	Status        string                 `json:"status"`
	ThreadContext *ThreadContext         `json:"threadContext,omitempty"`
}

// CreateCommentRequest is the body for creating a comment (in a new thread or reply).
type CreateCommentRequest struct {
	Content     string `json:"content"`
	CommentType string `json:"commentType"`
}

// UpdateThreadRequest is the body for updating a thread's status.
type UpdateThreadRequest struct {
	Status string `json:"status"`
}

// SetVoteRequest is the body for setting a reviewer's vote on a PR.
type SetVoteRequest struct {
	Vote int `json:"vote"`
}

// ConnectionData represents the response from the connectionData endpoint.
type ConnectionData struct {
	AuthenticatedUser struct {
		ID          string `json:"id"`
		DisplayName string `json:"providerDisplayName"`
	} `json:"authenticatedUser"`
}

// GitBranch represents a git branch (ref) in Azure DevOps.
type GitBranch struct {
	Name string `json:"name"` // full ref name e.g. "refs/heads/main"
}

// ShortName strips the "refs/heads/" prefix.
func (b GitBranch) ShortName() string {
	return stripRefsPrefix(b.Name)
}

// CommitRef identifies a git commit in Azure DevOps.
type CommitRef struct {
	CommitID string `json:"commitId"`
}

// Iteration represents a PR iteration.
type Iteration struct {
	ID              int       `json:"id"`
	Description     string    `json:"description"`
	CreatedDate     time.Time `json:"createdDate"`
	SourceRefCommit CommitRef `json:"sourceRefCommit"`
	TargetRefCommit CommitRef `json:"targetRefCommit"`
}

// ChangeItem identifies a changed path.
type ChangeItem struct {
	Path string `json:"path"`
}

// IterationChange represents a changed file in an iteration.
type IterationChange struct {
	ChangeID     int        `json:"changeId"`
	ChangeType   string     `json:"changeType"`
	Item         ChangeItem `json:"item"`
	OriginalPath string     `json:"originalPath,omitempty"`
}

// --- Work item types ---

// WorkItem represents an Azure DevOps work item.
type WorkItem struct {
	ID     int            `json:"id"`
	Fields WorkItemFields `json:"fields"`
	URL    string         `json:"url"`
}

// WorkItemFields holds the system fields of a work item.
type WorkItemFields struct {
	Title        string      `json:"System.Title"`
	State        string      `json:"System.State"`
	WorkItemType string      `json:"System.WorkItemType"`
	AssignedTo   IdentityRef `json:"System.AssignedTo"`
	Description  string      `json:"System.Description"`
	CreatedDate  time.Time   `json:"System.CreatedDate"`
	ChangedDate  time.Time   `json:"System.ChangedDate"`
	AreaPath     string      `json:"System.AreaPath"`
	TeamProject  string      `json:"System.TeamProject"`
}

// WIQLRequest is the body for a WIQL query.
type WIQLRequest struct {
	Query string `json:"query"`
}

// WIQLResult is the response from a WIQL query.
type WIQLResult struct {
	WorkItems []WIQLRef `json:"workItems"`
}

// WIQLRef is a reference to a work item returned from WIQL.
type WIQLRef struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

// WorkItemPatchOp is a single operation in a JSON Patch document for work items.
type WorkItemPatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

// WorkItemComment is a comment on a work item.
type WorkItemComment struct {
	ID          int         `json:"id"`
	Text        string      `json:"text"`
	CreatedBy   IdentityRef `json:"createdBy"`
	CreatedDate time.Time   `json:"createdDate"`
}

// WorkItemCommentsResult is the response for listing work item comments.
type WorkItemCommentsResult struct {
	Count    int               `json:"count"`
	Comments []WorkItemComment `json:"comments"`
}

// AddWorkItemCommentRequest is the body for adding a comment to a work item.
type AddWorkItemCommentRequest struct {
	Text string `json:"text"`
}

// WorkItemLinkValue is the value for linking a work item relation.
type WorkItemLinkValue struct {
	Rel        string                 `json:"rel"`
	URL        string                 `json:"url"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// WorkItemTypeState is a single state returned by the work item type states API.
type WorkItemTypeState struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Category string `json:"category"`
}

// WorkItemStates is a fallback map used only if the API call to fetch states fails.
var WorkItemStates = map[string][]string{
	"Bug":        {"New", "Active", "Resolved", "Closed"},
	"User Story": {"New", "Active", "Resolved", "Closed"},
	"Task":       {"New", "Active", "Closed"},
	"Feature":    {"New", "Active", "Resolved", "Closed"},
	"Epic":       {"New", "Active", "Resolved", "Closed"},
}
