package theme

import "charm.land/lipgloss/v2"

var (
	// Colors
	Primary = lipgloss.Color("#7B68EE") // Medium slate blue
	Success = lipgloss.Color("#28A745")
	Warning = lipgloss.Color("#FFC107")
	Danger  = lipgloss.Color("#DC3545")
	Info    = lipgloss.Color("#17A2B8")
	Muted   = lipgloss.Color("#6C757D")
	White   = lipgloss.Color("#FFFFFF")
	Subtle  = lipgloss.Color("#383838")
	Border  = lipgloss.Color("#444444")

	// Tab bar
	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Primary).
			Padding(0, 2)

	InactiveTab = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 2)

	TabBar = lipgloss.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Border).
		MarginBottom(1)

	// Status bar
	StatusBar = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 1)

	// PR Status badges
	StatusActive = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true)

	StatusCompleted = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	StatusAbandoned = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	StatusDraft = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Reviewer votes
	VoteApproved = lipgloss.NewStyle().
			Foreground(Success)

	// vote=5: "Approved with suggestions" — yellow to distinguish from a full approval
	VoteApprovedWithSuggestions = lipgloss.NewStyle().
					Foreground(Warning)

	VoteRejected = lipgloss.NewStyle().
			Foreground(Danger)

	VoteWaiting = lipgloss.NewStyle().
			Foreground(Warning)

	VoteNone = lipgloss.NewStyle().
			Foreground(Muted)

	// Detail view
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		MarginBottom(1)

	Label = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Border).
			MarginTop(1).
			MarginBottom(1)

	CommentAuthor = lipgloss.NewStyle().
			Bold(true).
			Foreground(Info)

	CommentDate = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	CommentBody = lipgloss.NewStyle().
			PaddingLeft(2)

	ThreadStatus = lipgloss.NewStyle().
			Foreground(Warning)

	FilePath = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Help
	HelpKey = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)

	HelpDesc = lipgloss.NewStyle().
			Foreground(Muted)

	// Spinner / loading
	Spinner = lipgloss.NewStyle().
		Foreground(Primary)

	ErrorText = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	// Table border
	TableBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border)

	// Filter bar
	FilterPrompt = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	FilterText = lipgloss.NewStyle().
			Foreground(White)

	FilterBar = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Border).
			PaddingLeft(1)

	FilterCount = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Focused thread highlight
	FocusedThread = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(Primary).
			PaddingLeft(1)

	UnfocusedThread = lipgloss.NewStyle().
			PaddingLeft(3)

	// Compose textarea
	ComposeLabel = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	ComposeHint = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Confirm dialog
	ConfirmPrompt = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true).
			PaddingLeft(1)

	ConfirmHint = lipgloss.NewStyle().
			Foreground(Muted).
			PaddingLeft(1)

	// Success flash message
	SuccessText = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	// Diff view
	DiffAdd = lipgloss.NewStyle().
		Foreground(Success)

	DiffDelete = lipgloss.NewStyle().
			Foreground(Danger)

	DiffHunk = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true)

	DiffMeta = lipgloss.NewStyle().
			Foreground(Muted).
			Bold(true)
)

// StatusStyle returns the appropriate style for a PR status string.
func StatusStyle(status string, isDraft bool) lipgloss.Style {
	if isDraft {
		return StatusDraft
	}
	switch status {
	case "active":
		return StatusActive
	case "completed":
		return StatusCompleted
	case "abandoned":
		return StatusAbandoned
	default:
		return lipgloss.NewStyle()
	}
}

// VoteStyle returns the appropriate style for a reviewer vote value.
// Vote values: 10=approved, 5=approved with suggestions, 0=no vote,
// -5=waiting for author, -10=rejected.
func VoteStyle(vote int) lipgloss.Style {
	switch vote {
	case 10:
		return VoteApproved
	case 5:
		return VoteApprovedWithSuggestions
	case 0:
		return VoteNone
	case -5:
		return VoteWaiting
	case -10:
		return VoteRejected
	default:
		return VoteNone // Unknown vote value — neutral
	}
}
