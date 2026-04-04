# azboard

A fast, keyboard-driven terminal UI for Azure DevOps. Review pull requests, manage work
items, and interact with your ADO project without leaving the terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Bubbles](https://github.com/charmbracelet/bubbles), and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

---

## Installation

Requires Go 1.21+.

```bash
git clone https://github.com/popplywop/azboard
cd azboard
go build -o azboard .
```

Move the binary somewhere on your `$PATH`:

```bash
mv azboard /usr/local/bin/azboard
```

---

## Configuration

Create `~/.config/azboard/config.env`:

```env
# Azure DevOps organization URL (required)
AZBOARD_ORG_URL=https://dev.azure.com/your-org

# Project name (required)
AZBOARD_PROJECT=your-project

# Auth method: "azcli" (default) or "pat"
AZBOARD_AUTH=azcli

# Personal access token (required if AUTH=pat)
AZBOARD_PAT=your-pat-token

# Repositories to load (comma-separated repo names)
# Leave empty to start with no repos and use the interactive picker (R key)
AZBOARD_REPOS=my-api,my-frontend

# Work item types to show (comma-separated)
# Default: User Story,Bug,Task,Feature,Epic
AZBOARD_WORK_ITEM_TYPES=User Story,Bug,Task

# Default merge strategy: squash (default), merge, rebase, semilinear
AZBOARD_DEFAULT_MERGE_STRATEGY=squash
```

Config priority: CLI flags > environment variables > config file.

### Auth methods

**Azure CLI (recommended):** Install the [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli)
and run `az login`. azboard will automatically obtain and refresh tokens.

```env
AZBOARD_AUTH=azcli
```

**Personal Access Token:** Generate a PAT in Azure DevOps with `Code (Read & Write)` and
`Work Items (Read & Write)` scopes.

```env
AZBOARD_AUTH=pat
AZBOARD_PAT=your-pat-token
```

---

## Usage

```bash
azboard
# or with overrides:
azboard --org https://dev.azure.com/my-org --project my-project
```

On first launch with no `AZBOARD_REPOS` configured, you will see an empty state. Press `R`
to open the repo picker and select which repositories to load.

---

## Features

### Pull Requests

- **List view** — all PRs across your configured repositories in a single table
- **Scoped views** — Active, Draft, Completed, Abandoned, All; cycle with `[` / `]`
- **Live filtering** — press `/` to filter by title, repo, author, status, PR ID, or reviewer
- **Create PR** — `n` to open a multi-step creation form (title, source branch, target branch,
  description, draft toggle)
- **PR detail** — title, branches, status, author, reviewers with vote icons, description,
  comment threads
- **Merge PR** — `m` to open the merge dialog; choose strategy (squash / merge commit /
  rebase / semi-linear) and optionally delete the source branch
- **Abandon PR** — `X` to abandon with confirmation
- **Draft toggle** — `D` to convert between draft and ready-for-review
- **Open in browser** — `o` to open the PR in your default browser
- **Voting** — approve (`a`), approve with suggestions (`A`), reject (`x`),
  wait for author (`w`), reset (`0`)
- **Comment threads** — read, reply (`c`), create new thread (`C`), resolve/reactivate (`s`),
  navigate with `n` / `N`

### Code Review

- **File tree** — browse changed files in a collapsible directory tree; `f` from PR detail
- **Diff viewer** — color-coded unified diff with line number gutters; `enter` on a file
- **Per-iteration diffs** — `←` / `→` in the files pane to compare different PR iterations
- **Inline diff comments** — press `i` to enter cursor mode in the diff, navigate to a line,
  press `c` to post a comment anchored to that line

### Work Items

- **List view** — User Stories, Bugs, Tasks, Features, and Epics in a filterable table
- **Scoped views** — My Work (assigned to you), Active, All
- **Work item detail** — title, type, state, assignee, area path, description, comments
- **State transitions** — `s` to pick the next state for a work item
- **Add comments** — `c` to add a comment to a work item
- **Link to PR** — `L` to link a work item to a PR by PR ID
- **Open in browser** — `o` to open the work item in Azure DevOps

### Repo Selection

- **Config-based** — set `AZBOARD_REPOS` in config for repos to load on startup
- **Interactive picker** — press `R` from the PR list to open a searchable, multi-select
  repo picker; `ctrl+s` in the picker saves the selection to your config file
- **Session-only** — repo selections made in the picker persist for the session; use
  `ctrl+s` to make them permanent

### Auth & Configuration

- **Azure CLI auth** — automatic token acquisition and refresh via `az account get-access-token`
- **PAT auth** — static personal access token
- **URL format support** — both `https://dev.azure.com/org` and `https://org.visualstudio.com`
- **Config priority** — CLI flags override environment variables, which override the config file

---

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `Tab` | Switch between Pull Requests and Work Items tabs |
| `?` | Toggle help overlay |
| `q` | Go back / quit |
| `ctrl+c` | Quit |

### PR List

| Key | Action |
|-----|--------|
| `[` / `]` | Cycle PR scopes (Active / Draft / Completed / Abandoned / All) |
| `/` | Open filter |
| `enter` | Open PR detail |
| `n` | Create new PR |
| `R` | Open repo picker |
| `r` | Refresh list |

### PR Detail — Overview

| Key | Action |
|-----|--------|
| `f` | Switch to files pane |
| `n` / `N` | Next / previous comment thread |
| `c` | Reply to focused thread |
| `C` | Create new comment thread |
| `s` | Toggle thread status (resolve / reactivate) |
| `a` | Approve |
| `A` | Approve with suggestions |
| `x` | Reject |
| `w` | Wait for author |
| `0` | Reset vote |
| `m` | Merge PR |
| `X` | Abandon PR |
| `D` | Toggle draft / ready |
| `o` | Open in browser |
| `r` | Refresh |
| `esc` | Unfocus thread / go back |

### PR Detail — Files Pane

| Key | Action |
|-----|--------|
| `←` / `→` | Previous / next iteration |
| `enter` | View diff for selected file / toggle directory |
| `r` | Refresh files |
| `esc` | Back to overview |

### PR Detail — Diff Pane

| Key | Action |
|-----|--------|
| `i` | Enter cursor mode (select a line) |
| `j` / `k` or `↑` / `↓` | Move cursor line (in cursor mode) |
| `c` | Post inline comment on cursor line (in cursor mode) |
| `esc` | Exit cursor mode / back to files |

### Work Item List

| Key | Action |
|-----|--------|
| `[` / `]` | Cycle scopes (My Work / Active / All) |
| `/` | Filter |
| `enter` | Open work item detail |
| `r` | Refresh |

### Work Item Detail

| Key | Action |
|-----|--------|
| `s` | State transition |
| `c` | Add comment |
| `L` | Link to PR |
| `o` | Open in browser |
| `r` | Refresh |
| `esc` | Back to list |

---

## Roadmap

See [MVP.md](./MVP.md) for the detailed implementation plan and phased feature breakdown.

**Post-MVP (planned):**
- Sprint board / kanban view
- CI/CD pipeline status and run logs
- PR reviewer assignment at creation time
- Markdown rendering in PR descriptions and work item descriptions
- Configurable keybindings
- Notifications
- External diff pager support (delta, difftastic)
- Multi-project switching

---

## Contributing

azboard is in active development. Bug reports and pull requests are welcome.

Build and run locally:

```bash
go build -o azboard . && ./azboard
```

Run with a custom config:

```bash
./azboard --config /path/to/config.env
```
