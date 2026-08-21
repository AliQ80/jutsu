package main

import "testing"

func TestSortKeyParseAndFormat(t *testing.T) {
	provider := GetChoiceProvider("SORT_KEY")
	if provider == nil {
		t.Fatalf("SORT_KEY provider not registered")
	}

	raw := "author-date,name-"
	sels := provider.Parse(raw)
	if len(sels) != 2 {
		t.Fatalf("expected 2 selections, got %d", len(sels))
	}
	if sels[0].Entry.Canonical != "author-date" || sels[0].Direction != DirAscending {
		t.Errorf("first selection mismatch: got %v %v", sels[0].Entry.Canonical, sels[0].Direction)
	}
	if sels[1].Entry.Canonical != "name" || sels[1].Direction != DirDescending {
		t.Errorf("second selection mismatch: got %v %v", sels[1].Entry.Canonical, sels[1].Direction)
	}

	// Formatting should round‑trip to the same raw string (order preserved).
	formatted := provider.Format(sels)
	if formatted != raw {
		t.Errorf("format round‑trip mismatch: got %q, want %q", formatted, raw)
	}

	// Ensure order is preserved when parsing a different order.
	raw2 := "name,author-date"
	sels2 := provider.Parse(raw2)
	if len(sels2) != 2 {
		t.Fatalf("expected 2 selections for raw2, got %d", len(sels2))
	}
	if sels2[0].Entry.Canonical != "name" || sels2[0].Direction != DirAscending {
		t.Errorf("raw2 first selection mismatch: got %v %v", sels2[0].Entry.Canonical, sels2[0].Direction)
	}
	if sels2[1].Entry.Canonical != "author-date" || sels2[1].Direction != DirAscending {
		t.Errorf("raw2 second selection mismatch: got %v %v", sels2[1].Entry.Canonical, sels2[1].Direction)
	}
	if provider.Format(sels2) != raw2 {
		t.Errorf("format round‑trip mismatch for raw2: got %q, want %q", provider.Format(sels2), raw2)
	}
}
