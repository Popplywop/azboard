# azboard — MVP Implementation Plan

## Goal

A developer should be able to handle their **entire Azure DevOps PR review workflow** and
get **basic work item visibility** without leaving the terminal.

This document is scoped to the MVP only. Sprint board, CI/CD pipeline views, branch
management, and multi-org support are explicitly post-MVP.

---

## Decisions & Constraints

- **Repo selection**: no repos configured on startup → show empty state with instructions.
  Repos are selected via an interactive popup modal (`R` key). Selection is held in memory
  for the session. A keybind (`ctrl+s` in the repo picker) writes the selection back to
  the config file (`AZBOARD_REPOS`). Config can also pre-populate the selection on startup.
- **PR creation**: minimal form — title, source branch (freeform text, user specifies repo
  context), target branch, optional description. No reviewer assignment or policy
  configuration in the TUI at creation time.
- **Merge strategies**: all four ADO strategies supported — squash, merge commit, rebase,
  semi-linear. Default strategy configurable via `AZBOARD_DEFAULT_MERGE_STRATEGY`.
- **Delete source branch on merge**: merge dialog includes a yes/no toggle; defaults to yes.
- **Inline diff comments**: always anchor to the latest PR iteration.
- **Work item write actions**: state transitions, add comment, link to PR.
- **Work item types shown**: configurable via `AZBOARD_WORK_ITEM_TYPES`
  (default: `User Story,Bug,Task,Feature,Epic`).
- **Current user identity**: fetched once at startup, stored in `AppModel`, passed down to
  all sub-models that need it (work items "My Work" scope, voting, etc.).

---

## Architecture Overview

```
AppModel (internal/ui/app.go)
  ├── viewList      → ListModel       (internal/ui/prs/list.go)
  │     └── RepoPicker modal          (internal/ui/repopicker/picker.go)
  │     └── CreatePRModel             (internal/ui/prs/create.go)
  ├── viewDetail    → DetailModel     (internal/ui/prs/detail.go)
  │     ├── FilesModel                (internal/ui/prs/files.go)
  │     └── DiffModel                 (internal/ui/prs/diff.go)
  └── viewWorkItems → WorkItemListModel  (internal/ui/workitems/list.go)
        └── WorkItemDetailModel          (internal/ui/workitems/detail.go)
```

### New message types (additions)

```go
// Repo picker
ReposLoadedMsg      { repos []GitRepository }
RepoPickerDoneMsg   { selected []string }

// PR lifecycle
PRCreatedMsg        { pr PullRequest }
PRMergedMsg         { prID int }
PRAbandonedMsg      { prID int }
PRDraftToggledMsg   { prID int, isDraft bool }

// Work items
WorkItemsLoadedMsg  { items []WorkItem }
WorkItemSelectedMsg { item WorkItem }
WorkItemUpdatedMsg  { item WorkItem }
```

---

## Phase 1 — Repo Selection

### Config (`internal/config/config.go`)

Add fields to `Config` struct:

```go
Repos               []string  // AZBOARD_REPOS, comma-separated repo names
WorkItemTypes       []string  // AZBOARD_WORK_ITEM_TYPES, comma-separated
DefaultMergeStrategy string   // AZBOARD_DEFAULT_MERGE_STRATEGY (default: "squash")
```

Parsing rules:
- `AZBOARD_REPOS` — split on `,`, trim whitespace, empty string → empty slice (not an error)
- `AZBOARD_WORK_ITEM_TYPES` — same; default value `User Story,Bug,Task,Feature,Epic`
- `AZBOARD_DEFAULT_MERGE_STRATEGY` — one of `squash`, `merge`, `rebase`, `semilinear`; default `squash`

### API (`internal/api/repos.go` — new file)

```go
func (c *Client) ListRepositories() ([]GitRepository, error)
// GET {project}/_apis/git/repositories
```

### Repo Picker Modal (`internal/ui/repopicker/picker.go` — new file)

A floating overlay rendered on top of the PR list.

**Behavior:**
- On startup: if `config.Repos` is empty, the list view immediately renders the empty state
  (no spinner, no API call for PRs). The modal is not auto-opened; instead the empty state
  message tells the user to press `R`.
