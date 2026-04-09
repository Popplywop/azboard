package uiutil

import "testing"

func TestTruncateShort(t *testing.T) {
	got := Truncate("hi", 10)
	if got != "hi" {
		t.Errorf("Truncate(%q, 10) = %q, want %q", "hi", got, "hi")
	}
}

func TestTruncateExactLength(t *testing.T) {
	got := Truncate("hello", 5)
	if got != "hello" {
		t.Errorf("Truncate(%q, 5) = %q, want %q", "hello", got, "hello")
	}
}

func TestTruncateLong(t *testing.T) {
	got := Truncate("hello world", 6)
	if got != "hello\u2026" {
		t.Errorf("Truncate(%q, 6) = %q, want %q", "hello world", got, "hello\u2026")
	}
}

func TestTruncateMultibyte(t *testing.T) {
	input := "\u4f60\u597d\u4e16\u754c" // 4 runes
	got := Truncate(input, 3)
	want := "\u4f60\u597d\u2026" // 2 runes + ellipsis
	if got != want {
		t.Errorf("Truncate(%q, 3) = %q, want %q", input, got, want)
	}
}

func TestTruncateZero(t *testing.T) {
	got := Truncate("hello", 0)
	if got != "" {
		t.Errorf("Truncate(%q, 0) = %q, want empty", "hello", got)
	}
}

func TestTruncateOne(t *testing.T) {
	got := Truncate("hello", 1)
	if got != "h" {
		t.Errorf("Truncate(%q, 1) = %q, want %q", "hello", got, "h")
	}
}

func TestWordWrapShort(t *testing.T) {
	got := WordWrap("short", 80)
	if got != "short" {
		t.Errorf("WordWrap = %q, want %q", got, "short")
	}
}

func TestWordWrapLong(t *testing.T) {
	got := WordWrap("hello world foo", 11)
	if got != "hello world\nfoo" {
		t.Errorf("WordWrap = %q, want %q", got, "hello world\nfoo")
	}
}

func TestWordWrapMultipleParagraphs(t *testing.T) {
	got := WordWrap("line one\nline two", 80)
	if got != "line one\nline two" {
		t.Errorf("WordWrap = %q, want %q", got, "line one\nline two")
	}
}

func TestWordWrapLongWord(t *testing.T) {
	got := WrapLine("abcdefghij", 4)
	if got != "abcd\nefgh\nij" {
		t.Errorf("WrapLine = %q, want %q", got, "abcd\nefgh\nij")
	}
}

func TestWordWrapZeroWidth(t *testing.T) {
	got := WordWrap("hello", 0)
	if got != "hello" {
		t.Errorf("WordWrap(0) = %q, want passthrough", got)
	}
}

func TestWrapLineEmpty(t *testing.T) {
	got := WrapLine("", 80)
	if got != "" {
		t.Errorf("WrapLine(\"\") = %q, want empty", got)
	}
}

func TestWrapLineSingleWord(t *testing.T) {
	got := WrapLine("hello", 80)
	if got != "hello" {
		t.Errorf("WrapLine = %q, want %q", got, "hello")
	}
}
