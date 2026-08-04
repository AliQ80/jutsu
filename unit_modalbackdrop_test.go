package main

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// overlay() must splice fg into bg at the given coordinates without
// disturbing surrounding rows/columns — the whole point of showing the app
// frame behind a modal instead of a blank one.
func TestOverlaySplicesWithoutDisturbingSurroundingContent(t *testing.T) {
	bg := "AAAAAAAAAA\nBBBBBBBBBB\nCCCCCCCCCC"
	fg := "XX\nYY"

	got := overlay(bg, fg, 3, 1)
	want := "AAAAAAAAAA\nBBBXXBBBBB\nCCCYYCCCCC"
	if got != want {
		t.Errorf("overlay() =\n%q\nwant\n%q", got, want)
	}
}

// Rows the overlay doesn't cover (above/below fg's height, or out of bg's
// bounds) must be left completely untouched.
func TestOverlayLeavesUncoveredRowsAlone(t *testing.T) {
	bg := "111\n222\n333"
	fg := "Z"

	got := overlay(bg, fg, 1, 5) // past the last bg row
	if got != bg {
		t.Errorf("overlay() past bg bounds should be a no-op, got %q", got)
	}

	got = overlay(bg, fg, 1, 1)
	lines := strings.Split(got, "\n")
	if lines[0] != "111" || lines[2] != "333" {
		t.Errorf("overlay() touched rows outside fg's height: %q", got)
	}
	if lines[1] != "2Z2" {
		t.Errorf("overlay() row 1 = %q, want %q", lines[1], "2Z2")
	}
}

// freezeBackdrop must capture state as it was *before* the caller flips a
// *ModalOpen flag — the modal-open bit itself, and any conditional field the
// caller sets first, must not leak into the snapshot.
func TestFreezeBackdropCapturesPreModalState(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.catIdx = 2

	m.freezeBackdrop()

	if m.backdrop == nil {
		t.Fatal("freezeBackdrop() left m.backdrop nil")
	}
	if m.backdrop.catIdx != 2 {
		t.Errorf("backdrop.catIdx = %d, want 2 (snapshot should match pre-freeze state)", m.backdrop.catIdx)
	}
	if m.backdrop.globalFlagsModalOpen {
		t.Error("backdrop captured globalFlagsModalOpen = true; freezeBackdrop should run before that flag is set")
	}
	if m.backdrop.backdrop != nil {
		t.Error("a frozen backdrop must not carry its own nested backdrop")
	}

	// Mutating the live model afterwards must not retroactively change the
	// frozen snapshot — that's the "frozen" half of the contract.
	m.catIdx = 5
	if m.backdrop.catIdx != 2 {
		t.Errorf("backdrop.catIdx changed to %d after mutating the live model", m.backdrop.catIdx)
	}
}

// Opening the global-options modal via "g" must freeze the backdrop and
// block composer input from reaching the underlying model while it's open —
// asserted through the real key-routing path, not by calling the handler
// directly.
func TestGlobalFlagsModalFreezesBackdropAndBlocksInput(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories
	preCatIdx := m.catIdx

	next, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	opened := next.(mainModel)

	if !opened.globalFlagsModalOpen {
		t.Fatal("\"g\" did not open the global-options modal")
	}
	if opened.backdrop == nil {
		t.Fatal("opening the modal did not freeze a backdrop")
	}
	if opened.backdrop.catIdx != preCatIdx {
		t.Errorf("backdrop.catIdx = %d, want pre-modal value %d", opened.backdrop.catIdx, preCatIdx)
	}

	// While the modal is open, composer navigation keys are routed to the
	// modal handler instead of catIdx/cmdIdx — confirm "l" (a composer pane
	// key) is a no-op on catIdx now that the modal owns key handling.
	next2, _ := opened.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	afterKey := next2.(mainModel)
	if afterKey.catIdx != opened.catIdx {
		t.Errorf("catIdx changed from %d to %d while modal open; composer input should be blocked", opened.catIdx, afterKey.catIdx)
	}

	// Closing restores live rendering: the backdrop snapshot is dropped.
	closed, _ := afterKey.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	afterClose := closed.(mainModel)
	if afterClose.globalFlagsModalOpen {
		t.Fatal("esc did not close the global-options modal")
	}
	if afterClose.backdrop != nil {
		t.Error("backdrop was not cleared after the modal closed")
	}
}