- `R` from list view: fetches all repos via `ListRepositories()`, opens the picker.
- The picker shows a searchable, multi-select list:
  - `textinput` at the top for fuzzy filtering by repo name
  - Each row: `[x]` or `[ ]` checkbox + repo name
  - `space` toggles selection
  - Pre-checks repos that match the current in-memory selection
  - `enter` confirms and emits `RepoPickerDoneMsg`
  - `ctrl+s` confirms AND writes selected repo names to `AZBOARD_REPOS` in the config file
  - `esc` cancels without changing selection
- Rendered as a centered box with a lipgloss border, width ~60% of terminal, height ~60%.

**On `RepoPickerDoneMsg`:**
- `ListModel` updates its `repos` field and re-fetches PRs.

### PR List (`internal/ui/prs/list.go`)

- Store `repos []string` in `ListModel`.
- `fetchPRs()`: if `repos` is empty, return immediately with `PRsLoadedMsg{prs: nil}` (triggers empty state).
  Otherwise, fire one `ListPullRequests` call per repo in parallel (using `tea.Batch`), merge results, dedup by PR ID, sort by creation date descending.
- Scope bar: show active repo names as a secondary pill row below the scope tabs when repos are selected.
- Empty state (no repos): render centered message:
  ```
  No repositories selected.
  Press R to open the repo picker, or set AZBOARD_REPOS in your config file.
  ```

---

## Phase 2 — PR Lifecycle

### API additions (`internal/api/pullrequests.go`)

```go
func (c *Client) CreatePullRequest(
    repoID, title, sourceBranch, targetBranch, description string,
    isDraft bool,
) (PullRequest, error)
// POST {project}/_apis/git/repositories/{repo}/pullrequests
// Body: CreatePullRequestRequest

func (c *Client) MergePullRequest(
    repoID string, prID int,
    strategy string,        // "squash" | "noFastForward" | "rebase" | "rebaseMerge"
    commitMsg string,
    deleteSourceBranch bool,
) error
// PATCH {project}/_apis/git/repositories/{repo}/pullrequests/{id}
// Body: { status: "completed", completionOptions: { mergeStrategy, deleteSourceBranch, mergeCommitMessage } }

func (c *Client) AbandonPullRequest(repoID string, prID int) error
// PATCH — body: { status: "abandoned" }

func (c *Client) ToggleDraft(repoID string, prID int, isDraft bool) error
// PATCH — body: { isDraft: true|false }
```

### New types (`internal/api/types.go`)

```go
type CreatePullRequestRequest struct {
    Title         string `json:"title"`
    Description   string `json:"description,omitempty"`
    SourceRefName string `json:"sourceRefName"` // "refs/heads/branch-name"
    TargetRefName string `json:"targetRefName"`
    IsDraft       bool   `json:"isDraft"`
}

type CompletePullRequestRequest struct {
    Status            string            `json:"status"`
    CompletionOptions CompletionOptions `json:"completionOptions"`
}

type CompletionOptions struct {
    MergeStrategy      string `json:"mergeStrategy"`
    DeleteSourceBranch bool   `json:"deleteSourceBranch"`
    MergeCommitMessage string `json:"mergeCommitMessage,omitempty"`
}
```

ADO merge strategy string mapping:

| UI Label         | API value        |
|------------------|------------------|
| Squash merge     | `squash`         |
| Merge commit     | `noFastForward`  |
| Rebase           | `rebase`         |
| Semi-linear      | `rebaseMerge`    |

### PR Creation Form (`internal/ui/prs/create.go` — new file)

Multi-step form following the existing compose/confirm pattern in `detail.go`.

**Steps (linear, `tab`/`enter` advances, `esc` goes back one step):**
1. **Repo** — text input with instructions: user types the exact repo name (must match a repo
   in their configured/session repos list). Validated on advance.
2. **Title** — single-line `textinput`
3. **Source branch** — single-line `textinput`, prefixed with `refs/heads/` automatically
4. **Target branch** — single-line `textinput`, prefixed with `refs/heads/` automatically
5. **Description** — multi-line `textarea`, optional (`ctrl+s` to skip/submit)
6. **Draft?** — yes/no toggle (`y`/`n` or `space`)

**On submit** (`ctrl+s` from any step with required fields filled):
- Calls `CreatePullRequest`
- On success: emits `PRCreatedMsg`, navigates to new PR detail view, shows flash "PR created"
- On error: shows flash error, stays in form

