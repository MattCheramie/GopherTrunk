package siglab

import (
	"strings"
	"testing"
)

// TestClassifyNoLockReason pins the no-lock diagnosis classifier (issues
// #764 / #771): a capture-quality failure must be named as such, not left to be
// misread as a tuning or AGC fault. The SNR-limited case mirrors the verified
// #771 10 MS/s capture (EVM 22.5 % / demod SNR 9.5 dB, no FSW sync).
func TestClassifyNoLockReason(t *testing.T) {
	p25 := func(cc *CCStatsBreakdown, hits int) *P25P1Detail {
		return &P25P1Detail{CCStats: cc, WinningHits: hits}
	}
	demod := func(evm, snr float64) *SignalQuality {
		return &SignalQuality{Demod: &DemodMetrics{Modulation: "c4fm", EVMPct: evm, SNREstimateDB: snr}}
	}
	cases := []struct {
		name string
		r    *Result
		want string // expected substring ("" ⇒ must be empty)
	}{
		{"locked ⇒ no reason", &Result{Locked: true, Symbols: 1000}, ""},
		{"no symbols", &Result{Symbols: 0}, "no symbols"},
		{"SNR-limited (the #771 10 MS/s case)",
			&Result{Symbols: 48000, Signal: demod(22.5, 9.5), Detail: p25(&CCStatsBreakdown{}, 0)},
			"SNR-limited capture"},
		{"clean eye but no sync ⇒ alignment",
			&Result{Symbols: 48000, Signal: demod(6.0, 20.0), Detail: p25(&CCStatsBreakdown{}, 0)},
			"frame sync never aligned"},
		{"synced but NID never validates ⇒ degraded frames",
			&Result{Symbols: 48000, Signal: demod(13.0, 13.0), Detail: p25(&CCStatsBreakdown{NIDFailed: 40}, 5)},
			"NID never validates"},
		{"valid NIDs but no control DUID",
			&Result{Symbols: 48000, Detail: p25(&CCStatsBreakdown{NIDTrusted: 10}, 8)},
			"not the control channel"},
		{"no -diag, no frames ⇒ hint at -diag/-auto-tune",
			&Result{Symbols: 48000, Detail: p25(&CCStatsBreakdown{}, 0)},
			"re-run with -diag"},
	}
	for _, c := range cases {
		got := classifyNoLockReason(c.r)
		if c.want == "" {
			if got != "" {
				t.Errorf("%s: want empty reason, got %q", c.name, got)
			}
			continue
		}
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want substring %q", c.name, got, c.want)
		}
	}
}

func TestFieldMatchesHexTolerant(t *testing.T) {
	cases := []struct {
		expected any
		got      any
		want     bool
	}{
		{"0x293", uint16(0x293), true},
		{"659", uint16(0x293), true}, // 0x293 == 659
		{0x293, uint16(0x293), true},
		{"0xA", uint8(10), true},
		{"0x294", uint16(0x293), false},
		{"sch/hd", "sch/hd", true},
		{"sch/hd", "bsch", false},
	}
	for _, c := range cases {
		if got := fieldMatches(c.expected, c.got); got != c.want {
			t.Errorf("fieldMatches(%v, %v) = %v, want %v", c.expected, c.got, got, c.want)
		}
	}
}

func TestEvaluateAcceptance(t *testing.T) {
	r := &Result{
		Locked:         true,
		LockLatencySec: 0.5,
		Lock:           &LockInfo{FrequencyHz: 851000000, Fields: map[string]any{"NAC": uint16(0x293)}},
		Grants:         []GrantRecord{{}, {}},
		ExpectedBaud:   4800,
		EffectiveBaud:  4800,
		Signal:         &SignalQuality{DecodeErrorRate: 1.0},
	}
	pass := evaluateAcceptance(r, &Acceptance{
		Lock:               boolPtr(true),
		LockLatencyMaxSec:  1.0,
		LockFields:         map[string]any{"nac": "0x293", "frequency_hz": 851000000},
		MinGrants:          2,
		BaudTolerancePct:   1,
		MaxDecodeErrorRate: 5,
	})
	if !pass.Pass {
		t.Errorf("expected pass, got failures: %v", pass.Failures)
	}

	fail := evaluateAcceptance(r, &Acceptance{
		Lock:       boolPtr(true),
		LockFields: map[string]any{"nac": "0x999"},
		MinGrants:  5,
	})
	if fail.Pass {
		t.Error("expected failure for wrong NAC + too-few grants")
	}
	if len(fail.Failures) != 2 {
		t.Errorf("expected 2 failures, got %d: %v", len(fail.Failures), fail.Failures)
	}
}

func TestEvaluateAcceptanceDemodBounds(t *testing.T) {
	r := &Result{
		Signal: &SignalQuality{Demod: &DemodMetrics{Modulation: "c4fm", EVMPct: 12.0, SNREstimateDB: 18.0}},
	}
	if v := evaluateAcceptance(r, &Acceptance{MaxEVMPct: 15, MinSNRdB: 15}); !v.Pass {
		t.Errorf("expected pass within EVM/SNR bounds, got %v", v.Failures)
	}
	if v := evaluateAcceptance(r, &Acceptance{MaxEVMPct: 10}); v.Pass {
		t.Error("expected EVM failure (12%% > 10%%)")
	}
	if v := evaluateAcceptance(r, &Acceptance{MinSNRdB: 20}); v.Pass {
		t.Error("expected SNR failure (18 dB < 20 dB)")
	}
	// Missing demod metrics fails a required bound rather than silently passing.
	bare := &Result{Signal: &SignalQuality{}}
	if v := evaluateAcceptance(bare, &Acceptance{MaxEVMPct: 15}); v.Pass {
		t.Error("expected failure when demod metrics absent but EVM bound set")
	}
}
