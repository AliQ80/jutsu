package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// --- Version parsing / comparison --------------------------------------------

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in              string
		maj, min, patch int
		ok              bool
	}{
		{"v1.2.3", 1, 2, 3, true},
		{"1.2.3", 1, 2, 3, true},
		{"v0.44.0", 0, 44, 0, true},
		{"jj 0.42.0", 0, 42, 0, true},
		{"dev", 0, 0, 0, false},
		{"nightly", 0, 0, 0, false},
		{"v1.2", 0, 0, 0, false},
		{"v1.2.3.4", 0, 0, 0, false},
		{"v1.2.3-rc1", 0, 0, 0, false},
		{"v1.2.x", 0, 0, 0, false},
		{"", 0, 0, 0, false},
	}
	for _, c := range cases {
		maj, min, patch, ok := parseSemver(c.in)
		if ok != c.ok || maj != c.maj || min != c.min || patch != c.patch {
			t.Errorf("parseSemver(%q) = (%d,%d,%d,%v); want (%d,%d,%d,%v)",
				c.in, maj, min, patch, ok, c.maj, c.min, c.patch, c.ok)
		}
	}
}

func TestIsUpdateAvailable(t *testing.T) {
	cases := []struct {
		name    string
		running string // simulated jutsuVersion
		latest  string
		want    bool
	}{
		{"newer patch", "v1.2.3", "v1.2.4", true},
		{"newer minor", "v1.2.3", "v1.3.0", true},
		{"newer major", "v1.2.3", "v2.0.0", true},
		{"equal", "v1.2.3", "v1.2.3", false},
		{"older", "v1.2.3", "v1.2.2", false},
		{"dev running", "dev", "v1.2.4", false},
		{"running unparseable", "nightly", "v1.2.4", false},
		{"latest prerelease", "v1.2.3", "v1.2.4-rc1", false},
		{"latest malformed", "v1.2.3", "v1.2", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withJutsuVersion(t, c.running)
			if got := isUpdateAvailable(c.latest); got != c.want {
				t.Errorf("isUpdateAvailable(%q) with running %q = %v; want %v", c.latest, c.running, got, c.want)
			}
		})
	}
}

// --- GitHub lookup error handling (silent no-update) ------------------------

func TestFetchLatestReleaseErrors(t *testing.T) {
	t.Run("non-200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		saved := releaseEndpoint
		releaseEndpoint = srv.URL
		t.Cleanup(func() { releaseEndpoint = saved })

		tag, url, err := fetchLatestRelease()
		if err == nil {
			t.Fatal("expected error for non-200 status")
		}
		if tag != "" || url != "" {
			t.Errorf("fetchLatestRelease returned tag=%q url=%q on error; want empty", tag, url)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		}))
		defer srv.Close()
		saved := releaseEndpoint
		releaseEndpoint = srv.URL
		t.Cleanup(func() { releaseEndpoint = saved })

		tag, url, err := fetchLatestRelease()
		if err == nil {
			t.Fatal("expected decode error for malformed json")
		}
		if tag != "" || url != "" {
			t.Errorf("fetchLatestRelease returned tag=%q url=%q on error; want empty", tag, url)
		}
	})

	t.Run("missing tag_name", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"html_url":"https://example.com"}`))
		}))
		defer srv.Close()
		saved := releaseEndpoint
		releaseEndpoint = srv.URL
		t.Cleanup(func() { releaseEndpoint = saved })

		tag, url, err := fetchLatestRelease()
		if err == nil {
			t.Fatal("expected error for missing tag_name")
		}
		if tag != "" || url != "" {
			t.Errorf("fetchLatestRelease returned tag=%q url=%q on error; want empty", tag, url)
		}
	})
}

func TestFetchLatestReleaseDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("release lookup sent no User-Agent header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name":"v1.2.4","html_url":"https://github.com/AliQ80/jutsu/releases/tag/v1.2.4"}`))
	}))
	defer srv.Close()
	saved := releaseEndpoint
	releaseEndpoint = srv.URL
	t.Cleanup(func() { releaseEndpoint = saved })

	tag, url, err := fetchLatestRelease()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.4" {
		t.Errorf("tag = %q; want v1.2.4", tag)
	}
	if url != "https://github.com/AliQ80/jutsu/releases/tag/v1.2.4" {
		t.Errorf("url = %q; want the release page URL", url)
	}
}

// --- Startup state handling (releaseCheckMsg handler) ------------------------

