package main

import "testing"

func TestConvergeInteractiveInvocation(t *testing.T) {
	t.Run("interactive by default", func(t *testing.T) {
		m := newModel()
		focusCommand(t, &m, "converge", "")
		if got := m.isInteractiveInvocation(); !got {
			t.Errorf("isInteractiveInvocation() = %v, want true", got)
		}
	})

	t.Run("no-interactive opts out", func(t *testing.T) {
		m := newModel()
		focusCommand(t, &m, "converge", "")
		selectFlag(t, &m, "--no-interactive", "")
		if got := m.isInteractiveInvocation(); got {
			t.Errorf("isInteractiveInvocation() = %v, want false", got)
		}
	})
}
