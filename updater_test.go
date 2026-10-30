package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func updateTar(t *testing.T, names []string, typ byte, content []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for _, name := range names {
		h := &tar.Header{Name: name, Mode: 0755, Size: int64(len(content)), Typeflag: typ}
		if typ == tar.TypeSymlink {
			h.Linkname = "elsewhere"
			h.Size = 0
		}
		if e := tw.WriteHeader(h); e != nil {
			t.Fatal(e)
		}
		if h.Size > 0 {
			if _, e := tw.Write(content); e != nil {
				t.Fatal(e)
			}
		}
	}
	if e := tw.Close(); e != nil {
		t.Fatal(e)
	}
	if e := gz.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func updateZip(t *testing.T, names []string, mode os.FileMode, content []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(mode)
		w, e := z.CreateHeader(h)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = w.Write(content); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestUpdateArchivePlatforms(t *testing.T) {
	for _, osName := range []string{"linux", "darwin", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			got, e := updateArchiveName("v1.2.3", osName, arch)
			if e != nil || !strings.HasPrefix(got, "jutsu_1.2.3_"+osName+"_"+arch) {
				t.Fatalf("%s: %v", got, e)
			}
		}
	}
	for _, args := range [][3]string{{"v1.2.3", "freebsd", "amd64"}, {"v1.2.3", "linux", "386"}, {"bad", "linux", "amd64"}} {
		if _, e := updateArchiveName(args[0], args[1], args[2]); e == nil {
			t.Fatal(args)
		}
	}
}
func TestUpdateChecksums(t *testing.T) {
	data := []byte("binary")
	name := "jutsu.tar.gz"
	sum := sha256.Sum256(data)
	good := fmt.Sprintf("%x  %s\n", sum, name)
	for _, tc := range []struct {
		name, manifest string
		bad            bool
	}{
		{"valid", good, false}, {"uppercase", strings.ToUpper(fmt.Sprintf("%x", sum)) + " *" + name, false},
		{"missing", "", true}, {"duplicate", good + good, true}, {"malformed", "1234 " + name, true},
		{"malformed duplicate", good + good[:len(good)-1] + " extra\n", true},
		{"mismatch", strings.Repeat("0", 64) + " " + name, true}, {"different file", strings.ReplaceAll(good, name, "another"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := verifyUpdateChecksum([]byte(tc.manifest), data, name)
			if (e != nil) != tc.bad {
				t.Fatal(e)
			}
		})
	}
}
func TestUpdateArchiveExtraction(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			expected := "jutsu"
			if goos == "windows" {
				expected += ".exe"
			}
			for _, tc := range []struct {
				name    string
				names   []string
				link    bool
				wantErr bool
			}{
				{"valid", []string{"LICENSE", expected}, false, false},
				{"missing", []string{"LICENSE"}, false, true},
				{"duplicate", []string{expected, expected}, false, true},
				{"traversal", []string{"../escape", expected}, false, true},
				{"absolute", []string{"/escape", expected}, false, true},
				{"backslash", []string{"..\\escape", expected}, false, true},
				{"link", []string{expected}, true, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var b []byte
					if goos == "windows" {
						mode := os.FileMode(0755)
						if tc.link {
							mode |= os.ModeSymlink
						}
						b = updateZip(t, tc.names, mode, []byte("new binary"))
					} else {
						typ := byte(tar.TypeReg)
						if tc.link {
							typ = tar.TypeSymlink
						}
						b = updateTar(t, tc.names, typ, []byte("new binary"))
					}
					got, e := extractUpdate(b, goos)
					if (e != nil) != tc.wantErr {
						t.Fatalf("%q %v", got, e)
					}
					if e == nil && string(got) != "new binary" {
						t.Fatal(string(got))
					}
				})
			}
		})
	}
	if _, e := extractUpdate([]byte("broken"), "linux"); e == nil {
		t.Fatal("accepted invalid gzip")
	}
	if _, e := extractUpdate([]byte("broken"), "windows"); e == nil {
		t.Fatal("accepted invalid zip")
	}
}

