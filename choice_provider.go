package main

import (
	"strings"
)

// Direction represents the three-state cycle for a choice entry.
type Direction int

const (
	DirUnselected Direction = iota // not selected
	DirAscending                   // ascending order
	DirDescending                  // descending order (trailing '-')
)

// ChoiceEntry describes a selectable option for an input type.
type ChoiceEntry struct {
	// Canonical is the value used in the jj command (e.g., "author-date").
	Canonical string
	// Label is a human‑readable description shown in the picker.
	Label string
}

// ChoiceSelection represents a user's selection of a ChoiceEntry together with its direction.
type ChoiceSelection struct {
	Entry     *ChoiceEntry
	Direction Direction
}

// ChoiceProvider holds metadata and helper functions for a particular input type.
type ChoiceProvider struct {
	InputType string
	// Entries maps canonical values to their metadata.
	Entries map[string]*ChoiceEntry
	// Parse converts a raw input string into ordered selections.
	Parse func(string) []ChoiceSelection
	// Format converts ordered selections back to a raw input string.
	Format func([]ChoiceSelection) string
}

var choiceProviders = map[string]*ChoiceProvider{}

// RegisterChoiceProvider registers a provider for a given input type.
func RegisterChoiceProvider(p *ChoiceProvider) {
	choiceProviders[p.InputType] = p
}

// GetChoiceProvider returns the provider for the given input type, or nil.
func GetChoiceProvider(inputType string) *ChoiceProvider {
	return choiceProviders[inputType]
}

// init registers the SORT_KEY provider with known fields.
func init() {
	// Define known sort fields.
	entries := map[string]*ChoiceEntry{
		"name":            {Canonical: "name", Label: "Name"},
		"author-name":     {Canonical: "author-name", Label: "Author name"},
		"author-email":    {Canonical: "author-email", Label: "Author email"},
		"author-date":     {Canonical: "author-date", Label: "Author date"},
		"committer-name":  {Canonical: "committer-name", Label: "Committer name"},
		"committer-email": {Canonical: "committer-email", Label: "Committer email"},
		"committer-date":  {Canonical: "committer-date", Label: "Committer date"},
	}

	// Parsing logic for SORT_KEY values.
	parse := func(raw string) []ChoiceSelection {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		parts := strings.Split(raw, ",")
		var sels []ChoiceSelection
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			dir := DirAscending
			if strings.HasSuffix(p, "-") {
				dir = DirDescending
				p = strings.TrimSuffix(p, "-")
			}
			if entry, ok := entries[p]; ok {
				sels = append(sels, ChoiceSelection{Entry: entry, Direction: dir})
			}
		}
		return sels
	}

	// Formatting logic for SORT_KEY values.
	format := func(sels []ChoiceSelection) string {
		var parts []string
		for _, s := range sels {
			if s.Direction == DirUnselected {
				continue
			}
			val := s.Entry.Canonical
			if s.Direction == DirDescending {
				val += "-"
			}
			parts = append(parts, val)
		}
		return strings.Join(parts, ",")
	}

	RegisterChoiceProvider(&ChoiceProvider{
		InputType: "SORT_KEY",
		Entries:   entries,
		Parse:     parse,
		Format:    format,
	})
}
