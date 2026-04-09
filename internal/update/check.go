package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const releasesURL = "https://api.github.com/repos/Popplywop/azboard/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckLatestVersion checks GitHub for the latest release. Returns the latest
// version string and whether it is newer than current. Returns ("", false) on
// any error (network, parse, etc.) so callers can silently ignore failures.
func CheckLatestVersion(current string) (string, bool) {
	if current == "" || current == "dev" {
		return "", false
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(releasesURL)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", false
	}

	latest := release.TagName
	if latest == "" {
		return "", false
	}

	if isNewer(latest, current) {
		return latest, true
	}
	return latest, false
}

// isNewer returns true if latest is a higher semver than current.
// Both may have an optional "v" prefix.
func isNewer(latest, current string) bool {
	lMajor, lMinor, lPatch, lok := parseSemver(latest)
	cMajor, cMinor, cPatch, cok := parseSemver(current)
	if !lok || !cok {
		// Fall back to string comparison if not valid semver
		return strings.TrimPrefix(latest, "v") != strings.TrimPrefix(current, "v")
	}

	if lMajor != cMajor {
		return lMajor > cMajor
	}
	if lMinor != cMinor {
		return lMinor > cMinor
	}
	return lPatch > cPatch
}

// parseSemver parses "v1.2.3" or "1.2.3" into (major, minor, patch, ok).
func parseSemver(v string) (int, int, int, bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	// Strip any pre-release suffix (e.g. "1-rc1")
	parts[2] = strings.SplitN(parts[2], "-", 2)[0]

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

// FormatUpdateNotice returns a user-friendly update message.
func FormatUpdateNotice(latest, current string) string {
	return fmt.Sprintf("Update available: %s (you have %s)", latest, current)
}
