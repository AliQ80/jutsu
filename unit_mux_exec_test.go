package main

import "testing"

// TestTrimPaneTranscript covers the one piece of guesswork in the herdr
// handoff: reducing a scraped pane transcript (shell prompt echo, command
// output, sentinel, next prompt) to just the command's own output.
func TestTrimPaneTranscript(t *testing.T) {
	const (
		script   = "/tmp/jutsu-herdr-123/run.sh"
		sentinel = "__JUTSU_DONE__"
	)

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "typical transcript",
			raw:  "~/repo $ sh " + script + "\nWorking copy now at: abc123\nRebased 2 descendant commits\n" + sentinel + "\n~/repo $ ",
			want: "Working copy now at: abc123\nRebased 2 descendant commits",
		},
		{
			name: "no output between echo and sentinel",
			raw:  "~/repo $ sh " + script + "\n" + sentinel + "\n~/repo $ ",
			want: "",
		},
		{
			name: "prompt redrawn, only the last echo counts",
			raw:  "~/repo $ sh " + script + "\r~/repo $ sh " + script + "\nChanges pushed\n" + sentinel + "\n",
			want: "Changes pushed",
		},
		{
			name: "no sentinel (pane read raced the printf)",
			raw:  "~/repo $ sh " + script + "\nChanges pushed\n",
			want: "Changes pushed",
		},
		{
			name: "no echo line (scrolled out of the read window)",
			raw:  "Changes pushed\n" + sentinel + "\n",
			want: "Changes pushed",
		},
		{
			name: "echo line without a trailing newline yet",
			raw:  "~/repo $ sh " + script,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimPaneTranscript(tt.raw, script, sentinel); got != tt.want {
				t.Errorf("trimPaneTranscript() = %q, want %q", got, tt.want)
			}
		})
	}
}
