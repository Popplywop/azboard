package workitems

import "testing"

func TestStripHTMLBasic(t *testing.T) {
	got := stripHTML("<b>hello</b>")
	if got != "hello" {
		t.Errorf("stripHTML(<b>hello</b>) = %q, want %q", got, "hello")
	}
}

func TestStripHTMLEntities(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"&amp;", "&"},
		{"&lt;", "<"},
		{"&gt;", ">"},
		{"a&nbsp;b", "a b"},
		{"&quot;", `"`},
		{"a&amp;b&lt;c", "a&b<c"},
	}
	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripHTMLNested(t *testing.T) {
	got := stripHTML("<div><p>text</p></div>")
	if got != "text" {
		t.Errorf("stripHTML = %q, want %q", got, "text")
	}
}

func TestStripHTMLNoTags(t *testing.T) {
	got := stripHTML("plain text")
	if got != "plain text" {
		t.Errorf("stripHTML(%q) = %q, want passthrough", "plain text", got)
	}
}

func TestStripHTMLEmpty(t *testing.T) {
	got := stripHTML("")
	if got != "" {
		t.Errorf("stripHTML(\"\") = %q, want empty", got)
	}
}

func TestStripHTMLMultipleEntities(t *testing.T) {
	got := stripHTML("<p>a &amp; b &lt; c&nbsp;&gt; d</p>")
	if got != "a & b < c > d" {
		t.Errorf("stripHTML = %q, want %q", got, "a & b < c > d")
	}
}