Accessible via `n` key from list view.

### Detail View additions (`internal/ui/prs/detail.go`)

New keybindings in overview pane (normal mode):

| Key | Action |
|-----|--------|
| `m` | Open merge dialog |
| `X` | Abandon PR (confirm prompt) |
| `D` | Toggle draft/ready (confirm prompt) |
| `o` | Open PR URL in browser (`xdg-open` on Linux, `open` on macOS) |

**Merge dialog** (inline, replaces confirm prompt area):
- Radio selector for merge strategy (arrow keys to change, default from config)
- Toggle: "Delete source branch after merge? [Y/n]"
- `enter` / `ctrl+s` to confirm, `esc` to cancel
- On success: flash "PR merged", emit `PRMergedMsg`, return to list

**Abandon confirm:**
- Standard confirm prompt: `Abandon this PR? [y/N]`
- On success: flash "PR abandoned", emit `PRAbandonedMsg`, return to list

**Draft toggle confirm:**
- `"Convert to draft? [y/N]"` or `"Mark as ready for review? [y/N]"`
- On success: flash message, refresh PR detail

**Open in browser:**
- Construct URL: `{orgURL}/{project}/_git/{repo}/pullrequest/{id}`
- Use `exec.Command("xdg-open", url)` on Linux, `exec.Command("open", url)` on macOS
- Use `runtime.GOOS` to select the right command

---

## Phase 3 — Diff Improvements

### Line number gutter (`internal/ui/prs/diff.go`)

Parse `@@` hunk headers to track line numbers. The header format is:
```
@@ -35,6 +35,12 @@ optional context
```

Gutter format (10 chars wide):
```
  35   35 │ unchanged line
       36 │ added line      (old side blank)
  35      │ deleted line    (new side blank)
```

Style: muted foreground, `│` separator in `Border` color.

Implementation:
- `colorizeDiff()` becomes stateful: tracks `oldLine`, `newLine` counters, resets on each
  `@@` header.
- Gutter is prepended to each rendered line before applying diff color.
- Long-line wrapping must account for the gutter width (subtract 10 from available width).

### Per-iteration diff (`internal/ui/prs/files.go` + `detail.go`)

**Iteration picker** rendered above the file tree in the files pane:
```
  Iteration 3 of 3  (Mar 28, 2026)   ← →
```
- `←`/`→` keys cycle iterations
- Iteration list fetched once when entering files pane (already fetched in `fetchFiles`)
- Changing iteration re-fires `fetchDiff` with new `sourceRefCommit`/`targetRefCommit`
- Defaults to latest (highest ID) iteration on load

**`FilesModel` changes:**
- Add `iterations []Iteration`, `currentIteration int` fields
- Iteration header rendered above the tree in `View()`

### Inline diff comments (`internal/ui/prs/diff.go` + API)

**Diff viewport cursor mode:**
- `DiffModel` gains a `cursorLine int` and `cursorMode bool`
- `j`/`k` (or `↑`/`↓`) move the cursor line when in cursor mode
- `i` key enters cursor mode; `esc` exits it
- Cursor line is highlighted with a distinct background

**`c` key in cursor mode:**
- Switches `DetailModel` to compose mode, storing `pendingDiffComment` with file path +
  line number
- On submit, calls extended `CreateThread` with `threadContext`

**API extension** (`internal/api/pullrequests.go`):
```go
type ThreadContext struct {
    FilePath       string     `json:"filePath"`
    RightFileStart *LineRange `json:"rightFileStart,omitempty"`
    RightFileEnd   *LineRange `json:"rightFileEnd,omitempty"`
}

type LineRange struct {
    Line   int `json:"line"`
    Offset int `json:"offset"`
}

// CreateThread updated signature:
func (c *Client) CreateThread(
    repoID string, prID int,
    content string,
    ctx *ThreadContext, // nil for general comments
) (Thread, error)
```

---

## Phase 4 — Work Items

### API (`internal/api/workitems.go` — new file)

