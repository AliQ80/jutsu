package main

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// The snapshot sent to the UI is immutable. Unknown response lengths retain a
// moving marker and byte count instead of inventing a completion percentage.
type updateDownloadProgress struct{ received, total int64 }

func (p updateDownloadProgress) render(width int) (string, string) {
	width = max(1, width)
	received := max(int64(0), p.received)
	if p.total > 0 {
		fraction := min(1, float64(received)/float64(p.total))
		filled := int(fraction * float64(width))
		bar := updateTrackStyle.Render("[") + updateProgressStyle.Render(strings.Repeat("━", filled)) + updateTrackStyle.Render(strings.Repeat("─", width-filled)+fmt.Sprintf("] %3.0f%%", fraction*100))
		return bar, formatUpdateBytes(received) + " / " + formatUpdateBytes(p.total)
	}
	position := int(received / (32 << 10) % int64(width))
	bar := updateTrackStyle.Render("["+strings.Repeat("─", position)) + updateProgressStyle.Render("━") + updateTrackStyle.Render(strings.Repeat("─", width-position-1)+"]")
	return bar, formatUpdateBytes(received) + " downloaded"
}
func formatUpdateBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1<<20 {
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
}

type updateProgressReader struct {
	reader          io.Reader
	received, total int64
	report          func(int64, int64)
	last            time.Time
}

func (r *updateProgressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.received += int64(n)
	if n > 0 && time.Since(r.last) >= 100*time.Millisecond {
		r.report(r.received, r.total)
		r.last = time.Now()
	}
	return n, err
}
