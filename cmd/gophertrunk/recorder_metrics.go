package main

import (
	"github.com/MattCheramie/GopherTrunk/internal/metrics"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// recorderObserver hands the recorder the Prometheus sink for its
// completed / discarded / error / truncated counters (issue #1288 §13).
// A nil *metrics.Metrics must become a nil INTERFACE, not a typed nil, or
// the recorder's nil check passes and every finalize dereferences nil.
func recorderObserver(m *metrics.Metrics) voice.RecorderObserver {
	if m == nil {
		return nil
	}
	return m
}
