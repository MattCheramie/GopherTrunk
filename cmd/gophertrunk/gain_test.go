package main

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

func TestParseGain(t *testing.T) {
	cases := []struct {
		in     string
		want   int
		wantOk bool
	}{
		{"", -1, true},
		{"auto", -1, true},
		{"AUTO", -1, true},
		{"  Auto  ", -1, true},
		{"496", 496, true},
		{"49.6", 496, true},
		{"49,6", 496, true}, // comma decimal tolerated
		{"0", 0, true},
		{"-1", -1, true},
		// Malformed inputs surface as ok=false so the daemon can warn
		// rather than silently using a wrong gain.
		{"high", 0, false},
		{"--", 0, false},
		{"49.6.0", 0, false},
	}
	for _, c := range cases {
		got, ok := parseGain(c.in)
		if got != c.want || ok != c.wantOk {
			t.Errorf("parseGain(%q) = (%d, %v), want (%d, %v)",
				c.in, got, ok, c.want, c.wantOk)
		}
	}
}

func TestGainLooksLikeDBMistake(t *testing.T) {
	cases := []struct {
		raw     string
		tenthDB int
		want    bool
	}{
		// Bare integers that parse to <= 5.0 dB are almost certainly a
		// dB value the operator forgot to express in tenths.
		{"32", 32, true}, // the reported case: meant 32 dB, got 3.2 dB
		{"1", 1, true},
		{"50", 50, true}, // boundary: 5.0 dB still suspicious
		// Plausible manual gains (>5.0 dB) and the shipped examples.
		{"51", 51, false},
		{"229", 229, false},
		{"320", 320, false}, // the correct value for 32 dB
		{"496", 496, false},
		// Decimal forms are interpreted as whole dB by parseGain, so a
		// decimal is an explicit choice — never flag it.
		{"32.0", 320, false},
		{"4.0", 40, false},
		{"49.6", 496, false},
		{"3,2", 32, false}, // comma decimal, locale paste
		// auto / disabled gain must never warn.
		{"auto", -1, false},
		{"", -1, false},
		{"0", 0, false},
		{"-1", -1, false},
	}
	for _, c := range cases {
		if got := gainLooksLikeDBMistake(c.raw, c.tenthDB); got != c.want {
			t.Errorf("gainLooksLikeDBMistake(%q, %d) = %v, want %v",
				c.raw, c.tenthDB, got, c.want)
		}
	}
}

func TestGainLooksTooLow(t *testing.T) {
	cases := []struct {
		raw     string
		tenthDB int
		want    bool
	}{
		// The (50,150)-tenths band: a real, cleanly-parsed manual gain that
		// is still too low for digital decode (issue #402: 8.7 dB applied).
		{"87", 87, true},   // the reported case: 8.7 dB, no FSW
		{"51", 51, true},   // just above the dB-mistake band
		{"149", 149, true}, // just below the floor
		// The dB-mistake band (<=50 tenths) is owned by gainLooksLikeDBMistake,
		// so gainLooksTooLow must not double-fire there.
		{"32", 32, false},
		{"50", 50, false},
		{"1", 1, false},
		// Plausible manual gains at or above the floor never warn.
		{"150", 150, false},
		{"229", 229, false},
		{"496", 496, false},
		// Decimal forms are an explicit dB choice; a low one is taken at face
		// value here (warnGainUnits already skips decimals), so still flagged
		// as too-low when below the floor since it's a real applied gain.
		{"8.7", 87, true},
		{"15.0", 150, false},
		// auto / disabled / zero gain must never warn.
		{"auto", -1, false},
		{"", -1, false},
		{"0", 0, false},
		{"-1", -1, false},
	}
	for _, c := range cases {
		if got := gainLooksTooLow(c.raw, c.tenthDB); got != c.want {
			t.Errorf("gainLooksTooLow(%q, %d) = %v, want %v",
				c.raw, c.tenthDB, got, c.want)
		}
	}
}

func TestFixedGainOnSharedWideband(t *testing.T) {
	cases := []struct {
		name        string
		role        sdr.Role
		numChannels int
		tenthDB     int
		want        bool
	}{
		// The reported case (issue #749): a multi-tap wideband dongle pinned
		// to a fixed gain — fires regardless of how reasonable the level is,
		// because the problem is cross-site dynamic range, not absolute level.
		{"wideband 2 taps, 60 dB", sdr.RoleWideband, 2, 600, true},
		{"wideband 4 taps, 30 dB", sdr.RoleWideband, 4, 300, true},
		{"wideband 5 taps, 49.6 dB", sdr.RoleWideband, 5, 496, true},
		{"wideband 2 taps, gain 0", sdr.RoleWideband, 2, 0, true},
		// auto (tenthDB == -1) is exactly the recommended remedy: never warn.
		{"wideband 4 taps, auto", sdr.RoleWideband, 4, -1, false},
		// A single-tap wideband device has no co-tenants to starve.
		{"wideband 1 tap, fixed", sdr.RoleWideband, 1, 600, false},
		{"wideband 0 taps, fixed", sdr.RoleWideband, 0, 600, false},
		// Non-wideband roles tune their own channel with their own gain, so
		// the shared-front-end starvation can't apply — never warn.
		{"control, fixed", sdr.RoleControl, 4, 600, false},
		{"voice, fixed", sdr.RoleVoice, 4, 600, false},
		{"auto role, fixed", sdr.RoleAuto, 4, 600, false},
	}
	for _, c := range cases {
		if got := fixedGainOnSharedWideband(c.role, c.numChannels, c.tenthDB); got != c.want {
			t.Errorf("%s: fixedGainOnSharedWideband(%v, %d, %d) = %v, want %v",
				c.name, c.role, c.numChannels, c.tenthDB, got, c.want)
		}
	}
}