func TestReleaseCheckMsgHandler(t *testing.T) {
	t.Run("newer release recorded", func(t *testing.T) {
		withJutsuVersion(t, "v1.2.3")
		m := newModel()
		m.width, m.height = 120, 40
		updated, _ := m.Update(releaseCheckMsg{version: "v1.2.4", url: "https://github.com/AliQ80/jutsu/releases/tag/v1.2.4"})
		mm := updated.(mainModel)
		if mm.latestReleaseVersion != "v1.2.4" {
			t.Errorf("latestReleaseVersion = %q; want v1.2.4", mm.latestReleaseVersion)
		}
		if mm.latestReleaseURL != "https://github.com/AliQ80/jutsu/releases/tag/v1.2.4" {
			t.Errorf("latestReleaseURL = %q; want release page URL", mm.latestReleaseURL)
		}
	})

	t.Run("silent no-update leaves state unset", func(t *testing.T) {
		withJutsuVersion(t, "v1.2.3")
		m := newModel()
		m.width, m.height = 120, 40
		// An empty message is the silent "no update" outcome (failed lookup,
		// dev build, equal, older, or malformed). It must not record a release.
		updated, _ := m.Update(releaseCheckMsg{})
		mm := updated.(mainModel)
		if mm.latestReleaseVersion != "" {
			t.Errorf("empty releaseCheckMsg should not set state; got %q", mm.latestReleaseVersion)
		}
	})
}

// --- Update-row rendering: full / compact / omitted / absent -----------------

func TestRenderUpdateRowVariants(t *testing.T) {
	t.Run("no update omitted", func(t *testing.T) {
		m := newModel()
		m.width = 120
		if got := m.renderUpdateRow(); got != "" {
			t.Errorf("renderUpdateRow with no update = %q; want empty", got)
		}
	})

	t.Run("full when wide", func(t *testing.T) {
		m := newModel()
		m.width = 120
		m.latestReleaseVersion = "v1.2.4"
		got := stripANSI(m.renderUpdateRow())
		if !strings.Contains(got, "is available — press ? for details") {
			t.Errorf("full update row missing; got %q", got)
		}
		if strings.Contains(got, "\n") {
			t.Errorf("update row wrapped; got %q", got)
		}
	})

	t.Run("compact when narrow", func(t *testing.T) {
		m := newModel()
		m.latestReleaseVersion = "v1.2.4"
		full := "↑ Jutsu v1.2.4 is available — press ? for details"
		compact := "↑ Jutsu v1.2.4 available (? for details)"
		fullW := lipgloss.Width(full)
		compactW := lipgloss.Width(compact)
		w := fullW - 1
		if w < compactW {
			t.Skip("cannot find a width between compact and full for this string")
		}
		m.width = w
		got := stripANSI(m.renderUpdateRow())
		if !strings.Contains(got, "available (? for details)") {
			t.Errorf("compact update row missing; got %q", got)
		}
		if strings.Contains(got, "is available — press") {
			t.Errorf("compact row unexpectedly shows full text; got %q", got)
		}
	})

	t.Run("omitted when too narrow", func(t *testing.T) {
		m := newModel()
		m.latestReleaseVersion = "v1.2.4"
		m.width = 5
		if got := m.renderUpdateRow(); got != "" {
			t.Errorf("update row should be omitted at width 5; got %q", got)
		}
	})
}

// renderFrameForUpdate builds the live content frame with a given detected
// release version ("" = no update) at a fixed size.
func renderFrameForUpdate(t *testing.T, version string, width, height int) string {
	t.Helper()
	m := newModel()
	m.width, m.height = width, height
	m.latestReleaseVersion = version
	return m.renderContentFrame()
}

func TestUpdateRowInMainFrame(t *testing.T) {
	base := renderFrameForUpdate(t, "", 120, 40)
	if strings.Contains(base, "is available") {
		t.Fatal("baseline frame unexpectedly shows an update row")
	}

	withUpdate := renderFrameForUpdate(t, "v1.2.4", 120, 40)
	if !strings.Contains(withUpdate, "is available — press ? for details") {
		t.Errorf("frame missing the update row; got:\n%s", withUpdate)
	}
	plain := stripANSI(withUpdate)
	if !strings.Contains(plain, "about") {
		t.Error("frame lost the help-bar about hint with an update row present")
	}
	if !strings.Contains(plain, "📂") {
		t.Error("frame lost the working-directory badge with an update row present")
	}
	if !strings.Contains(plain, "jj") {
		t.Error("frame lost the jj compatibility badge with an update row present")
	}
	// The row is accounted for in the layout: total frame height is unchanged.
	if lipgloss.Height(withUpdate) != lipgloss.Height(base) {
		t.Errorf("frame height changed with update row: got %d, want %d (row must be reserved, not added)",
			lipgloss.Height(withUpdate), lipgloss.Height(base))
	}
	if n := strings.Count(withUpdate, "is available — press ? for details"); n != 1 {
		t.Errorf("update row appears %d times; want 1", n)
	}
}

