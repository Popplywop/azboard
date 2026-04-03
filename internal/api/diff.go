package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type gitItemResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// GetFileContentAtCommit fetches file content for a specific commit.
func (c *Client) GetFileContentAtCommit(repoID, filePath, commitID string) (string, error) {
	v := url.Values{}
	v.Set("path", filePath)
	v.Set("versionType", "commit")
	v.Set("version", commitID)
	v.Set("includeContent", "true")

	path := fmt.Sprintf("/git/repositories/%s/items?%s", repoID, v.Encode())

	var item gitItemResponse
	if err := c.get(path, &item); err != nil {
		return "", err
	}

	return item.Content, nil
}

// BuildUnifiedDiff generates a local unified diff for a changed file.
func (c *Client) BuildUnifiedDiff(repoID string, change IterationChange, oldCommitID, newCommitID string) (string, error) {
	oldPath := change.Item.Path
	if change.OriginalPath != "" {
		oldPath = change.OriginalPath
	}
	newPath := change.Item.Path

	var oldContent string
	var newContent string
	var err error

	changeType := strings.ToLower(change.ChangeType)
	if changeType != "add" {
		oldContent, err = c.GetFileContentAtCommit(repoID, oldPath, oldCommitID)
		if err != nil {
			return "", fmt.Errorf("load old file content: %w", err)
		}
		oldContent = normalizeLineEndings(oldContent)
	}

	if changeType != "delete" {
		newContent, err = c.GetFileContentAtCommit(repoID, newPath, newCommitID)
		if err != nil {
			return "", fmt.Errorf("load new file content: %w", err)
		}
		newContent = normalizeLineEndings(newContent)
	}

	ud := difflib.UnifiedDiff{
		A:        difflib.SplitLines(ensureTrailingNewline(oldContent)),
		B:        difflib.SplitLines(ensureTrailingNewline(newContent)),
		FromFile: "a" + oldPath,
		ToFile:   "b" + newPath,
		Context:  3,
	}

	diff, err := difflib.GetUnifiedDiffString(ud)
	if err != nil {
		return "", fmt.Errorf("generate unified diff: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		return fmt.Sprintf("--- a%s\n+++ b%s\n(no textual changes)\n", oldPath, newPath), nil
	}

	return diff, nil
}

func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func normalizeLineEndings(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
