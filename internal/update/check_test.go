package update

import "testing"

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input                 string
		major, minor, patch   int
		ok                    bool
	}{
		{"v1.2.3", 1, 2, 3, true},
		{"1.2.3", 1, 2, 3, true},
		{"v0.0.1", 0, 0, 1, true},
		{"v1.0.0-rc1", 1, 0, 0, true},
		{"v10.20.30", 10, 20, 30, true},
		{"dev", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"v1.2", 0, 0, 0, false},
		{"v1", 0, 0, 0, false},
		{"abc.def.ghi", 0, 0, 0, false},
	}
	for _, tt := range tests {
		major, minor, patch, ok := parseSemver(tt.input)
		if ok != tt.ok {
			t.Errorf("parseSemver(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && (major != tt.major || minor != tt.minor || patch != tt.patch) {
			t.Errorf("parseSemver(%q) = (%d,%d,%d), want (%d,%d,%d)",
				tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.1.0", "v1.0.0", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", false},
		{"v0.1.0", "v0.0.1", true},
		{"0.2.0", "v0.1.0", true},
		{"v0.1.0", "0.1.0", false},
	}
	for _, tt := range tests {
		got := isNewer(tt.latest, tt.current)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestCheckLatestVersionDevSkipped(t *testing.T) {
	_, hasUpdate := CheckLatestVersion("dev")
	if hasUpdate {
		t.Error("expected no update for dev version")
	}
	_, hasUpdate = CheckLatestVersion("")
	if hasUpdate {
		t.Error("expected no update for empty version")
	}
}

func TestFormatUpdateNotice(t *testing.T) {
	got := FormatUpdateNotice("v1.2.0", "v1.0.0")
	if got != "Update available: v1.2.0 (you have v1.0.0)" {
		t.Errorf("FormatUpdateNotice = %q", got)
	}
}
