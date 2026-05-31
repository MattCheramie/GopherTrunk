package main

import "testing"

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