```go
func (c *Client) ListWorkItems(types []string, assignedTo string) ([]WorkItem, error)
// POST {project}/_apis/wit/wiql
// Body: { query: "SELECT [Id],[Title],[State],[AssignedTo],[WorkItemType]
//         FROM WorkItems
//         WHERE [System.TeamProject] = '{project}'
//         AND [System.WorkItemType] IN ('{types}')
//         [AND [System.AssignedTo] = '{assignedTo}']   -- only for "My Work" scope
//         ORDER BY [System.ChangedDate] DESC" }
// Then: GET {project}/_apis/wit/workitems?ids=1,2,3&$expand=fields

func (c *Client) GetWorkItem(id int) (WorkItem, error)
// GET {project}/_apis/wit/workitems/{id}?$expand=all

func (c *Client) UpdateWorkItemState(id int, state string) error
// PATCH {project}/_apis/wit/workitems/{id}
// Content-Type: application/json-patch+json
// Body: [{ "op": "add", "path": "/fields/System.State", "value": "{state}" }]

func (c *Client) AddWorkItemComment(id int, text string) error
// POST {project}/_apis/wit/workitems/{id}/comments

func (c *Client) LinkWorkItemToPR(workItemID int, prArtifactURL string) error
// PATCH {project}/_apis/wit/workitems/{workItemID}
// Body: [{ "op": "add", "path": "/relations/-",
//          "value": { "rel": "ArtifactLink",
//                     "url": "{prArtifactURL}",
//                     "attributes": { "name": "Pull Request" } } }]
```

PR artifact URL format: `vstfs:///Git/PullRequestId/{projectID}/{repoID}/{prID}`
(requires fetching project ID at startup or deriving from existing API responses).

### New types (`internal/api/types.go`)

```go
type WorkItem struct {
    ID     int              `json:"id"`
    Fields WorkItemFields   `json:"fields"`
    URL    string           `json:"url"`
}

type WorkItemFields struct {
    Title        string      `json:"System.Title"`
    State        string      `json:"System.State"`
    WorkItemType string      `json:"System.WorkItemType"`
    AssignedTo   IdentityRef `json:"System.AssignedTo"`
    Description  string      `json:"System.Description"`
    CreatedDate  time.Time   `json:"System.CreatedDate"`
    ChangedDate  time.Time   `json:"System.ChangedDate"`
    AreaPath     string      `json:"System.AreaPath"`
}

type WIQLRequest struct {
    Query string `json:"query"`
}

type WIQLResult struct {
    WorkItems []WIQLRef `json:"workItems"`
}

type WIQLRef struct {
    ID  int    `json:"id"`
    URL string `json:"url"`
}

type WorkItemPatchOp struct {
    Op    string `json:"op"`
    Path  string `json:"path"`
    Value any    `json:"value"`
}

// Valid states per work item type (used to build state transition picker)
var WorkItemStates = map[string][]string{
    "Bug":        {"New", "Active", "Resolved", "Closed"},
    "User Story": {"New", "Active", "Resolved", "Closed"},
    "Task":       {"To Do", "In Progress", "Done"},
    "Feature":    {"New", "In Progress", "Resolved", "Closed"},
    "Epic":       {"New", "In Progress", "Resolved", "Closed"},
}
```

### Work Item List (`internal/ui/workitems/list.go` — new file)

Structural mirror of `internal/ui/prs/list.go`.

**Scopes:**
- `My Work` — WIQL with `AssignedTo = @me` (or current user display name)
- `Active` — all active items of configured types
- `All` — no state filter

**Table columns:**
- Type (icon, fixed 3 chars): `◉` Bug, `◈` User Story, `◻` Task, `◆` Epic, `◇` Feature
- ID (fixed 6 chars)
- Title (proportional, largest share)
- State (fixed 12 chars, styled badge)
- Assigned To (fixed 20 chars, truncated)

**Keys:**
- `[`/`]` — cycle scopes
- `/` — filter (same client-side multi-term AND, searches title, ID, state, assignee, type)
- `enter` — open work item detail
- `r` — refresh

**Empty state:** `"No work items found."`

### Work Item Detail (`internal/ui/workitems/detail.go` — new file)

Single viewport pane (no sub-panes for MVP).

**Rendered sections:**
1. Title (bold)
2. Metadata row: Type • State (badge) • Assigned To • Area Path
3. Created / Last Updated dates
4. Description (word-wrapped, plain text — HTML stripped from ADO rich text fields)
5. Comments (author, date, body — same render style as PR threads)

**Keys:**

