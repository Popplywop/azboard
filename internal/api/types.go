package api

import "time"

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
	const prefix = "refs/heads/"
	if len(ref) > len(prefix) {
		return ref[len(prefix):]
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
	FilePath string `json:"filePath"`
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

// CreateThreadRequest is the body for creating a new comment thread.
type CreateThreadRequest struct {
	Comments []CreateCommentRequest `json:"comments"`
	Status   string                 `json:"status"`
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
