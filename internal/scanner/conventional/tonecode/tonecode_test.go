package tonecode

import "testing"

func TestOverlap(t *testing.T) {
	ctcss := func(hz float64) Gate { return Gate{Mode: "ctcss", CTCSSHz: hz} }
	dcs := func(code, pol string) Gate { return Gate{Mode: "dcs", DCSCode: code, DCSPolarity: pol} }
	for _, tc := range []struct {
		name string
		a, b Gate
		want bool
	}{
		{"untoned opens on everything", Gate{}, ctcss(100), true},
		{"none is untoned", Gate{Mode: "none"}, dcs("023", ""), true},
		{"same CTCSS", ctcss(123), ctcss(123), true},
		// 150.0 / 151.4 Hz are 1.4 Hz apart: the detector cannot separate them.
		{"150.0 and 151.4 Hz", ctcss(150.0), ctcss(151.4), true},
		{"distinct CTCSS", ctcss(100), ctcss(141.3), false},
		{"adjacent EIA tones 2.3 Hz apart", ctcss(159.8), ctcss(162.2), false},
		{"CTCSS never meets DCS", ctcss(100), dcs("023", ""), false},
		{"same DCS", dcs("023", "normal"), dcs("023", "n"), true},
		// 023N and 047I are the same bits on air (a rotation of each other's
		// complement), so no detector can tell them apart.
		{"DCS alias 023N = 047I", dcs("023", ""), dcs("047", "inverted"), true},
		{"DCS alias 023I = 047N", dcs("023", "i"), dcs("047", ""), true},
		{"023N and 023I are different words", dcs("023", "normal"), dcs("023", "inverted"), false},
		{"both overlaps either sense", dcs("023", "both"), dcs("023", "inverted"), true},
		{"distinct DCS codes", dcs("023", ""), dcs("754", ""), false},
	} {
		got, err := Overlap(tc.a, tc.b)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s: Overlap = %v, want %v", tc.name, got, tc.want)
		}
		if rev, _ := Overlap(tc.b, tc.a); rev != got {
			t.Errorf("%s: Overlap is not symmetric", tc.name)
		}
	}
	if _, err := Overlap(dcs("09x", ""), dcs("023", "")); err == nil {
		t.Error("bad DCS code: no error")
	}
}

// Distinct DCS codes are Golay(23,12) codewords at least 7 bits apart, so
// across every pair of normal-sense codes only aliases overlap: each code
// overlaps exactly the codes whose words are rotations of its own.
func TestDistinctDCSCodesNeverOverlapUnlessAliases(t *testing.T) {
	codes := []string{"023", "047", "054", "065", "071", "114", "115", "116", "125", "131", "754"}
	for i, a := range codes {
		for _, b := range codes[i+1:] {
			ov, err := Overlap(Gate{Mode: "dcs", DCSCode: a}, Gate{Mode: "dcs", DCSCode: b})
			if err != nil {
				t.Fatal(err)
			}
			wa, _ := DCSCodeword(a)
			wb, _ := DCSCodeword(b)
			rot := false
			for k, r := range DCSRotations(wa) {
				if k%2 == 0 && r == wb {
					rot = true
				}
			}
			if ov != rot {
				t.Errorf("DCS %s / %s: overlap %v, rotation of each other %v", a, b, ov, rot)
			}
		}
	}
}

func TestGateNames(t *testing.T) {
	for _, tc := range []struct {
		g       Gate
		str, id string
	}{
		{Gate{}, "", ""},
		{Gate{Mode: "ctcss", CTCSSHz: 123}, "CTCSS 123.0 Hz", "ctcss123.0"},
		{Gate{Mode: "dcs", DCSCode: "023"}, "DCS 023N", "dcs023n"},
		{Gate{Mode: "dcs", DCSCode: "023", DCSPolarity: "inverted"}, "DCS 023I", "dcs023i"},
		{Gate{Mode: "dcs", DCSCode: "023", DCSPolarity: "both"}, "DCS 023 (both)", "dcs023b"},
	} {
		if s := tc.g.String(); s != tc.str {
			t.Errorf("%+v String = %q, want %q", tc.g, s, tc.str)
		}
		if id := tc.g.ID(); id != tc.id {
			t.Errorf("%+v ID = %q, want %q", tc.g, id, tc.id)
		}
	}
}