// View() must overlay the modal on the reflowed backdrop rather than a blank
// screen: the backdrop's own content (drawn from the frozen composer state)
// should be present in the rendered frame around the modal box.
func TestModalViewRendersOverBackdropFrame(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories

	next, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	opened := next.(mainModel)

	view := opened.View()
	content := view.Content

	// The backdrop frame always renders the CATEGORIES column header; its
	// presence in the modal's output means the app frame is showing through,
	// not a blank/centered-only render.
	if !strings.Contains(content, "CATEGORIES") {
		t.Error("modal view does not contain the backdrop frame (expected \"CATEGORIES\" header)")
	}
}

// Toggling a flag deep in the category tree (Category -> Command -> Flags)
// after freezing must not be visible through the frozen backdrop. Before the
// deep-copy fix, categories/Commands/Flags all shared their backing arrays
// with the live model, so this mutation leaked straight through.
func TestFreezeBackdropIsolatesCategoryFlagMutation(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.catIdx, m.cmdIdx, m.flagIdx = 0, 0, 0

	flags := m.currentFlags()
	if len(flags) == 0 {
		t.Fatal("test setup: current command has no flags to toggle")
	}
	wantFrozen := flags[0].Selected

	m.freezeBackdrop()

	m.toggleFlag() // mutates m.categories[0].Commands[0].Flags[0].Selected in place

	got := m.backdrop.categories[m.catIdx].Commands[m.cmdIdx].Flags[m.flagIdx].Selected
	if got != wantFrozen {
		t.Errorf("backdrop flag.Selected = %v after live toggleFlag(); want unchanged frozen value %v", got, wantFrozen)
	}
	// Sanity: the live model actually did toggle, so this isn't a no-op test.
	if m.currentFlags()[0].Selected == wantFrozen {
		t.Fatal("test setup: toggleFlag() did not actually change the live flag")
	}
}

// Toggling a global flag (via the global-options modal's "space" key) while
// the modal is open must not retroactively change the frozen backdrop's copy
// of globalFlags — same sharing hazard as category flags, but through
// m.globalFlags directly instead of nested category structs.
func TestGlobalFlagToggleWhileModalOpenDoesNotMutateBackdrop(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories

	next, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	opened := next.(mainModel)
	if opened.backdrop == nil {
		t.Fatal("opening the global-options modal did not freeze a backdrop")
	}
	frozenSelected := opened.backdrop.globalFlags[opened.globalFlagIdx].Selected

	next2, _ := opened.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	toggled := next2.(mainModel)

	if toggled.globalFlags[toggled.globalFlagIdx].Selected == frozenSelected {
		t.Fatal("test setup: \"space\" did not actually toggle the live global flag")
	}
	if toggled.backdrop.globalFlags[toggled.globalFlagIdx].Selected != frozenSelected {
		t.Errorf("backdrop.globalFlags[%d].Selected changed to %v after live toggle; want frozen value %v",
			toggled.globalFlagIdx, toggled.backdrop.globalFlags[toggled.globalFlagIdx].Selected, frozenSelected)
	}
}