// HTTPS fixture uses the same metadata and archive contract as GitHub, with no
// dependency on a real release and no writes to the running test executable.
func updateFixture(t *testing.T, goos string) (updateSource, *httptest.Server) {
	t.Helper()
	var server *httptest.Server
	name, _ := updateArchiveName("v1.2.3", goos, runtime.GOARCH)
	var archive []byte
	if goos == "windows" {
		archive = updateZip(t, []string{"jutsu.exe"}, 0755, []byte("new release"))
	} else {
		archive = updateTar(t, []string{"jutsu"}, tar.TypeReg, []byte("new release"))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/tags/v1.2.3", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.2.3", "assets": []map[string]string{{"name": name, "browser_download_url": server.URL + "/archive"}, {"name": "checksums.txt", "browser_download_url": server.URL + "/checksums"}}})
	})
	mux.HandleFunc("/archive", func(w http.ResponseWriter, r *http.Request) { w.Write(archive) })
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "%x  %s\n", sha256.Sum256(archive), name) })
	server = httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	return updateSource{client: server.Client(), endpoint: server.URL + "/tags/"}, server
}
func TestUpdatePinnedReleaseAndPreparation(t *testing.T) {
	s, _ := updateFixture(t, runtime.GOOS)
	target := filepath.Join(t.TempDir(), "custom executable")
	if e := os.WriteFile(target, []byte("old"), 0755); e != nil {
		t.Fatal(e)
	}
	var phases []string
	tx, e := prepareUpdate(context.Background(), s, target, "v1.2.3", func(phase string) { phases = append(phases, phase) })
	if e != nil {
		t.Fatal(e)
	}
	defer tx.cleanup()
	if len(phases) != 4 {
		t.Fatal(phases)
	}
	b, _ := os.ReadFile(tx.staged())
	if string(b) != "new release" {
		t.Fatal(string(b))
	}
	old, _ := os.ReadFile(target)
	if string(old) != "old" {
		t.Fatal("preparation modified installed executable")
	}
	if _, e = s.binary(context.Background(), "v1.2.4", runtime.GOOS, runtime.GOARCH, func(string) {}); e == nil {
		t.Fatal("silently selected a different release")
	}
	if _, e := beginUpdate(target); e == nil {
		t.Fatal("allowed concurrent updater")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.binary(ctx, "v1.2.3", runtime.GOOS, runtime.GOARCH, func(string) {}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
func TestUpdateDownloadErrorsAndLimits(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(503)
			return
		}
		io.WriteString(w, "too large")
	}))
	defer server.Close()
	s := updateSource{client: server.Client(), endpoint: server.URL}
	for _, tc := range []struct {
		address string
		limit   int64
	}{{server.URL, 2}, {server.URL + "/fail", 100}, {"http://example.test", 100}} {
		if _, e := s.download(context.Background(), tc.address, tc.limit); e == nil {
			t.Fatal(tc)
		}
	}
}
func stagedUpdateFixture(t *testing.T) *updateTransaction {
	t.Helper()
	target := filepath.Join(t.TempDir(), "jutsu with spaces")
	if e := os.WriteFile(target, []byte("old"), 0755); e != nil {
		t.Fatal(e)
	}
	tx, e := beginUpdate(target)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { tx.cleanup() })
	if e = tx.stage(context.Background(), []byte("new")); e != nil {
		t.Fatal(e)
	}
	return tx
}
func TestUpdateIdentityAndRollback(t *testing.T) {
	t.Run("changed target", func(t *testing.T) {
		tx := stagedUpdateFixture(t)
		if e := os.WriteFile(tx.Target, []byte("other"), 0755); e != nil {
			t.Fatal(e)
		}
		if e := tx.revalidate(); e == nil {
			t.Fatal("accepted changed target")
		}
	})
	t.Run("changed staged", func(t *testing.T) {
		tx := stagedUpdateFixture(t)
		os.WriteFile(tx.staged(), []byte("other"), 0755)
		if e := tx.revalidate(); e == nil {
			t.Fatal("accepted changed staged")
		}
	})
	t.Run("restored", func(t *testing.T) {
		tx := stagedUpdateFixture(t)
		if e := replaceUpdateFile(tx.staged(), tx.Target); e != nil {
			t.Fatal(e)
		}
		e := tx.rollback(errors.New("launch failed"))
		if !strings.Contains(e.Error(), "restored") {
			t.Fatal(e)
		}
		b, _ := os.ReadFile(tx.Target)
		if string(b) != "old" {
			t.Fatal("did not restore old executable")
		}
	})
	t.Run("restore failed", func(t *testing.T) {
		tx := stagedUpdateFixture(t)
		os.Remove(tx.Target)
		os.Mkdir(tx.Target, 0700)
		os.WriteFile(filepath.Join(tx.Target, "blocker"), []byte("x"), 0600)
		e := tx.rollback(errors.New("failed"))
		if !strings.Contains(e.Error(), "Recovery executable:") {
			t.Fatal(e)
		}
		if _, e := os.Stat(tx.backup()); e != nil {
			t.Fatal("lost recovery copy")
		}
	})
}
func TestUpdateRecovery(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(fmt.Sprint(commit), func(t *testing.T) {
			tx := stagedUpdateFixture(t)
			tx.Committed = commit
			if commit {
				if e := replaceUpdateFile(tx.staged(), tx.Target); e != nil {
					t.Fatal(e)
				}
			}
			tx.save()
			tx.lock.Close()
			tx.lock = nil
			notices := inspectUpdateRecovery(tx.Target)
			if len(notices) != 0 {
				t.Fatal(notices)
			}
			if _, e := os.Stat(tx.Directory); !os.IsNotExist(e) {
				t.Fatal("did not clean completed or uncommitted transaction")
			}
		})
	}
	t.Run("interrupted", func(t *testing.T) {
		tx := stagedUpdateFixture(t)
		tx.Committed = true
		tx.save()
		tx.lock.Close()
		tx.lock = nil
		if notices := inspectUpdateRecovery(tx.Target); len(notices) != 1 {
			t.Fatal(notices)
		}
		if _, e := os.Stat(tx.backup()); e != nil {
			t.Fatal(e)
		}
	})
	t.Run("reject foreign paths", func(t *testing.T) {
		tx := stagedUpdateFixture(t)
		tx.Target = filepath.Join(t.TempDir(), "foreign")
		if e := validateUpdateTransaction(tx, tx.record()); e == nil {
			t.Fatal("accepted foreign target")
		}
	})
}
func updateKey(s string) tea.KeyPressMsg { return tea.KeyPressMsg{Code: rune(s[0]), Text: s} }
func TestUpdateModalLifecycle(t *testing.T) {
	t.Run("no release", func(t *testing.T) {
		m := newModel()
		m.aboutModalOpen = true
		next, cmd := m.handleAboutModalKeys(updateKey("u"))
		if next.(mainModel).updater != nil || cmd != nil {
			t.Fatal("started without release")
		}
	})
	t.Run("running", func(t *testing.T) {
		m := newModel()
		m.latestReleaseVersion = "v1.2.3"
		m.running = true
		next, cmd := m.startUpdate()
		if cmd != nil || !strings.Contains(next.(mainModel).updateStatus, "running command") {
			t.Fatal("did not block running command")
		}
	})
	t.Run("only explicit action", func(t *testing.T) {
		m := newModel()
		m.latestReleaseVersion = "v1.2.3"
		m.width, m.height = 120, 40
		m.aboutModalOpen = true
		m.renderAboutModal()
		if m.updater != nil {
			t.Fatal("rendering started update")
		}
		next, cmd := m.startUpdate()
		mm := next.(mainModel)
		defer mm.updater.cancel()
		if cmd == nil || mm.updater == nil {
			t.Fatal("action did not start attempt")
		}
		again, cmd := mm.startUpdate()
		if again.(mainModel).updater != mm.updater || cmd != nil {
			t.Fatal("duplicate attempt")
		}
	})
	t.Run("cancel closes after cleanup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		a := &updateAttempt{cancel: cancel, events: make(chan updateEvent, 8)}
		m := newModel()
		m.updater = a
		m.aboutModalOpen = true
		m.focusPane = focusDocs
		next, _ := m.handleAboutModalKeys(tea.KeyPressMsg{Code: tea.KeyEsc})
		mm := next.(mainModel)
		if ctx.Err() == nil || !mm.aboutModalOpen {
			t.Fatal("did not request asynchronous cancellation")
		}
		tx := stagedUpdateFixture(t)
		next, cmd := mm.handleUpdateEvent(updateEvent{attempt: a, transaction: tx, final: true})
		mm = next.(mainModel)
		if mm.aboutModalOpen || mm.updater != nil || mm.focusPane != focusDocs || cmd != nil {
			t.Fatal("cancel did not restore focus")
		}
		if _, e := os.Stat(tx.Directory); !os.IsNotExist(e) {
			t.Fatal("leaked stage")
		}
	})
	t.Run("commit ignores dismissal", func(t *testing.T) {
		for _, key := range []tea.KeyPressMsg{{Code: tea.KeyEsc}, updateKey("q"), updateKey("u"), {Code: 'c', Mod: tea.ModCtrl}} {
			m := newModel()
			m.aboutModalOpen = true
			m.updateCommitting = true
			next, cmd := m.handleAboutModalKeys(key)
			if !next.(mainModel).aboutModalOpen || cmd != nil {
				t.Fatal("interrupted commit")
			}
		}
	})
	t.Run("stale results cleaned", func(t *testing.T) {
		tx := stagedUpdateFixture(t)
		m := newModel()
		a := &updateAttempt{}
		next, _ := m.handleUpdateEvent(updateEvent{attempt: a, transaction: tx, final: true})
		if next.(mainModel).updateCommitting {
			t.Fatal("accepted stale result")
		}
		if _, e := os.Stat(tx.Directory); !os.IsNotExist(e) {
			t.Fatal("leaked stale stage")
		}
	})
	t.Run("error allows retry", func(t *testing.T) {
		a := &updateAttempt{cancel: func() {}}
		m := newModel()
		m.updater = a
		m.aboutModalOpen = true
		next, _ := m.handleUpdateEvent(updateEvent{attempt: a, final: true, err: errors.New("bad checksum")})
		mm := next.(mainModel)
		if mm.updater != nil || !mm.aboutModalOpen || !strings.Contains(mm.updateStatus, "retry") {
			t.Fatal("unusable error state")
		}
	})
}
func TestUpdateModalBounds(t *testing.T) {
	for _, w := range []int{24, 50, 80, 120} {
		m := newModel()
		m.width, m.height = w, 40
		m.latestReleaseVersion = "v1.2.3"
		m.latestReleaseURL = "https://github.com/AliQ80/jutsu/releases/tag/v1.2.3"
		m.updateStatus = strings.Repeat("long error ", 30)
		got := m.renderAboutModal()
		if lipgloss.Width(got) > w {
			t.Fatalf("width %d grew to %d", w, lipgloss.Width(got))
		}
	}
}

