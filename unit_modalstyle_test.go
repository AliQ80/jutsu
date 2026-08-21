package main

import (
	"strings"
	"testing"
)

// The shared modalBoxStyle must not paint a fixed background: About, auth,
// and SSH host-key modals all render through it, and all three must inherit
// the terminal's default background instead of the old hard-coded #1e1e2e
// (rendered as \x1b[48;2;30;30;46m) — while keeping their rounded border and
// content. Other background escapes (e.g. the help-key chips' surface2) are
// expected and must not trip the assertion.
func TestSharedModalStyleHasNoFixedBackground(t *testing.T) {
	m := newModel()
	m.hostKeyPrompt = hostKeyFetchedMsg{
		host:        "example.com:22",
		fingerprint: "SHA256:abc123def456",
	}

	cases := []struct {
		name string
		got  string
		want string // distinctive content substring
	}{
		{"about", m.renderAboutModal(), "https://aliq80.github.io/jutsu/"},
		{"auth", m.renderAuthModal(), "Remote needs input"},
		{"host-key", m.renderHostKeyModal(), "Unknown SSH host"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Contains(tc.got, "\x1b[48;2;30;30;46m") {
				t.Errorf("modal renders the fixed #1e1e2e background; want terminal default:\n%s", tc.got)
			}
			if !strings.Contains(tc.got, tc.want) {
				t.Errorf("modal lost its content (missing %q):\n%s", tc.want, tc.got)
			}
			for _, b := range []string{"╭", "╮", "╰", "╯"} {
				if !strings.Contains(tc.got, b) {
					t.Errorf("modal lost its rounded border (%q missing):\n%s", b, tc.got)
				}
			}
		})
	}
}
