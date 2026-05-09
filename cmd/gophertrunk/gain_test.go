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