func TestUpdateRowCompactInMainFrame(t *testing.T) {
	// A long version makes the full form overflow at width 120, so the compact
	// form is selected within the frame.
	long := "v" + strings.Repeat("1", 80)
	withUpdate := renderFrameForUpdate(t, long, 120, 40)
	plain := stripANSI(withUpdate)
	if !strings.Contains(plain, "available (? for details)") {
		t.Errorf("frame missing compact update row; got:\n%s", withUpdate)
	}
	if strings.Contains(plain, "is available — press ? for details") {
		t.Error("frame showed full update text when compact should fit")
	}
}

func TestUpdateRowOmittedInMainFrame(t *testing.T) {
	// A version so long that even the compact form overflows: the row is
	// omitted and the frame height stays at the no-update baseline.
	huge := "v" + strings.Repeat("1", 200)
	withUpdate := renderFrameForUpdate(t, huge, 120, 40)
	if strings.Contains(stripANSI(withUpdate), "available") {
		t.Errorf("frame should omit the update row when nothing fits; got:\n%s", withUpdate)
	}
	base := renderFrameForUpdate(t, "", 120, 40)
	if lipgloss.Height(withUpdate) != lipgloss.Height(base) {
		t.Errorf("omitted row changed frame height: got %d, want %d", lipgloss.Height(withUpdate), lipgloss.Height(base))
	}
	if !strings.Contains(stripANSI(withUpdate), "about") {
		t.Error("frame lost the help-bar about hint when the row is omitted")
	}
}

func TestUpdateRowInEnlargedLayouts(t *testing.T) {
	for _, setup := range []struct {
		name string
		act  func(*mainModel)
	}{
		{"output enlarged", func(m *mainModel) { m.outputEnlarged = true; m.focusPane = focusOutput }},
		{"docs enlarged", func(m *mainModel) { m.docsEnlarged = true; m.focusPane = focusDocs }},
	} {
		t.Run(setup.name, func(t *testing.T) {
			base := newModel()
			base.width, base.height = 120, 40
			setup.act(&base)
			baseFrame := base.renderContentFrame()
			if strings.Contains(baseFrame, "is available") {
				t.Fatal("baseline enlarged frame unexpectedly shows an update row")
			}

			m := newModel()
			m.width, m.height = 120, 40
			m.latestReleaseVersion = "v1.2.4"
			setup.act(&m)
			frame := m.renderContentFrame()
			if !strings.Contains(frame, "is available — press ? for details") {
				t.Errorf("%s frame missing the update row; got:\n%s", setup.name, frame)
			}
			if !strings.Contains(stripANSI(frame), "about") {
				t.Errorf("%s frame lost the help-bar about hint", setup.name)
			}
			if lipgloss.Height(frame) != lipgloss.Height(baseFrame) {
				t.Errorf("%s frame height changed with update row: got %d, want %d",
					setup.name, lipgloss.Height(frame), lipgloss.Height(baseFrame))
			}
		})
	}
}

// --- About-modal update details ----------------------------------------------

func TestRenderAboutModalWithUpdate(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.latestReleaseVersion = "v1.2.4"
	m.latestReleaseURL = "https://github.com/AliQ80/jutsu/releases/tag/v1.2.4"

	got := m.renderAboutModal()
	if !strings.Contains(got, "Update available") {
		t.Errorf("about modal missing update header; got:\n%s", got)
	}
	if !strings.Contains(got, "v1.2.4") {
		t.Errorf("about modal missing release version; got:\n%s", got)
	}
	if !strings.Contains(got, "https://github.com/AliQ80/jutsu/releases/tag/v1.2.4") {
		t.Errorf("about modal missing release URL; got:\n%s", got)
	}
	if !strings.Contains(got, "jutsu     ") {
		t.Errorf("about modal lost the running-version line; got:\n%s", got)
	}
}

func TestRenderAboutModalWithoutUpdate(t *testing.T) {
	m := newModel()
	got := m.renderAboutModal()
	if strings.Contains(got, "Update available") {
		t.Errorf("about modal should not show update details with no update; got:\n%s", got)
	}
	if strings.Contains(got, "releases/tag") {
		t.Errorf("about modal should not show a release URL with no update; got:\n%s", got)
	}
}
