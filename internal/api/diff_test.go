package api

import "testing"

func TestNormalizeLineEndingsCRLF(t *testing.T) {
	got := normalizeLineEndings("a\r\nb\r\nc")
	if got != "a\nb\nc" {
		t.Errorf("got %q, want %q", got, "a\nb\nc")
	}
}

func TestNormalizeLineEndingsCR(t *testing.T) {
	got := normalizeLineEndings("a\rb\rc")
	if got != "a\nb\nc" {
		t.Errorf("got %q, want %q", got, "a\nb\nc")
	}
}

func TestNormalizeLineEndingsLF(t *testing.T) {
	got := normalizeLineEndings("a\nb\nc")
	if got != "a\nb\nc" {
		t.Errorf("got %q, want %q", got, "a\nb\nc")
	}
}

func TestNormalizeLineEndingsMixed(t *testing.T) {
	got := normalizeLineEndings("a\r\nb\rc\n")
	if got != "a\nb\nc\n" {
		t.Errorf("got %q, want %q", got, "a\nb\nc\n")
	}
}

func TestEnsureTrailingNewlineWithout(t *testing.T) {
	got := ensureTrailingNewline("abc")
	if got != "abc\n" {
		t.Errorf("got %q, want %q", got, "abc\n")
	}
}

func TestEnsureTrailingNewlineWith(t *testing.T) {
	got := ensureTrailingNewline("abc\n")
	if got != "abc\n" {
		t.Errorf("got %q, want %q", got, "abc\n")
	}
}

func TestEnsureTrailingNewlineEmpty(t *testing.T) {
	got := ensureTrailingNewline("")
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}
