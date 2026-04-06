package ui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Quit                   key.Binding
	Back                   key.Binding
	Select                 key.Binding
	Refresh                key.Binding
	Help                   key.Binding
	Tab                    key.Binding
	Up                     key.Binding
	Down                   key.Binding
	Filter                 key.Binding
	NextThread             key.Binding
	PrevThread             key.Binding
	Reply                  key.Binding
	NewThread              key.Binding
	ResolveThread          key.Binding
	Approve                key.Binding
	ApproveWithSuggestions key.Binding
	Reject                 key.Binding
	WaitForAuthor          key.Binding
	ResetVote              key.Binding
	Submit                 key.Binding
	// PR lifecycle
	Merge       key.Binding
	Abandon     key.Binding
	DraftToggle key.Binding
	OpenBrowser key.Binding
	// List
	RepoPicker key.Binding
	CreatePR   key.Binding
	// Work items
	StateTransition key.Binding
	AddComment      key.Binding
	LinkPR          key.Binding
}

var Keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch tab"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	NextThread: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next thread"),
	),
	PrevThread: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "prev thread"),
	),
	Reply: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "reply to thread"),
	),
	NewThread: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "new comment"),
	),
	ResolveThread: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "resolve/reactivate"),
	),
	Approve: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "approve"),
	),
	ApproveWithSuggestions: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "approve w/ suggestions"),
	),
	Reject: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "reject"),
	),
	WaitForAuthor: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "wait for author"),
	),
	ResetVote: key.NewBinding(
		key.WithKeys("0"),
		key.WithHelp("0", "reset vote"),
	),
	Submit: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "submit"),
	),
	Merge: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "merge PR"),
	),
	Abandon: key.NewBinding(
		key.WithKeys("X"),
		key.WithHelp("X", "abandon PR"),
	),
	DraftToggle: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "toggle draft/ready"),
	),
	OpenBrowser: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	RepoPicker: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "repo picker"),
	),
	CreatePR: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "create PR"),
	),
	StateTransition: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "state transition"),
	),
	AddComment: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "add comment"),
	),
	LinkPR: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "link to PR"),
	),
}
