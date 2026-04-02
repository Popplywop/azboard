# azboard — Roadmap & Implementation Plan

## Overview

azboard is a terminal UI (TUI) for Azure DevOps, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). It provides a fast, keyboard-driven interface for viewing and interacting with pull requests, sprint boards, and work items — all from the terminal.

## Current State (v0.1)

### PR List View
- Table of active PRs with ID, title, repo, author, status, reviewer vote icons
- Real-time client-side filtering (`/`) across title, repo, author, status, PR ID, reviewer names
- Multi-term AND matching (space-separated)
- Filter count badge showing matches

### PR Detail View
- PR metadata: title, branch info, status, author, creation date
- Reviewer list with vote status icons
- PR description with word wrapping
- Comment threads with author, date, content (word-wrapped)
- Thread navigation (`n`/`N`) with visual focus indicator
- **Interactive features:**
  - Reply to thread (`c`)
  - Create new comment thread (`C`)
  - Resolve/reactivate thread (`s`)
  - Vote: approve (`a`), approve with suggestions (`A`), reject (`x`), wait for author (`w`), reset (`0`)
  - Confirmation prompt for votes
  - Flash messages for success/error feedback

### Auth & Config
- PAT auth via config file (`~/.config/azboard/config.env`)
- Azure CLI token auth (fallback)
- Supports both `dev.azure.com` and `*.visualstudio.com` org URLs
- Config priority: flags > env vars > config file

---

## Planned: File Diff View

### Goal
Show changed files and unified diffs within the PR detail view, similar to the ADO web UI's "Files" tab.

### API Endpoints

| Purpose | Endpoint | Method |
|---|---|---|
| PR iterations | `_apis/git/repositories/{repo}/pullrequests/{pr}/iterations` | GET |
| Changed files per iteration | `_apis/git/repositories/{repo}/pullrequests/{pr}/iterations/{id}/changes` | GET |
| File content at commit | `_apis/git/repositories/{repo}/items?path={path}&versionType=commit&version={commitId}` | GET |

### Architecture

```
detail.go (existing)
  ├── Overview mode (current)
  ├── Files mode (new)
  │   ├── File list — table of changed files with change type + line counts
  │   └── Diff view — unified diff for selected file, color-coded
  └── Sub-view navigation: `f` toggles to files, `esc` goes back
```

### New Files

```
internal/api/iterations.go     — iteration + file change API methods
internal/api/diff.go           — file content fetching, local diff generation
internal/ui/prs/files.go       — file list model (table of changed files)
internal/ui/prs/diff.go        — diff viewport model (unified diff rendering)
```

### New Types

```go
type Iteration struct {
    ID          int       `json:"id"`
    Description string    `json:"description"`
    CreatedDate time.Time `json:"createdDate"`
    SourceRefCommit CommitRef `json:"sourceRefCommit"`
    TargetRefCommit CommitRef `json:"targetRefCommit"`
}

type CommitRef struct {
    CommitID string `json:"commitId"`
}

type IterationChange struct {
    ChangeID       int        `json:"changeId"`
    ChangeType     string     `json:"changeType"` // add, edit, delete, rename
    Item           ChangeItem `json:"item"`
    OriginalPath   string     `json:"originalPath,omitempty"`
}

type ChangeItem struct {
    Path string `json:"path"`
}
```

### UX Design

**File list view** (press `f` from detail):
```
── Files (12 changed) ──────────────────
  M  src/api/client.go          +15 -3
  M  src/api/types.go           +42 -0
  A  src/api/pullrequests.go    +98
  D  src/old/legacy.go          -45
  M  src/ui/detail.go           +120 -30

  ↑/↓ navigate · enter view diff · esc back
```

**Diff view** (press `enter` on a file):
```
── src/api/client.go ──────────────────
@@ -35,6 +35,12 @@
 func NewClient(cfg *config.Config) (*Client, error) {
+    orgURL := fmt.Sprintf("https://dev.azure.com/%s/_apis", cfg.Org)
+    if cfg.OrgURL != "" {
+        base := strings.TrimRight(cfg.OrgURL, "/")
+        orgURL = base + "/_apis"
+    }
     c := &Client{
-        orgURL:  fmt.Sprintf("https://dev.azure.com/...
+        orgURL:  orgURL,
     }
```

### Diff Rendering
- Green (`+`) for additions, red (`-`) for deletions, cyan for `@@` hunk headers
- Line numbers in the gutter
- Word-wrapped to viewport width
- Scrollable viewport for large diffs

### Implementation Steps

1. Add iteration/changes API methods
2. Add file content fetching (GET item at specific commit)
3. Generate unified diff locally (Go's `github.com/sergi/go-diff` or similar)
4. Build file list model with change type icons (A/M/D/R) and line count stats
5. Build diff viewport model with color-coded rendering
6. Wire into detail.go as a sub-view mode
7. Add `f` keybinding and navigation

### Estimated Effort
~2 days for a solid MVP.

---

## Future Plans

### Sprint Board View
- Kanban-style columns grouped by work item state
- Cards showing work item title, assignee, type icon
- Drag-free navigation: arrow keys to move between columns/cards

### Work Item View
- Table of work items with filtering
- Detail view with fields, description, history
- Ability to update state, assign, add comments

### Other Ideas
- PR creation from the TUI
- Branch management
- Build/pipeline status
- Notifications
