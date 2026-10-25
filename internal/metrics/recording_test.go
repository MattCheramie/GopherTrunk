package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Metrics must satisfy the recorder's observer interface, or the daemon's
// wiring silently stops compiling the sink in.
var _ voice.RecorderObserver = (*Metrics)(nil)

func TestRecordingOutcomeMetrics(t *testing.T) {
	m, _ := New(nil, nil, "test", false)
	defer m.Close()

	m.RecordingCompleted(40 * time.Millisecond)
	m.RecordingCompleted(2 * time.Second)
	m.RecordingDiscarded("too_short")
	m.RecordingDiscarded("too_short")
	m.RecordingDiscarded("empty")
	m.RecordingError("write")
	m.RecordingTruncated()

	if got := testutil.ToFloat64(m.rec.completed); got != 2 {
		t.Errorf("completed = %v, want 2", got)
	}
	if got := testutil.ToFloat64(m.rec.discarded.WithLabelValues("too_short")); got != 2 {
		t.Errorf("discarded{too_short} = %v, want 2", got)
	}
	if got := testutil.ToFloat64(m.rec.errors.WithLabelValues("write")); got != 1 {
		t.Errorf("errors{write} = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.rec.truncated); got != 1 {
		t.Errorf("truncated = %v, want 1", got)
	}
	body := scrapeMetrics(t, m)
	for _, want := range []string{
		"gophertrunk_recordings_completed_total 2",
		`gophertrunk_recordings_discarded_total{reason="empty"} 1`,
		`gophertrunk_recording_errors_total{stage="write"} 1`,
		"gophertrunk_recordings_truncated_total 1",
		"gophertrunk_recording_finalize_seconds_count 2",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape missing %q", want)
		}
	}
}
