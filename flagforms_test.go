package main

import "testing"

// Covers the three flag spellings (long / bar / exec) and the docs label
// across the interesting alias combinations.
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
			bar:  "--after",
			exec: "-A",
			docs: "-A, --insert-after <REVSETS> [aliases: --after]",
		},
		{
			name: "short value with longer alias",
			f:    Flag{Name: "onto", Value: "-o", Alias: "destination", InputType: "REVSETS"},
			long: "--onto",
			bar:  "--onto",
			exec: "-o",
			docs: "-o, --onto <REVSETS> [aliases: --destination]",
		},
		{
			name: "long-only value with shorter alias",
			f:    Flag{Name: "operation", Value: "--operation", Alias: "op", InputType: "OPERATION"},
			long: "--operation",
			bar:  "--op",
			exec: "--op",
			docs: "--operation <OPERATION> [aliases: --op]",
		},
		{
			name: "short value without alias",
			f:    Flag{Name: "message", Value: "-m", InputType: "MESSAGE"},
			long: "--message",
			bar:  "--message",
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
