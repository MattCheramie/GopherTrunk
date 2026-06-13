package hunt

import (
	"context"
	"testing"
)

func TestPickGain(t *testing.T) {
	tests := []struct {
		name   string
		scores []GainScore
		want   int // index, or -1
	}{
		{"empty", nil, -1},
		{"all errored", []GainScore{{GainTenthDB: 10, Err: "x"}, {GainTenthDB: 20, Err: "y"}}, -1},
		{
			"prefer locked over lower error",
			[]GainScore{
				{GainTenthDB: 100, ErrorRate: 0, Locked: false}, // 0 error but no lock
				{GainTenthDB: 200, ErrorRate: 5, Locked: true},  // locked
			},
			1,
		},
		{
			"min error among locked",
			[]GainScore{
				{GainTenthDB: 100, ErrorRate: 8, Locked: true},
				{GainTenthDB: 200, ErrorRate: 3, Locked: true},
				{GainTenthDB: 300, ErrorRate: 6, Locked: true},
			},
			1,
		},
		{
			"tie on error ⇒ lower gain",
			[]GainScore{
				{GainTenthDB: 300, ErrorRate: 2, Locked: true},
				{GainTenthDB: 100, ErrorRate: 2, Locked: true},
			},
			1,
		},
		{
			"errored entries skipped",
			[]GainScore{
				{GainTenthDB: 100, Err: "capture"},
				{GainTenthDB: 200, ErrorRate: 4, Locked: true},
			},
			1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickGain(tc.scores); got != tc.want {
				t.Errorf("pickGain = %d, want %d", got, tc.want)
			}
		})
	}
}

// fakeGainSource is a FileIQSource that also records the gains SetGain was asked
// to apply, so a test can drive AutoGainSweep without an SDR.
type fakeGainSource struct {
	*FileIQSource
	gains []int
}

func (f *fakeGainSource) SetGain(tenthDB int) error {
	f.gains = append(f.gains, tenthDB)
	return nil
}

func TestAutoGainSweep_Orchestration(t *testing.T) {
	// Zero IQ ⇒ siglab identify is inconclusive (no lock, error rate 0) at every
	// gain, so the sweep exercises the SetGain→Tune→capture→decode loop and the
	// tie-break picks the lowest gain.
	src := &fakeGainSource{FileIQSource: NewFileIQSource(make([]complex64, 4800), 48_000)}
	gains := []int{100, 300, 200}
	recs, err := AutoGainSweep(context.Background(), src, src, []uint32{851_000_000}, AutoGainOptions{
		Gains:        gains,
		DwellSeconds: 0.05,
		SampleRateHz: 48_000,
	})
	if err != nil {
		t.Fatalf("AutoGainSweep: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("recs = %d, want 1", len(recs))
	}
	if len(recs[0].PerGain) != len(gains) {
		t.Errorf("per-gain scores = %d, want %d", len(recs[0].PerGain), len(gains))
	}
	// Every requested gain was applied, in order.
	if len(src.gains) != len(gains) {
		t.Fatalf("SetGain calls = %v, want %v", src.gains, gains)
	}
	for i, g := range gains {
		if src.gains[i] != g {
			t.Errorf("SetGain[%d] = %d, want %d", i, src.gains[i], g)
		}
	}
	// All scores tie at (no lock, 0 error) ⇒ lowest gain wins.
	if recs[0].BestGainTenthDB != 100 {
		t.Errorf("best gain = %d, want 100 (lowest on tie)", recs[0].BestGainTenthDB)
	}
}

func TestAutoGainSweep_RequiresGainControl(t *testing.T) {
	src := NewFileIQSource(make([]complex64, 16), 48_000)
	if _, err := AutoGainSweep(context.Background(), src, nil, []uint32{1}, AutoGainOptions{SampleRateHz: 48_000}); err == nil {
		t.Error("expected error when gain controller is nil")
	}
}
