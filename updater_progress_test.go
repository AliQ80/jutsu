package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func TestUpdateDownloadByteProgress(t *testing.T) {
	for _, known := range []bool{true, false} {
		t.Run(fmt.Sprint(known), func(t *testing.T) {
			payload := strings.Repeat("x", 16<<10)
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if known {
					w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
				} else {
					w.(http.Flusher).Flush()
				}
				fmt.Fprint(w, payload)
			}))
			defer srv.Close()
			source := updateSource{client: srv.Client()}
			var snapshots []updateDownloadProgress
			data, e := source.download(context.Background(), srv.URL, updateMaxArchive, func(received, total int64) { snapshots = append(snapshots, updateDownloadProgress{received, total}) })
			if e != nil || string(data) != payload {
				t.Fatalf("download: %v", e)
			}
			if len(snapshots) < 3 || snapshots[0].received != 0 {
				t.Fatal(snapshots)
			}
			last := snapshots[len(snapshots)-1]
			if last.received != int64(len(payload)) {
				t.Fatal(last)
			}
			intermediate := false
			for _, p := range snapshots {
				if p.received > 0 && p.received < int64(len(payload)) {
					intermediate = true
				}
			}
			if !intermediate {
				t.Fatal("no progress until download completed")
			}
			if known && last.total != int64(len(payload)) {
				t.Fatal(last)
			}
			if !known && last.total > 0 {
				t.Fatal("invented total for chunked response")
			}
		})
	}
}
func TestUpdateDownloadProgressRender(t *testing.T) {
	for _, tc := range []struct {
		name     string
		progress updateDownloadProgress
		label    string
		counts   string
	}{
		{"zero", updateDownloadProgress{0, 2 << 20}, "0%", "0 B / 2.0 MiB"},
		{"half", updateDownloadProgress{1 << 20, 2 << 20}, "50%", "1.0 MiB / 2.0 MiB"},
		{"complete", updateDownloadProgress{2 << 20, 2 << 20}, "100%", "2.0 MiB / 2.0 MiB"},
		{"clamped", updateDownloadProgress{3 << 20, 2 << 20}, "100%", "3.0 MiB / 2.0 MiB"},
		{"unknown", updateDownloadProgress{32 << 10, -1}, "", "32.0 KiB downloaded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bar, counts := tc.progress.render(20)
			if !strings.Contains(bar, tc.label) || counts != tc.counts {
				t.Fatalf("%q %q", bar, counts)
			}
			if tc.progress.total <= 0 && strings.Contains(bar, "%") {
				t.Fatal("unknown-length percentage")
			}
			if lipgloss.Width(bar) > 27 {
				t.Fatal("bar exceeds configured width")
			}
		})
	}
}
func TestUpdateProgressModelAndBounds(t *testing.T) {
	a := &updateAttempt{events: make(chan updateEvent, 8)}
	m := newModel()
	m.updater = a
	m.latestReleaseVersion = "v1.2.3"
	next, _ := m.handleUpdateEvent(updateEvent{attempt: a, download: &updateDownloadProgress{1 << 20, 2 << 20}})
	m = next.(mainModel)
	for _, width := range []int{24, 50, 80, 120} {
		m.width, m.height = width, 40
		got := m.renderAboutModal()
		if lipgloss.Width(got) > width {
			t.Fatalf("width %d grew to %d", width, lipgloss.Width(got))
		}
		if width >= 50 && !strings.Contains(got, "50%") {
			t.Fatal("missing percentage")
		}
	}
	next, _ = m.handleUpdateEvent(updateEvent{attempt: &updateAttempt{}, download: &updateDownloadProgress{2, 2}})
	if next.(mainModel).updateDownload.received != 1<<20 {
		t.Fatal("accepted stale download progress")
	}
	next, _ = m.handleUpdateEvent(updateEvent{attempt: a, phase: "Verifying checksum…"})
	if next.(mainModel).updateDownload != nil {
		t.Fatal("retained download bar during verification")
	}
	a.closeRequested = true
	next, _ = m.handleUpdateEvent(updateEvent{attempt: a, download: &updateDownloadProgress{2 << 20, 2 << 20}})
	if next.(mainModel).updateDownload.received != 1<<20 {
		t.Fatal("download progress replaced cancellation state")
	}
}
func TestUpdateShutdownDrainsProgress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &updateAttempt{cancel: cancel, events: make(chan updateEvent, 8), done: make(chan struct{})}
	for i := 0; i < cap(a.events); i++ {
		a.events <- updateEvent{attempt: a, download: &updateDownloadProgress{int64(i), 100}}
	}
	tx := stagedUpdateFixture(t)
	go func() {
		<-ctx.Done()
		a.prepared = tx
		a.events <- updateEvent{attempt: a, transaction: tx, final: true}
		close(a.done)
	}()
	m := newModel()
	m.updater = a
	stopped := make(chan struct{})
	go func() { m.stopUpdatePreparation(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown blocked behind progress messages")
	}
	if _, e := os.Stat(tx.Directory); !os.IsNotExist(e) {
		t.Fatal("shutdown leaked prepared executable")
	}
}