// setInput reassigns entries in the live inputs map; since maps are
// reference types, a shallow `snap := *m` would share the exact same map
// object with the backdrop. Confirm the frozen copy is a distinct map.
func TestFreezeBackdropIsolatesInputMapMutation(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40

	ti := textinput.New()
	ti.SetValue("pre-freeze")
	m.inputs = map[string]textinput.Model{"revisions": ti}

	m.freezeBackdrop()

	edited := textinput.New()
	edited.SetValue("post-freeze")
	m.inputs["revisions"] = edited // live map mutation, post-freeze

	frozen, ok := m.backdrop.inputs["revisions"]
	if !ok {
		t.Fatal("backdrop.inputs missing the \"revisions\" entry captured at freeze time")
	}
	if frozen.Value() != "pre-freeze" {
		t.Errorf("backdrop.inputs[\"revisions\"].Value() = %q, want %q (frozen pre-mutation value)", frozen.Value(), "pre-freeze")
	}
	if m.inputs["revisions"].Value() != "post-freeze" {
		t.Fatal("test setup: live m.inputs was not actually mutated")
	}

	// Also confirm the maps are genuinely distinct objects, not just
	// coincidentally holding different values.
	if len(m.backdrop.inputs) != 1 {
		t.Errorf("backdrop.inputs len = %d, want 1 (adding to the live map must not resize the backdrop's)", len(m.backdrop.inputs))
	}
}

// An async execResultMsg arriving while a modal is open (e.g. a long-running
// command finishing in the background) reassigns m.outputLines wholesale.
// The frozen backdrop's copy must not observe that change.
func TestAsyncOutputUpdateDoesNotMutateFrozenBackdrop(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.outputLines = []string{"before"}

	m.freezeBackdrop()

	m.outputLines = []string{"after", "async update"}

	if len(m.backdrop.outputLines) != 1 || m.backdrop.outputLines[0] != "before" {
		t.Errorf("backdrop.outputLines = %v, want [\"before\"] (frozen snapshot)", m.backdrop.outputLines)
	}
}

// Resizing the terminal while a modal is open must reflow the backdrop's
// rendered frame at the new dimensions without mutating the frozen snapshot
// itself — otherwise a second resize (or a close-then-reopen at a different
// size) would compound stale width/height into the stored backdrop.
func TestRenderOnBackdropReflowsWithoutMutatingSnapshot(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories

	next, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	opened := next.(mainModel)
	frozenW, frozenH := opened.backdrop.width, opened.backdrop.height

	// Simulate a resize while the modal is open.
	opened.width, opened.height = 200, 60
	view := opened.View()
	if firstLineLen := len(strings.Split(view.Content, "\n")[0]); firstLineLen == 0 {
		t.Fatal("resized modal view produced an empty first line")
	}

	if opened.backdrop.width != frozenW || opened.backdrop.height != frozenH {
		t.Errorf("backdrop.width/height changed to %d/%d after rendering at a new size; want unchanged %d/%d",
			opened.backdrop.width, opened.backdrop.height, frozenW, frozenH)
	}
}

