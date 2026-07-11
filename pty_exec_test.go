//go:build !windows

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCapturedTrailingOutput(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		want   string
		wantOK bool
	}{
		{
			name:   "marker followed by plain text",
			raw:    "editor screen stuff" + string(altScreenExitSeq) + "Rebased 3 descendant commits\n",
			want:   "Rebased 3 descendant commits",
			wantOK: true,
		},
		{
			name:   "marker followed by ANSI-colored text",
			raw:    string(altScreenExitSeq) + "\x1b[32mRebased 3 commits\x1b[0m\n",
			want:   "Rebased 3 commits",
			wantOK: true,
		},
		{
			name:   "no marker",
			raw:    "just some editor screen bytes, no exit sequence",
			want:   "",
			wantOK: false,
		},
		{
			name:   "marker followed only by whitespace",
			raw:    string(altScreenExitSeq) + "  \n\n  ",
			want:   "",
			wantOK: false,
		},
		{
			name:   "multiple markers use only the last",
			raw:    string(altScreenExitSeq) + "stale text" + string(altScreenExitSeq) + "final text\n",
			want:   "final text",
			wantOK: true,
		},
		{
			// Ptys run in cooked mode, so a child's "\n" writes arrive here
			// as "\r\n" (ONLCR) — each recovered line must not carry a
			// trailing '\r'.
			name:   "CRLF line endings from cooked-mode pty are normalized",
			raw:    string(altScreenExitSeq) + "Rebased 3 descendant commits\r\nWorking copy now at abc123\r\n",
			want:   "Rebased 3 descendant commits\nWorking copy now at abc123",
			wantOK: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := capturedTrailingOutput([]byte(tc.raw))
			if ok != tc.wantOK || got != tc.want {
				t.Errorf("capturedTrailingOutput(%q) = (%q, %v), want (%q, %v)", tc.raw, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestBoundedWriter(t *testing.T) {
	var buf bytes.Buffer
	w := &boundedWriter{buf: &buf, max: 10}

	chunks := []string{"0123456789", "abcde", "FGHIJ"}
	for _, c := range chunks {
		if _, err := w.Write([]byte(c)); err != nil {
			t.Fatalf("Write(%q) error: %v", c, err)
		}
	}

	if buf.Len() > 10 {
		t.Fatalf("buffer grew past max: len=%d", buf.Len())
	}
	want := "FGHIJ"
	if got := buf.String(); !strings.HasSuffix(got, want) {
		t.Errorf("expected buffer to keep the tail %q, got %q", want, got)
	}
}
