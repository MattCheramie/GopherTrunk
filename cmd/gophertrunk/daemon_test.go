package main

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestLooksDriveRootedOnWindows(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Unix-style absolute paths lose their drive on Windows.
		{"/var/lib/gophertrunk/recordings", true},
		{"\\var\\lib\\gophertrunk\\recordings", true},
		{"/calls.db", true},
		// Drive-qualified and UNC paths are fine.
		{"C:\\Users\\me\\GopherTrunk", false},
		{"\\\\server\\share\\recordings", false},
		{"//server/share", false},
		// Relative paths are fine.
		{"recordings", false},
		{"data/recordings", false},
		{".\\recordings", false},
		{"", false},
	}
	for _, c := range cases {
		if got := looksDriveRootedOnWindows(c.path); got != c.want {
			t.Errorf("looksDriveRootedOnWindows(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// recordingTuner records the last SetCenterFreq argument and optionally
// returns a fixed error, so the offsetTuner wrapper can be asserted in
// isolation (issue #402).
type recordingTuner struct {
	last uint32
	err  error
}

func (r *recordingTuner) SetCenterFreq(hz uint32) error {
	r.last = hz
	return r.err
}

func TestOffsetTunerShiftsLO(t *testing.T) {
	inner := &recordingTuner{}
	tn := offsetTuner{inner: inner, offsetHz: 512_000}
	if err := tn.SetCenterFreq(851_000_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if inner.last != 850_488_000 {
		t.Errorf("inner LO = %d, want %d (logical freq minus offset)", inner.last, 850_488_000)
	}

	// Errors from the inner tuner propagate unchanged.
	wantErr := errors.New("tune failed")
	inner2 := &recordingTuner{err: wantErr}
	tn2 := offsetTuner{inner: inner2, offsetHz: 250_000}
	if err := tn2.SetCenterFreq(420_012_500); !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if inner2.last != 419_762_500 {
		t.Errorf("inner LO = %d, want %d", inner2.last, 419_762_500)
	}
}

func TestControlLOOffsetHz(t *testing.T) {
	const c4fm = 48_000.0
	const tetra = 144_000.0
	cases := []struct {
		name         string
		effectiveHz  uint32
		minDDCTarget float64
		dcAvoid      bool
		explicitHz   int
		want         float64
	}{
		{"disabled_by_default", 2_048_000, c4fm, false, 0, 0},
		{"passthrough_rate_no_room", 48_000, c4fm, true, 0, 0},
		{"auto_quarter_rate", 2_048_000, c4fm, true, 0, 512_000},
		{"auto_960k", 960_000, c4fm, true, 0, 240_000},
		{"explicit_honored", 2_048_000, c4fm, true, 300_000, 300_000},
		{"explicit_exceeds_nyquist_rejected", 2_048_000, c4fm, true, 2_000_000, 0},
		{"tetra_target_has_room", 2_048_000, tetra, true, 0, 512_000},
		{"disabled_ignores_explicit", 2_048_000, c4fm, false, 300_000, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := controlLOOffsetHz(c.effectiveHz, c.minDDCTarget, c.dcAvoid, c.explicitHz)
			if got != c.want {
				t.Errorf("controlLOOffsetHz(%d, %v, %v, %d) = %v, want %v",
					c.effectiveHz, c.minDDCTarget, c.dcAvoid, c.explicitHz, got, c.want)
			}
		})
	}
}

func TestMinControlDDCTarget(t *testing.T) {
	// No systems → default C4FM 48 kHz.
	if got := minControlDDCTarget(nil); got != 48_000 {
		t.Errorf("minControlDDCTarget(nil) = %v, want 48000", got)
	}
	// A pure P25 control SDR gates on 48 kHz.
	p25 := []trunking.System{{Name: "A", Protocol: trunking.ProtocolP25}}
	if got := minControlDDCTarget(p25); got != 48_000 {
		t.Errorf("minControlDDCTarget(P25) = %v, want 48000", got)
	}
	// A mixed C4FM + TETRA control SDR must gate on the narrower 48 kHz.
	mixed := []trunking.System{
		{Name: "A", Protocol: trunking.ProtocolTETRA},
		{Name: "B", Protocol: trunking.ProtocolP25},
	}
	if got := minControlDDCTarget(mixed); got != 48_000 {
		t.Errorf("minControlDDCTarget(mixed) = %v, want 48000 (narrowest)", got)
	}
	// A pure TETRA control SDR gates on 144 kHz.
	tetra := []trunking.System{{Name: "A", Protocol: trunking.ProtocolTETRA}}
	if got := minControlDDCTarget(tetra); got != 144_000 {
		t.Errorf("minControlDDCTarget(TETRA) = %v, want 144000", got)
	}
}