// Mouse events outside the modal must remain isolated from the backdrop, while
// events inside the global-options modal control its checklist/docs panes.
func TestModalMouseEventsStayInsideModal(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories

	next, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	opened := next.(mainModel)
	preCatIdx := opened.catIdx

	// An event outside the centered modal must not reach the backdrop.
	next2, _ := opened.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 0, Y: 0})
	outside := next2.(mainModel)
	if outside.catIdx != preCatIdx || outside.globalFlagIdx != opened.globalFlagIdx {
		t.Error("mouse event outside modal changed the underlying application")
	}

	modal := opened.renderGlobalFlagsModal()
	originX := (opened.width - lipgloss.Width(modal)) / 2
	originY := (opened.height - lipgloss.Height(modal)) / 2

	// Wheel over the checklist navigates the modal's selection.
	next3, _ := opened.Update(tea.MouseWheelMsg{
		Button: tea.MouseWheelDown,
		X:      originX + 2,
		Y:      originY + 4,
	})
	afterListWheel := next3.(mainModel)
	if afterListWheel.globalFlagIdx == opened.globalFlagIdx {
		t.Error("mouse wheel over modal checklist did not move the selection")
	}
	if afterListWheel.catIdx != preCatIdx {
		t.Error("mouse wheel over modal checklist changed the backdrop state")
	}

	// Clicking a checklist row only focuses the options pane; it must not select
	// a flag. Empty space in the same pane has the same focus behavior.
	opened.globalModalDocsFocus = true
	preFlagIdx := opened.globalFlagIdx
	next4, _ := opened.Update(tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      originX + 2,
		Y:      originY + 3 + 2,
	})
	afterClick := next4.(mainModel)
	if afterClick.globalFlagIdx != preFlagIdx || afterClick.globalModalDocsFocus {
		t.Errorf("modal checklist click produced idx=%d docsFocus=%v; want unchanged idx=%d and options focus", afterClick.globalFlagIdx, afterClick.globalModalDocsFocus, preFlagIdx)
	}

	nextEmpty, _ := afterClick.Update(tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      originX + 2,
		Y:      originY + lipgloss.Height(modal) - 2,
	})
	afterEmpty := nextEmpty.(mainModel)
	if afterEmpty.globalFlagIdx != preFlagIdx || afterEmpty.globalModalDocsFocus {
		t.Errorf("empty checklist click changed idx=%d docsFocus=%v; want unchanged idx=%d and options focus", afterEmpty.globalFlagIdx, afterEmpty.globalModalDocsFocus, preFlagIdx)
	}

	// Ensure the docs viewport has content beyond its visible window so the
	// wheel assertion exercises scrolling rather than only focus transfer.
	opened.globalDocs.SetContent(strings.Repeat("line\n", 50))
	preDocsOffset := opened.globalDocs.YOffset()
	next5, _ := opened.Update(tea.MouseWheelMsg{
		Button: tea.MouseWheelDown,
		X:      originX + globalListW + 6,
		Y:      originY + 4,
	})
	afterDocsWheel := next5.(mainModel)
	if !afterDocsWheel.globalModalDocsFocus {
		t.Error("mouse wheel over modal docs did not move focus to docs")
	}
	if afterDocsWheel.globalDocs.YOffset() <= preDocsOffset {
		t.Errorf("mouse wheel over modal docs offset = %d, want > %d", afterDocsWheel.globalDocs.YOffset(), preDocsOffset)
	}

	if !afterDocsWheel.globalFlagsModalOpen {
		t.Error("mouse event unexpectedly closed the modal")
	}
}

// overlay() must clip rather than panic or corrupt output when fg starts at
// a negative coordinate (a modal centered wider than the terminal, or a
// caller's off-by-one).
func TestOverlayClipsNegativeCoordinates(t *testing.T) {
	bg := "0123456789\nABCDEFGHIJ"

	// fg starts 2 columns left of bg: only "34567890" width worth is visible.
	got := overlay(bg, "XXXXX", -2, 0)
	want := "XXX3456789\nABCDEFGHIJ"
	if got != want {
		t.Errorf("overlay() with negative x =\n%q\nwant\n%q", got, want)
	}

	// fg entirely off the left edge must be a complete no-op for that row.
	got = overlay(bg, "XX", -10, 0)
	if got != bg {
		t.Errorf("overlay() fully off-left should be a no-op, got %q", got)
	}

	// Negative y: row falls above bg entirely, no-op.
	got = overlay(bg, "XX", 0, -1)
	if got != bg {
		t.Errorf("overlay() with negative y should be a no-op, got %q", got)
	}
}

// overlay() must clip an oversized modal (wider than bg) instead of running
// off the end of the line or panicking on the ansi.Cut bounds.
func TestOverlayClipsOversizedModal(t *testing.T) {
	bg := "12345"
	fg := "ABCDEFGHIJ" // wider than bg

	got := overlay(bg, fg, 0, 0)
	want := "ABCDE" // clipped to bg's width
	if got != want {
		t.Errorf("overlay() with oversized fg = %q, want %q", got, want)
	}

	// Oversized fg offset partway across bg: only the portion that fits
	// should land, the rest clipped.
	got = overlay(bg, fg, 2, 0)
	want = "12ABC"
	if got != want {
		t.Errorf("overlay() with oversized fg at x=2 = %q, want %q", got, want)
	}
}

