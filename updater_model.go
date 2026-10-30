package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

// prepareReleaseUpdate is the preparation boundary shared by the UI and fixtures.
var prepareReleaseUpdate = prepareUpdate

type updateEvent struct {
	attempt     *updateAttempt
	phase       string
	download    *updateDownloadProgress
	transaction *updateTransaction
	err         error
	final       bool
}
type updateAttempt struct {
	cancel         context.CancelFunc
	events         chan updateEvent
	done           chan struct{}
	prepared       *updateTransaction
	closeRequested bool
	quitRequested  bool
}
type updateHandoffMsg struct {
	attempt     *updateAttempt
	transaction *updateTransaction
	err         error
}

func awaitUpdate(a *updateAttempt) tea.Cmd { return func() tea.Msg { return <-a.events } }
func (m mainModel) startUpdate() (tea.Model, tea.Cmd) {
	if m.updater != nil || m.latestReleaseVersion == "" {
		return m, nil
	}
	if m.running {
		m.updateStatus = "Wait for the running command to finish, then press u."
		return m, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &updateAttempt{cancel: cancel, events: make(chan updateEvent, 8), done: make(chan struct{})}
	tag := m.latestReleaseVersion
	m.updater = a
	m.updateStatus = "Downloading " + tag + "…"
	m.updateDownload = nil
	start := func() tea.Msg {
		defer close(a.done)
		executable, err := os.Executable()
		var t *updateTransaction
		if err == nil {
			source := defaultUpdateSource()
			source.progress = func(received, total int64) {
				// Byte updates are expendable; never block the download or cancellation
				// behind a slow renderer. Phase and final events remain ordered.
				select {
				case a.events <- updateEvent{attempt: a, download: &updateDownloadProgress{received: received, total: total}}:
				default:
				}
			}
			t, err = prepareReleaseUpdate(ctx, source, executable, tag, func(phase string) {
				select {
				case a.events <- updateEvent{attempt: a, phase: phase}:
				case <-ctx.Done():
				}
			})
		}
		a.prepared = t
		// A final transaction is owned by the model until commit or cancellation.
		a.events <- updateEvent{attempt: a, transaction: t, err: err, final: true}
		return nil
	}
	return m, tea.Batch(start, awaitUpdate(a))
}
func (m mainModel) handleUpdateEvent(msg updateEvent) (tea.Model, tea.Cmd) {
	if msg.attempt != m.updater {
		if msg.transaction != nil {
			msg.transaction.cleanup()
		}
		return m, nil
	}
	a := m.updater
	if !msg.final {
		if !a.closeRequested && !a.quitRequested {
			if msg.download != nil {
				m.updateDownload = msg.download
			} else {
				m.updateStatus = msg.phase
				m.updateDownload = nil
			}
		}
		return m, awaitUpdate(a)
	}
	a.cancel()
	m.updateDownload = nil
	if a.closeRequested || a.quitRequested {
		if msg.transaction != nil {
			msg.transaction.cleanup()
		}
		m.updater = nil
		m.updateStatus = ""
		m.aboutModalOpen = false
		m.backdrop = nil
		if a.quitRequested {
			return m, tea.Quit
		}
		return m, nil
	}
	if msg.err != nil {
		m.updater = nil
		m.updateStatus = fmt.Sprintf("Update failed: %v. Press u to retry.", msg.err)
		return m, nil
	}
	m.updateCommitting = true
	m.updateStatus = "Installing and restarting…"
	return m, func() tea.Msg {
		return updateHandoffMsg{attempt: a, transaction: msg.transaction, err: prepareUpdateHandoff(msg.transaction)}
	}
}
func (m mainModel) handleUpdateHandoff(msg updateHandoffMsg) (tea.Model, tea.Cmd) {
	if m.updater != msg.attempt {
		msg.transaction.cleanup()
		return m, nil
	}
	if msg.err != nil {
		msg.transaction.cleanup()
		m.updater = nil
		m.updateCommitting = false
		m.updateStatus = fmt.Sprintf("Cannot start update: %v. Press u to retry.", msg.err)
		return m, nil
	}
	m.updateRestart = msg.transaction
	return m, tea.Quit
}

// Stop preparation before leaving the process on a terminal/signal error. Waiting
// on done makes ownership of the worker's staged transaction unambiguous.
func (m mainModel) stopUpdatePreparation() {
	if m.updater == nil || m.updateCommitting {
		return
	}
	m.updater.cancel()
	if m.updater.done != nil {
		waiting := true
		for waiting {
			select {
			case <-m.updater.done:
				waiting = false
			case <-m.updater.events:
			}
		}
	}
	if m.updater.prepared != nil {
		m.updater.prepared.cleanup()
	}
}
