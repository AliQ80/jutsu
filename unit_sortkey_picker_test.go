package main

import (
    "testing"

    tea "charm.land/bubbletea/v2"
)

// Test activation, navigation, selection, confirm, and cancel of the SORT_KEY picker.
func TestSortKeyPickerFlow(t *testing.T) {
    m := newModel()
    // Focus on the "bookmark list" command which has a SORT_KEY flag "sort".
    focusCommand(t, &m, "bookmark", "list")
    // Select the sort flag so it appears in the inputs.
    selectFlag(t, &m, "--sort", "")

    // Find the index of the sort flag in the combined inputs.
    combined := m.getActiveCombined()
    sortIdx := -1
    for i, item := range combined {
        if item.Name == "sort" && !item.IsArg {
            sortIdx = i
            break
        }
    }
    if sortIdx == -1 {
        t.Fatalf("sort flag not found in combined inputs")
    }
    m.focusPane = focusInputs
    m.focusedInputIdx = sortIdx

    // Activate picker with ';'
    next, _ := m.handleInputKeys(tea.KeyPressMsg{Code: ';', Text: ";"})
    m = next.(mainModel)
    if !m.sortKeyPickerOpen {
        t.Fatalf("picker did not open after ';' key")
    }
    if m.pickerProvider == nil {
        t.Fatalf("pickerProvider not set")
    }
    if len(m.pickerAllEntries) == 0 {
        t.Fatalf("pickerAllEntries empty")
    }
    // Initial focused index should be 0.
    if m.pickerFocusedIdx != 0 {
        t.Fatalf("expected pickerFocusedIdx 0, got %d", m.pickerFocusedIdx)
    }

    // Navigate down.
    next, _ = m.handleInputKeys(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
    m = next.(mainModel)
    if m.pickerFocusedIdx != 1 {
        t.Fatalf("expected pickerFocusedIdx 1 after down, got %d", m.pickerFocusedIdx)
    }

    // Select the currently focused entry with space.
    entry := m.pickerAllEntries[m.pickerFocusedIdx]
    next, _ = m.handleInputKeys(tea.KeyPressMsg{Code: tea.KeySpace, Text: "space"})
    m = next.(mainModel)
    dir, ok := m.pickerSelection[entry.Canonical]
    if !ok || dir != DirAscending {
        t.Fatalf("expected entry %s to be selected ascending, got %v", entry.Canonical, dir)
    }
    // Confirm selection with Enter.
    next, _ = m.handleInputKeys(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
    m = next.(mainModel)
    if m.sortKeyPickerOpen {
        t.Fatalf("picker should be closed after Enter")
    }
    // The input value should now contain the selected key.
    ti, ok := m.inputs["sort"]
    if !ok {
        t.Fatalf("sort input not found after confirming picker")
    }
    if got := ti.Value(); got != entry.Canonical {
        t.Fatalf("expected input value %q, got %q", entry.Canonical, got)
    }

    // Re-open picker and cancel with Esc.
    // Activate again.
    next, _ = m.handleInputKeys(tea.KeyPressMsg{Code: ';', Text: ";"})
    m = next.(mainModel)
    // Cancel.
    next, _ = m.handleInputKeys(tea.KeyPressMsg{Code: tea.KeyEsc, Text: "esc"})
    m = next.(mainModel)
    if m.sortKeyPickerOpen {
        t.Fatalf("picker should be closed after Esc")
    }
    // Value should remain unchanged (still the previously set value).
    ti, _ = m.inputs["sort"]
    if got := ti.Value(); got != entry.Canonical {
        t.Fatalf("input value changed after cancel, got %q, want %q", got, entry.Canonical)
    }
}
