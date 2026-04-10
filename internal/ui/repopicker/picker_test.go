package repopicker

import (
	"testing"

	"github.com/popplywop/azboard/internal/api"
)

func makeRepos(names ...string) []api.GitRepository {
	repos := make([]api.GitRepository, len(names))
	for i, n := range names {
		repos[i] = api.GitRepository{ID: n, Name: n}
	}
	return repos
}

func TestApplyFilterEmpty(t *testing.T) {
	m := NewPickerModel(nil)
	m.repos = makeRepos("alpha", "beta", "gamma")

	result := m.applyFilter()
	if len(result) != 3 {
		t.Errorf("empty filter should return all repos, got %d", len(result))
	}
}

func TestApplyFilterMatches(t *testing.T) {
	m := NewPickerModel(nil)
	m.repos = makeRepos("inventory-api", "web-portal", "auth-service")
	m.filter.SetValue("api")

	result := m.applyFilter()
	if len(result) != 1 || result[0].Name != "inventory-api" {
		t.Errorf("expected [inventory-api], got %v", result)
	}
}

func TestApplyFilterCaseInsensitive(t *testing.T) {
	m := NewPickerModel(nil)
	m.repos = makeRepos("MyRepo")
	m.filter.SetValue("myrepo")

	result := m.applyFilter()
	if len(result) != 1 {
		t.Errorf("filter should be case-insensitive, got %d results", len(result))
	}
}

func TestApplyFilterNoMatch(t *testing.T) {
	m := NewPickerModel(nil)
	m.repos = makeRepos("alpha", "beta")
	m.filter.SetValue("xyz")

	result := m.applyFilter()
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestSelectedNames(t *testing.T) {
	m := NewPickerModel([]string{"beta", "alpha"})

	names := m.selectedNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 selected, got %d", len(names))
	}
	// Should be sorted
	if names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("expected sorted [alpha beta], got %v", names)
	}
}

func TestSelectedNamesEmpty(t *testing.T) {
	m := NewPickerModel(nil)
	names := m.selectedNames()
	if len(names) != 0 {
		t.Errorf("expected 0 selected, got %d", len(names))
	}
}

func TestClampCursor(t *testing.T) {
	m := NewPickerModel(nil)
	m.filtered = makeRepos("a", "b")
	m.cursor = 5

	m.clampCursor()
	if m.cursor != 0 {
		t.Errorf("cursor should be clamped to 0, got %d", m.cursor)
	}
}

func TestClampCursorWithinBounds(t *testing.T) {
	m := NewPickerModel(nil)
	m.filtered = makeRepos("a", "b", "c")
	m.cursor = 1

	m.clampCursor()
	if m.cursor != 1 {
		t.Errorf("cursor should stay at 1, got %d", m.cursor)
	}
}

func TestClearFilter(t *testing.T) {
	m := NewPickerModel(nil)
	m.repos = makeRepos("alpha", "beta")
	m.filter.SetValue("alpha")
	m.filtered = m.applyFilter()
	m.cursor = 0

	m.clearFilter()
	if m.filter.Value() != "" {
		t.Error("filter should be cleared")
	}
	if len(m.filtered) != 2 {
		t.Errorf("filtered should be reset to all repos, got %d", len(m.filtered))
	}
}

func TestNewPickerModelPreselected(t *testing.T) {
	m := NewPickerModel([]string{"repo-a", "repo-b"})
	if !m.selected["repo-a"] || !m.selected["repo-b"] {
		t.Error("preselected repos should be marked in selected map")
	}
	if m.selected["repo-c"] {
		t.Error("non-preselected repos should not be in selected map")
	}
}
