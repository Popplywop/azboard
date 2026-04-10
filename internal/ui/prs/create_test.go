package prs

import (
	"testing"
)

func TestBranchPickerSetAll(t *testing.T) {
	p := newBranchPicker()
	p.setAll([]string{"main", "feature/login", "feature/api"})

	if len(p.all) != 3 {
		t.Errorf("expected 3 branches, got %d", len(p.all))
	}
	if len(p.filtered) != 3 {
		t.Errorf("expected 3 filtered, got %d", len(p.filtered))
	}
	if p.cursor != 0 {
		t.Errorf("cursor should be 0, got %d", p.cursor)
	}
}

func TestBranchPickerApplyFilter(t *testing.T) {
	p := newBranchPicker()
	p.setAll([]string{"main", "feature/login", "feature/api", "bugfix/crash"})

	p.filter.SetValue("feature")
	p.applyFilter()

	if len(p.filtered) != 2 {
		t.Errorf("expected 2 filtered branches, got %d", len(p.filtered))
	}
}

func TestBranchPickerApplyFilterEmpty(t *testing.T) {
	p := newBranchPicker()
	p.setAll([]string{"main", "develop"})

	p.filter.SetValue("")
	p.applyFilter()

	if len(p.filtered) != 2 {
		t.Errorf("empty filter should return all, got %d", len(p.filtered))
	}
}

func TestBranchPickerApplyFilterCaseInsensitive(t *testing.T) {
	p := newBranchPicker()
	p.setAll([]string{"Main", "Feature/Login"})

	p.filter.SetValue("main")
	p.applyFilter()

	if len(p.filtered) != 1 || p.filtered[0] != "Main" {
		t.Errorf("expected [Main], got %v", p.filtered)
	}
}

func TestBranchPickerApplyFilterClampsCursor(t *testing.T) {
	p := newBranchPicker()
	p.setAll([]string{"main", "develop", "feature"})
	p.cursor = 2

	p.filter.SetValue("main")
	p.applyFilter()

	if p.cursor != 0 {
		t.Errorf("cursor should be clamped to 0, got %d", p.cursor)
	}
}

func TestBranchPickerSelected(t *testing.T) {
	p := newBranchPicker()
	p.setAll([]string{"main", "develop"})
	p.cursor = 1

	if got := p.selected(); got != "develop" {
		t.Errorf("expected 'develop', got %q", got)
	}
}

func TestBranchPickerSelectedEmpty(t *testing.T) {
	p := newBranchPicker()

	if got := p.selected(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestBranchPickerApplyFilterNoMatch(t *testing.T) {
	p := newBranchPicker()
	p.setAll([]string{"main", "develop"})

	p.filter.SetValue("nonexistent")
	p.applyFilter()

	if len(p.filtered) != 0 {
		t.Errorf("expected 0 filtered, got %d", len(p.filtered))
	}
	if p.cursor != 0 {
		t.Errorf("cursor should be 0 for empty list, got %d", p.cursor)
	}
}
