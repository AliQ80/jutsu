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

	t.Run("ANSI-prefixed lines pass through untouched", func(t *testing.T) {
		in := "\x1b[1mHeader   With   Spacing\x1b[0m"
		if got := wordWrap(in, 10); got != in {
			t.Errorf("got %q, want %q (unchanged)", got, in)
		}
	})
}