| Key | Action |
|-----|--------|
| `s` | State transition picker |
| `c` | Add comment (compose textarea) |
| `L` | Link to PR — text input for PR ID |
| `o` | Open in browser |
| `r` | Refresh |
| `esc` | Back to work item list |

**State transition picker:**
- Inline prompt listing valid next states for this work item type (from `WorkItemStates` map)
- Arrow keys to select, `enter` to confirm, `esc` to cancel
- On success: flash "State updated to {state}", refresh detail

**Link to PR:**
- Text input: `"Enter PR ID: "`
- On submit: calls `LinkWorkItemToPR` with constructed artifact URL
- On success: flash "Linked to PR #{id}"

### App Shell (`internal/ui/app.go`)

- Fetch `currentUserID` (and user display name) at startup via `GetCurrentUserID()` +
  `GetCurrentUserProfile()` (or parse from `connectionData`). Store on `AppModel`.
  Pass down to `WorkItemListModel` for "My Work" scope.
- `Tab` key cycles `viewList` → `viewWorkItems` → `viewList` (two tabs for MVP, Sprint Board
  tab remains a stub).
- `viewWorkItems`: lazy-init `WorkItemListModel` on first tab switch. On subsequent switches
  the existing model is reused (no re-fetch unless user presses `r`).
- `BackToWorkItemListMsg` — new message type for returning from detail to list within work
  items tab.

---

## Phase 5 — Config & Polish

### Final config keys

| Key | Default | Description |
|-----|---------|-------------|
| `AZBOARD_REPOS` | _(empty)_ | Comma-separated repo names to load |
| `AZBOARD_WORK_ITEM_TYPES` | `User Story,Bug,Task,Feature,Epic` | Work item types to include |
| `AZBOARD_DEFAULT_MERGE_STRATEGY` | `squash` | Default strategy in merge dialog |

### Help overlay updates (`internal/ui/app.go`)

Add sections:
- **Work Items** — scopes, filter, state transition, comment, link to PR, open in browser
- **PR Actions** — merge, abandon, draft toggle, create, open in browser
- **Diff** — iteration switching, cursor mode, inline comment

### Theme additions (`internal/ui/theme/styles.go`)

- Work item type icon styles (colored per type)
- Work item state badge styles (mirrors PR status badges)
- Diff gutter style (muted, fixed-width)
- Merge strategy selector style (active/inactive row)

---

## New Files Summary

| File | Purpose |
|------|---------|
| `internal/api/repos.go` | `ListRepositories` API method |
| `internal/api/workitems.go` | All work item API methods (WIQL, get, update, comment, link) |
| `internal/ui/repopicker/picker.go` | Floating repo selection modal |
| `internal/ui/prs/create.go` | Multi-step PR creation form |
| `internal/ui/workitems/list.go` | Work item list view |
| `internal/ui/workitems/detail.go` | Work item detail view |

## Modified Files Summary

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add `Repos`, `WorkItemTypes`, `DefaultMergeStrategy` fields |
| `internal/api/types.go` | Work item types, PR create/merge request types, `ThreadContext` |
| `internal/api/pullrequests.go` | `CreatePR`, `MergePR`, `AbandonPR`, `ToggleDraft`; extend `CreateThread` |
| `internal/ui/app.go` | Tab switching, `viewWorkItems`, startup user identity fetch, pass `currentUserID` |
| `internal/ui/prs/list.go` | Repo-filtered fetching, empty state, `n` key for create, `R` for repo picker |
| `internal/ui/prs/detail.go` | Merge/abandon/draft toggle/browser actions, inline diff comment wiring |
| `internal/ui/prs/diff.go` | Line number gutter, cursor mode for inline commenting |
| `internal/ui/prs/files.go` | Iteration picker UI (header row, `←`/`→` navigation) |
| `internal/ui/keys.go` | New keybindings for all added actions |
| `internal/ui/theme/styles.go` | Work item styles, gutter style, merge strategy selector style |
| `main.go` | Pass config fields down; user identity fetch at startup |

---

## Out of Scope for MVP

The following are explicitly deferred:

- Sprint board / kanban view
- CI/CD pipeline status and logs
- Branch management (use lazygit)
- Multi-org / multi-project switching
- Markdown rendering in PR descriptions
- Configurable keybindings
- Notifications
- PR reviewer assignment at creation time
- Work item creation / editing fields
- External diff pager support (delta, difftastic)
- Binary releases / install scripts
