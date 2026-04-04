package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// GetFileContentAtCommit fetches the raw content of a file at a specific commit.
// Uses $format=text to retrieve plain bytes, which works for files of any size.
func (c *Client) GetFileContentAtCommit(repoID, filePath, commitID string) (string, error) {
	v := url.Values{}
	v.Set("path", filePath)
	v.Set("versionType", "commit")
	v.Set("version", commitID)
	v.Set("$format", "text")

	path := fmt.Sprintf("/git/repositories/%s/items?%s", repoID, v.Encode())
	return c.getContent(path)
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
		diff = fmt.Sprintf("--- a%s\n+++ b%s\n(no textual changes)\n", oldPath, newPath)
	}

	// For rename changes, prepend a git-style rename header so the viewer can
	// tell this was a rename rather than a delete+add of unrelated files.
	if changeType == "rename" && oldPath != newPath {
		diff = fmt.Sprintf("rename from %s\nrename to %s\n", oldPath, newPath) + diff
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
