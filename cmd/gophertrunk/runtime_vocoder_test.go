package main

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestRuntimeVocoderMapMatchesRecorderDefaults keeps the Settings panel's
// protocol → vocoder table (GET /api/v1/runtime vocoder_map) from drifting
// away from the map the recorder actually loads. The table is a hand-kept
// copy, and it said "ambe2" for P25 Phase 2 and "—" for YSF after the
// recorder had moved on; a wrong name here sends an operator to the wrong
// `decode -vocoder` when they re-decode a .raw sidecar.
func TestRuntimeVocoderMapMatchesRecorderDefaults(t *testing.T) {
	loaded := voice.DefaultVocoderForProtocol()
	shared := 0
	for proto, shown := range vocoderProtocolMap {
		want, ok := loaded[proto]
		if !ok {
			continue // display-only rows: analog protocols, the "dmr" family label
		}
		shared++
		if shown != want {
			t.Errorf("runtime vocoder_map[%q] = %q, but the recorder loads %q", proto, shown, want)
		}
	}
	if shared == 0 {
		t.Fatal("no protocol is named by both maps; the comparison is checking nothing")
	}
	// The DMR family is shown under one label; it must name what the tiers load.
	if got, want := vocoderProtocolMap["dmr"], loaded["dmr-tier2"]; got != want {
		t.Errorf("runtime vocoder_map[\"dmr\"] = %q, but the recorder loads %q for dmr-tier2", got, want)
	}
}
