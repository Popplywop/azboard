package uiutil

import (
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf8"
)

// Truncate shortens s to maxLen runes, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "\u2026"
}

// WordWrap wraps text to the given width at word boundaries, preserving newlines.
func WordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for i, paragraph := range strings.Split(text, "\n") {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(WrapLine(paragraph, width))
	}
	return result.String()
}

// WrapLine wraps a single line of text to the given width at word boundaries.
// Words longer than width are split on rune boundaries.
func WrapLine(line string, width int) string {
	if utf8.RuneCountInString(line) <= width {
		return line
	}

	var result strings.Builder
	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}

	lineLen := 0
	for i, word := range words {
		wordLen := utf8.RuneCountInString(word)

		if i == 0 {
			if wordLen > width {
				runes := []rune(word)
				for len(runes) > width {
					result.WriteString(string(runes[:width]))
					result.WriteString("\n")
					runes = runes[width:]
				}
				result.WriteString(string(runes))
				lineLen = len(runes)
			} else {
				result.WriteString(word)
				lineLen = wordLen
			}
			continue
		}

		if lineLen+1+wordLen > width {
			result.WriteString("\n")
			if wordLen > width {
				runes := []rune(word)
				for len(runes) > width {
					result.WriteString(string(runes[:width]))
					result.WriteString("\n")
					runes = runes[width:]
				}
				result.WriteString(string(runes))
				lineLen = len(runes)
			} else {
				result.WriteString(word)
				lineLen = wordLen
			}
		} else {
			result.WriteString(" ")
			result.WriteString(word)
			lineLen += 1 + wordLen
		}
	}

	return result.String()
}

// OpenBrowserURL opens a URL in the default system browser.
func OpenBrowserURL(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", "", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
