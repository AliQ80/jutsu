package main

import (
	"strings"
	"testing"
)

func TestWordWrap(t *testing.T) {
	t.Run("fits within width: whitespace still normalized to single spaces", func(t *testing.T) {
		in := "L    L' | K (split)"
		want := "L L' | K (split)"
		if got := wordWrap(in, 80); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("exceeds width: whitespace normalized and wrapped, no large gaps", func(t *testing.T) {
		in := "L                 L' |                 | K (split)   =>    K\" (remaining)"
		got := wordWrap(in, 20)
		if strings.Contains(got, "  ") {
			t.Errorf("multi-space run survived normalization: %q", got)
		}
		for _, line := range strings.Split(got, "\n") {
			if len([]rune(line)) > 20 {
				t.Errorf("line exceeds width 20: %q", line)
			}
		}
	})

	t.Run("plain prose still wraps at word boundaries", func(t *testing.T) {
		in := "the quick brown fox jumps over the lazy dog"
		got := wordWrap(in, 10)
		for _, line := range strings.Split(got, "\n") {
			if len([]rune(line)) > 10 {
				t.Errorf("line exceeds width 10: %q", line)
			}
		}
	})

	t.Run("indented lines are preformatted: pass through verbatim", func(t *testing.T) {
		in := "  B   C\n   \\ /\n    @"
		if got := wordWrap(in, 80); got != in {
			t.Errorf("got %q, want %q (unchanged)", got, in)
		}
	})

	t.Run("over-width preformatted line truncates instead of wrapping", func(t *testing.T) {
		in := "  B   C   =>    @   with   trailing   overflow"
		want := "  B   C   ="
		if got := wordWrap(in, 11); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ANSI-prefixed lines pass through untouched", func(t *testing.T) {
		in := "\x1b[1mHeader   With   Spacing\x1b[0m"
		if got := wordWrap(in, 10); got != in {
			t.Errorf("got %q, want %q (unchanged)", got, in)
		}
	})

	t.Run("indent-marked lines wrap narrower and every line is indented", func(t *testing.T) {
		in := indentMarker + "the quick brown fox jumps"
		want := "  the quick\n  brown fox\n  jumps"
		if got := wordWrap(in, 12); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("indentBlock marks prose lines but leaves blank and preformatted lines untouched", func(t *testing.T) {
		in := "para one\n\n  B   C\n   \\ /\n    @\n\npara two"
		want := indentMarker + "para one\n\n  B   C\n   \\ /\n    @\n\n" + indentMarker + "para two"
		if got := indentBlock(in); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
