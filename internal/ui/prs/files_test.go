package prs

import (
	"testing"

	"github.com/popplywop/azboard/internal/api"
)

func TestBuildTrieBasic(t *testing.T) {
	changes := []api.IterationChange{
		{Item: api.ChangeItem{Path: "/src/main.go"}, ChangeType: "edit"},
		{Item: api.ChangeItem{Path: "/src/util.go"}, ChangeType: "add"},
		{Item: api.ChangeItem{Path: "/README.md"}, ChangeType: "edit"},
	}
	root := buildTrie(changes)
	if root == nil {
		t.Fatal("buildTrie returned nil")
	}
	if len(root.children) != 2 { // src/ and README.md
		t.Errorf("root has %d children, want 2", len(root.children))
	}
	src, ok := root.children["src"]
	if !ok {
		t.Fatal("missing src child")
	}
	if !src.isDir {
		t.Error("src should be a directory")
	}
	if len(src.children) != 2 {
		t.Errorf("src has %d children, want 2", len(src.children))
	}
}

func TestBuildTrieDeepNesting(t *testing.T) {
	changes := []api.IterationChange{
		{Item: api.ChangeItem{Path: "/a/b/c/d.go"}, ChangeType: "edit"},
	}
	root := buildTrie(changes)
	a := root.children["a"]
	if a == nil || !a.isDir {
		t.Fatal("missing or non-dir 'a'")
	}
	b := a.children["b"]
	if b == nil || !b.isDir {
		t.Fatal("missing or non-dir 'b'")
	}
	c := b.children["c"]
	if c == nil || !c.isDir {
		t.Fatal("missing or non-dir 'c'")
	}
	d := c.children["d.go"]
	if d == nil || d.isDir {
		t.Fatal("missing or dir 'd.go'")
	}
}

func TestFlattenNodeCollapsed(t *testing.T) {
	changes := []api.IterationChange{
		{Item: api.ChangeItem{Path: "/src/main.go"}, ChangeType: "edit"},
		{Item: api.ChangeItem{Path: "/src/util.go"}, ChangeType: "add"},
	}
	root := buildTrie(changes)

	// Uncollapsed
	var entries []treeEntry
	flattenNode(root, "", true, map[string]bool{}, &entries)
	if len(entries) != 3 { // src dir + main.go + util.go
		t.Errorf("uncollapsed: got %d entries, want 3", len(entries))
	}

	// Collapsed
	entries = nil
	flattenNode(root, "", true, map[string]bool{"src": true}, &entries)
	if len(entries) != 1 { // just the src dir
		t.Errorf("collapsed: got %d entries, want 1", len(entries))
	}
	if !entries[0].isDir {
		t.Error("collapsed entry should be dir")
	}
}

func TestCanonicalDirPath(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"\u251c\u2500\u2500 src", "src"},
		{"\u2514\u2500\u2500 src", "src"},
		{"src", "src"},
		{"\u2502   \u251c\u2500\u2500 nested", "nested"},
	}
	for _, tt := range tests {
		got := canonicalDirPath(tt.input)
		if got != tt.want {
			t.Errorf("canonicalDirPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestChangeTypeIcon(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"add", "A"},
		{"Add", "A"},
		{"delete", "D"},
		{"rename", "R"},
		{"edit", "M"},
		{"unknown", "M"},
	}
	for _, tt := range tests {
		got := changeTypeIcon(tt.input)
		if got != tt.want {
			t.Errorf("changeTypeIcon(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