func TestUpdateInstallFailures(t *testing.T) {
	for _, failure := range []string{"none", "replace", "launch", "restore"} {
		t.Run(failure, func(t *testing.T) {
			tx := stagedUpdateFixture(t)
			calls := 0
			launched := false
			tx.Args = []string{"jutsu", "argument with spaces", "$(literal)"}
			tx.Env = []string{"EXAMPLE=literal value"}
			tx.Cwd = t.TempDir()
			e := tx.install(updateInstallOps{replace: func(from, to string) error {
				calls++
				if failure == "replace" && calls == 1 {
					return errors.New("replace failed")
				}
				if failure == "restore" && calls == 2 {
					return errors.New("restore failed")
				}
				return replaceUpdateFile(from, to)
			}, launch: func(got *updateTransaction) error {
				launched = true
				if got.Args[1] != "argument with spaces" || got.Args[2] != "$(literal)" || got.Env[0] != "EXAMPLE=literal value" {
					t.Fatal("restart context changed")
				}
				if failure == "launch" || failure == "restore" {
					return errors.New("launch failed")
				}
				return nil
			}})
			b, _ := os.ReadFile(tx.Target)
			if failure == "none" {
				if e != nil || string(b) != "new" || !launched {
					t.Fatalf("%s %v", b, e)
				}
				return
			}
			if e == nil {
				t.Fatal("failure was swallowed")
			}
			if failure == "restore" {
				if _, e := os.Stat(tx.backup()); e != nil {
					t.Fatal("recovery copy lost")
				}
				return
			}
			if string(b) != "old" {
				t.Fatal("old binary was not restored")
			}
			if failure == "replace" && launched {
				t.Fatal("launched after failed replacement")
			}
		})
	}
}
func TestUpdateArchiveEntryLimit(t *testing.T) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	if e := tw.WriteHeader(&tar.Header{Name: "jutsu", Mode: 0755, Size: updateMaxBinary + 1}); e != nil {
		t.Fatal(e)
	}
	// A truncated overlarge entry is rejected from its header before allocation.
	_ = tw.Close()
	_ = gz.Close()
	if _, e := extractUpdate(b.Bytes(), "linux"); e == nil || !strings.Contains(e.Error(), "size limit") {
		t.Fatal(e)
	}
}
func TestUpdateReleaseMetadataFailures(t *testing.T) {
	for _, body := range []string{"{", `{"tag_name":"v9.0.0"}`, `{"tag_name":"v1.2.3","assets":[]}`, `{"tag_name":"v1.2.3","assets":[{"name":"checksums.txt","browser_download_url":"https://example.test/a"},{"name":"checksums.txt","browser_download_url":"https://example.test/b"}]}`} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
			defer srv.Close()
			source := updateSource{client: srv.Client(), endpoint: srv.URL + "/"}
			if _, e := source.binary(context.Background(), "v1.2.3", runtime.GOOS, runtime.GOARCH, func(string) {}); e == nil {
				t.Fatal("accepted invalid metadata")
			}
		})
	}
}
func TestUpdateCancellationCleansStaging(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { cancel(); <-r.Context().Done() }))
	defer srv.Close()
	source := updateSource{client: srv.Client(), endpoint: srv.URL + "/"}
	target := filepath.Join(t.TempDir(), "jutsu")
	os.WriteFile(target, []byte("old"), 0755)
	tx, e := prepareUpdate(ctx, source, target, "v1.2.3", func(string) {})
	if tx != nil || !errors.Is(e, context.Canceled) {
		t.Fatalf("%v %v", tx, e)
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".jutsu-update-") {
			t.Fatal("cancelled preparation leaked staging directory")
		}
	}
	lock, e := lockUpdate(target)
	if e != nil {
		t.Fatal("cancelled preparation retained lock")
	}
	lock.Close()
}
func TestUpdateHandoffFailure(t *testing.T) {
	tx := stagedUpdateFixture(t)
	a := &updateAttempt{}
	m := newModel()
	m.updater = a
	m.aboutModalOpen = true
	m.updateCommitting = true
	next, cmd := m.handleUpdateHandoff(updateHandoffMsg{attempt: a, transaction: tx, err: errors.New("helper failed")})
	mm := next.(mainModel)
	if mm.updateCommitting || mm.updater != nil || mm.updateRestart != nil || !mm.aboutModalOpen || cmd != nil {
		t.Fatal("handoff failure did not keep current app usable")
	}
	b, _ := os.ReadFile(tx.Target)
	if string(b) != "old" {
		t.Fatal("helper failure changed binary")
	}
}
func TestUpdateQuitWaitsForCleanup(t *testing.T) {
	cancelled := false
	a := &updateAttempt{cancel: func() { cancelled = true }}
	m := newModel()
	m.aboutModalOpen = true
	m.updater = a
	next, cmd := m.handleAboutModalKeys(updateKey("q"))
	mm := next.(mainModel)
	if cmd != nil || !cancelled || !a.quitRequested {
		t.Fatal("quit skipped cancellation")
	}
	tx := stagedUpdateFixture(t)
	next, cmd = mm.handleUpdateEvent(updateEvent{attempt: a, final: true, transaction: tx})
	if cmd == nil {
		t.Fatal("did not quit after cleanup")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("wrong quit message")
	}
	if next.(mainModel).updater != nil {
		t.Fatal("retained attempt")
	}
}
func TestUpdateModalIgnoresMouseFocus(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories
	m.freezeBackdrop()
	m.aboutModalOpen = true
	next, _ := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 100, Y: 3})
	if next.(mainModel).focusPane != focusCategories {
		t.Fatal("mouse changed focus behind About")
	}
}
