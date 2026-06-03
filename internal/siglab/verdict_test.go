package siglab

import "testing"

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
