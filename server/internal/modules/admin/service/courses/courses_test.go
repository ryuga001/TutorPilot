package courses

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"lowercases and hyphenates", "Intro To Go", "intro-to-go"},
		{"collapses punctuation", "Go: The  Basics!", "go-the-basics"},
		{"trims leading and trailing separators", "  --Go--  ", "go"},
		{"keeps digits", "Algebra 101", "algebra-101"},
		{"falls back when nothing survives", "!!!", "course"},
		{"falls back on empty input", "", "course"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slugify(tt.in); got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSlugifyCapsLength(t *testing.T) {
	got := slugify(strings.Repeat("a", 500))
	if len(got) > 200 {
		t.Errorf("slug length = %d, want <= 200", len(got))
	}
}

func TestSlugifyNeverReturnsEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "---", "!!!", "\t\n"} {
		if got := slugify(in); got == "" {
			t.Errorf("slugify(%q) returned empty; the slug column is NOT NULL", in)
		}
	}
}

func TestSanitizeNameStripsPathTraversal(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"unix traversal", "../../etc/passwd", "passwd"},
		{"windows traversal", `..\..\windows\system32`, "system32"},
		{"absolute path", "/etc/hosts", "hosts"},
		{"spaces become hyphens", "my notes.pdf", "my-notes.pdf"},
		{"plain name untouched", "slides.pdf", "slides.pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeName(tt.in); got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeNameNeverYieldsASeparator(t *testing.T) {
	for _, in := range []string{"", ".", "/", "..", "///"} {
		got := sanitizeName(in)
		if strings.Contains(got, "/") || got == "" || got == "." {
			t.Errorf("sanitizeName(%q) = %q, which would corrupt the object key", in, got)
		}
	}
}

func TestRandTokenIsHexAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok := randToken()
		if len(tok) != 16 {
			t.Fatalf("token %q has length %d, want 16 hex chars", tok, len(tok))
		}
		if strings.Trim(tok, "0123456789abcdef") != "" {
			t.Fatalf("token %q is not hex", tok)
		}
		if seen[tok] {
			t.Fatalf("duplicate token %q after %d draws", tok, i)
		}
		seen[tok] = true
	}
}