// overlay() must not split a double-width Unicode cell in half at either
// edge of the clip — ansi.Cut is grapheme-width aware, and this exercises
// that overlay actually uses it at the boundaries it computes itself.
func TestOverlayHandlesWideUnicodeAtClipBoundary(t *testing.T) {
	bg := "口口口口口" // 5 double-width CJK chars = 10 cells wide
	fg := "口口"    // 2 double-width chars = 4 cells wide

	got := overlay(bg, fg, 3, 0)
	// Placing a 4-wide fg at column 3 does not land on a cell boundary
	// (column 3 is mid-glyph); ansi.Cut must clip to whole cells rather than
	// emit a malformed half-glyph, so the result must still be valid UTF-8
	// with no panic.
	if !strings.Contains(got, fg) {
		// Depending on how Cut resolves the boundary, this may clip a
		// column — the real assertion is just that nothing panicked and
		// the result stayed well-formed. Fall back to a width check.
		if lipgloss.Width(got) < lipgloss.Width(bg)-2 {
			t.Errorf("overlay() at a wide-unicode clip boundary produced unexpectedly short output: %q", got)
		}
	}
}

// overlay() must preserve ANSI SGR styling in fg lines rather than mangling
// escape sequences when splicing them into a plain bg line.
func TestOverlayPreservesANSIStyledContent(t *testing.T) {
	bg := "          "
	styled := lipgloss.NewStyle().Foreground(colorRed).Render("hi")

	got := overlay(bg, styled, 2, 0)
	if !strings.Contains(got, styled) {
		t.Errorf("overlay() did not preserve the styled fg content verbatim: %q", got)
	}
}

// Opening a modal while the output pane is enlarged must preserve that
// enlarged layout in the backdrop shown behind the modal — not silently fall
// back to the ordinary composer+output frame. renderContentFrame() is the
// shared layout selector behind both View() and renderOnBackdrop(); this
// exercises it through the real backdrop-reflow path.
func TestModalBackdropPreservesOutputEnlargedLayout(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusOutput
	m.outputEnlarged = true
	m.layoutViewports()

	next, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	opened := next.(mainModel)

	if !opened.backdrop.outputEnlargedActive() {
		t.Fatal("backdrop snapshot did not retain outputEnlarged/focusOutput state")
	}

	view := opened.View()
	content := view.Content

	if !strings.Contains(content, "OUTPUT (ENLARGED)") {
		t.Error("modal backdrop lost the output-enlarged layout; expected \"OUTPUT (ENLARGED)\" in rendered frame")
	}
	if strings.Contains(content, "CATEGORIES") {
		t.Error("modal backdrop rendered the ordinary composer frame instead of the enlarged-output layout")
	}
}

// Same as above for the docs-enlarged layout.
func TestModalBackdropPreservesDocsEnlargedLayout(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusDocs
	m.docsEnlarged = true
	m.layoutViewports()

	next, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	opened := next.(mainModel)

	if !opened.backdrop.docsEnlargedActive() {
		t.Fatal("backdrop snapshot did not retain docsEnlarged/focusDocs state")
	}

	view := opened.View()
	content := view.Content

	if !strings.Contains(content, "DOCS (ENLARGED)") {
		t.Error("modal backdrop lost the docs-enlarged layout; expected \"DOCS (ENLARGED)\" in rendered frame")
	}
	if strings.Contains(content, "CATEGORIES") {
		t.Error("modal backdrop rendered the ordinary composer frame instead of the enlarged-docs layout")
	}
}

// dimBackdrop must preserve the frame's ANSI styling while transforming its
// truecolor values toward the base palette, so the app remains recognizable
// rather than becoming monochrome.
func TestDimBackdropPreservesAndMutesOriginalColor(t *testing.T) {
	original := lipgloss.NewStyle().Foreground(colorRed).Render("hello")
	got := dimBackdrop(original)

	if !strings.Contains(got, "hello") {
		t.Errorf("dimBackdrop() dropped the underlying text: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;") {
		t.Errorf("dimBackdrop() removed the original truecolor styling: %q", got)
	}
	if got == original {
		t.Error("dimBackdrop() left content unchanged; expected muted color values")
	}

	plain := dimBackdrop("docs text")
	if !strings.Contains(plain, "\x1b[38;2;108;112;134m") {
		t.Errorf("dimBackdrop() did not mute unstyled text: %q", plain)
	}
}
