package api

import "testing"

func TestStripRefsPrefixValid(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/foo", "feature/foo"},
		{"refs/heads/a", "a"},
	}
	for _, tt := range tests {
		got := stripRefsPrefix(tt.input)
		if got != tt.want {
			t.Errorf("stripRefsPrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripRefsPrefixNoPrefix(t *testing.T) {
	tests := []string{"main", "feature/foo", "something-long-string"}
	for _, input := range tests {
		got := stripRefsPrefix(input)
		if got != input {
			t.Errorf("stripRefsPrefix(%q) = %q, want passthrough", input, got)
		}
	}
}

func TestStripRefsPrefixEmpty(t *testing.T) {
	got := stripRefsPrefix("")
	if got != "" {
		t.Errorf("stripRefsPrefix(\"\") = %q, want \"\"", got)
	}
}

func TestStripRefsPrefixExactPrefix(t *testing.T) {
	got := stripRefsPrefix("refs/heads/")
	if got != "" {
		t.Errorf("stripRefsPrefix(\"refs/heads/\") = %q, want \"\"", got)
	}
}

func TestPullRequestSourceBranch(t *testing.T) {
	pr := PullRequest{SourceRefName: "refs/heads/feature-x"}
	if got := pr.SourceBranch(); got != "feature-x" {
		t.Errorf("SourceBranch() = %q, want %q", got, "feature-x")
	}
}

func TestPullRequestTargetBranch(t *testing.T) {
	pr := PullRequest{TargetRefName: "refs/heads/main"}
	if got := pr.TargetBranch(); got != "main" {
		t.Errorf("TargetBranch() = %q, want %q", got, "main")
	}
}

func TestReviewerVoteString(t *testing.T) {
	tests := []struct {
		vote int
		want string
	}{
		{10, "Approved"},
		{5, "Approved with suggestions"},
		{0, "No vote"},
		{-5, "Waiting for author"},
		{-10, "Rejected"},
		{99, "Unknown"},
	}
	for _, tt := range tests {
		r := Reviewer{Vote: tt.vote}
		got := r.VoteString()
		if got != tt.want {
			t.Errorf("VoteString() for vote=%d: got %q, want %q", tt.vote, got, tt.want)
		}
	}
}

func TestReviewerVoteIcon(t *testing.T) {
	tests := []struct {
		vote int
		want string
	}{
		{10, "\u2713"},
		{5, "~"},
		{0, "\u00b7"},
		{-5, "\u23f3"},
		{-10, "\u2717"},
		{99, "?"},
	}
	for _, tt := range tests {
		r := Reviewer{Vote: tt.vote}
		got := r.VoteIcon()
		if got != tt.want {
			t.Errorf("VoteIcon() for vote=%d: got %q, want %q", tt.vote, got, tt.want)
		}
	}
}

func TestGitBranchShortName(t *testing.T) {
	b := GitBranch{Name: "refs/heads/feature-branch"}
	if got := b.ShortName(); got != "feature-branch" {
		t.Errorf("ShortName() = %q, want %q", got, "feature-branch")
	}
}
