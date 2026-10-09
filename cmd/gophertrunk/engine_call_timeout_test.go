package main

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestEngineCallTimeoutCoversVoiceHangtime pins that a voice_hangtime_ms longer
// than the engine's call watchdog takes effect (#1242). The composer touches
// the engine only while voice arrives, so with the default 30 s watchdog a
// 50 s hangtime was reaped ~30 s after the last voice. Fail-first: the daemon
// used to pass call_timeout_ms straight through (0 ⇒ the engine's 30 s).
func TestEngineCallTimeoutCoversVoiceHangtime(t *testing.T) {
	cases := []struct {
		name   string
		callMs int
		hangMs int
		want   time.Duration
	}{
		{"defaults unchanged", 0, 0, 30 * time.Second},
		{"default hangtime unchanged", 0, 3500, 30 * time.Second},
		{"explicit timeout kept", 60000, 3500, 60 * time.Second},
		{"issue 1242: 50 s hangtime, default timeout", 0, 50000, 55 * time.Second},
		{"short explicit timeout raised", 5000, 3500, 8500 * time.Millisecond},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := engineCallTimeout(config.TrunkingConfig{CallTimeoutMs: tc.callMs, VoiceHangtimeMs: tc.hangMs}, nil)
			if got != tc.want {
				t.Errorf("engineCallTimeout(call=%d ms, hangtime=%d ms) = %v, want %v", tc.callMs, tc.hangMs, got, tc.want)
			}
		})
	}
}
