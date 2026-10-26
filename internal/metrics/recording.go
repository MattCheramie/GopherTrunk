package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Recording outcome metrics (issue #1288 §13). Metrics implements
// voice.RecorderObserver; the daemon hands it to the recorder, which reports
// every finished, dropped and failed file. Label sets are fixed and small.

type recordingMetrics struct {
	completed prometheus.Counter
	discarded *prometheus.CounterVec
	errors    *prometheus.CounterVec
	truncated prometheus.Counter
	finalize  prometheus.Histogram
}

func newRecordingMetrics() (*recordingMetrics, []prometheus.Collector) {
	r := &recordingMetrics{
		completed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Name: "recordings_completed_total",
			Help: "Recording files finished and announced (call.complete).",
		}),
		discarded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "recordings_discarded_total",
			Help: "Recording files dropped instead of announced, by reason: empty (no audio), no_speech (only idle frames), too_short (under recordings.min_duration_ms), encrypted (recordings.skip_encrypted).",
		}, []string{"reason"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "recording_errors_total",
			Help: "Recording file operations that failed, by stage: open, write, close, normalize, sidecar.",
		}, []string{"stage"}),
		truncated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace, Name: "recordings_truncated_total",
			Help: "Calls finalized on the drain timeout because the voice chain never signalled it had flushed: the file may lack its last frames.",
		}),
		finalize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace, Name: "recording_finalize_seconds",
			Help:    "Time from the start of finalizing a recording file to its announcement (close, loudness normalize, sidecar).",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}),
	}
	return r, []prometheus.Collector{r.completed, r.discarded, r.errors, r.truncated, r.finalize}
}

// RecordingCompleted implements voice.RecorderObserver.
func (m *Metrics) RecordingCompleted(finalize time.Duration) {
	m.rec.completed.Inc()
	m.rec.finalize.Observe(finalize.Seconds())
}

// RecordingDiscarded implements voice.RecorderObserver.
func (m *Metrics) RecordingDiscarded(reason string) { m.rec.discarded.WithLabelValues(reason).Inc() }

// RecordingError implements voice.RecorderObserver.
func (m *Metrics) RecordingError(stage string) { m.rec.errors.WithLabelValues(stage).Inc() }

// RecordingTruncated implements voice.RecorderObserver.
func (m *Metrics) RecordingTruncated() { m.rec.truncated.Inc() }
