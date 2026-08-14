package main

import "testing"

// Covers the flag spellings (long / bar / exec) and the docs label across
// the interesting alias and alternate-spelling combinations. Value is the
// primary deterministic execution spelling: bar and exec both emit it, and
// every alternate form (Alias, Alternates) is docs-only.
func TestFlagForms(t *testing.T) {
	tests := []struct {
		name string
		f    Flag
		long string
		bar  string
		exec string
		docs string
	}{
		{
			name: "short value with shorter alias",
			f:    Flag{Name: "insert-after", Value: "-A", Alias: "after", InputType: "REVSETS"},
			long: "--insert-after",
			bar:  "-A",
			exec: "-A",
			docs: "-A, --insert-after <REVSETS> [aliases: --after]",
		},
		{
			name: "short value with longer alias",
			f:    Flag{Name: "onto", Value: "-o", Alias: "destination", InputType: "REVSETS"},
			long: "--onto",
			bar:  "-o",
			exec: "-o",
			docs: "-o, --onto <REVSETS> [aliases: --destination]",
		},
		{
			name: "long-only value with shorter alias",
			f:    Flag{Name: "operation", Value: "--operation", Alias: "op", InputType: "OPERATION"},
			long: "--operation",
			bar:  "--operation",
			exec: "--operation",
			docs: "--operation <OPERATION> [aliases: --op]",
		},
		{
			name: "short value without alias",
			f:    Flag{Name: "message", Value: "-m", InputType: "MESSAGE"},
			long: "--message",
			bar:  "-m",
			exec: "-m",
			docs: "-m, --message <MESSAGE>",
		},
		{
			name: "long-only value without alias or input",
			f:    Flag{Name: "no-edit", Value: "--no-edit"},
			long: "--no-edit",
			bar:  "--no-edit",
			exec: "--no-edit",
			docs: "--no-edit",
		},
		{
			name: "alternate short spelling is docs-only",
			f:    Flag{Name: "onto", Value: "-o", Alias: "destination", Alternates: []string{"-d"}, InputType: "REVSETS"},
			long: "--onto",
			bar:  "-o",
			exec: "-o",
			docs: "-o, --onto <REVSETS> [aliases: --destination, -d]",
		},
		{
			name: "multiple alternates including long form",
			f:    Flag{Name: "root", Value: "--root", Alternates: []string{"-d"}, InputType: "PATH"},
			long: "--root",
			bar:  "--root",
			exec: "--root",
			docs: "--root <PATH> [aliases: -d]",
		},
		{
			name: "short-only value without long spelling",
			f:    Flag{Name: "exec", Value: "-x", ShortOnly: true},
			long: "-x",
			bar:  "-x",
			exec: "-x",
			docs: "-x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flagLongForm(tt.f); got != tt.long {
				t.Errorf("flagLongForm = %q, want %q", got, tt.long)
			}
			if got := flagBarForm(tt.f); got != tt.bar {
				t.Errorf("flagBarForm = %q, want %q", got, tt.bar)
			}
			if got := flagExecForm(tt.f); got != tt.exec {
				t.Errorf("flagExecForm = %q, want %q", got, tt.exec)
			}
			if got := flagDocsLabel(tt.f); got != tt.docs {
				t.Errorf("flagDocsLabel = %q, want %q", got, tt.docs)
			}
		})
	}
}
