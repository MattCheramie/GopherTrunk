package conventional

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// #1184 on-air follow-up: the reporter's Kenwoods pinned the polarity
// convention (D025N → nrz_inverted=false, D025I → true), so a DCS gate
// now accepts only the configured sense. Before, both senses always
// opened, so a "023" gate also opened on a radio set to 023I — which
// is the same bit pattern on air as 047N.

func TestParseDCSPolarity(t *testing.T) {
	for in, want := range map[string]DCSPolarity{
		"": DCSPolarityNormal, "normal": DCSPolarityNormal, "N": DCSPolarityNormal,
		"inverted": DCSPolarityInverted, "i": DCSPolarityInverted, " Inverted ": DCSPolarityInverted,
		"both": DCSPolarityBoth,
	} {
		got, err := ParseDCSPolarity(in)
		if err != nil || got != want {
			t.Errorf("ParseDCSPolarity(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseDCSPolarity("reverse"); err == nil {
		t.Error(`ParseDCSPolarity("reverse") accepted`)
	}
}

// The default (normal) gate must stay shut on the same code sent
// inverted, and flag it as an N/I mismatch; and vice versa.
func TestDCSGateAcceptsOnlyConfiguredPolarity(t *testing.T) {
	const rate = 2_400_000
	for _, tc := range []struct {
		name     string
		pol      DCSPolarity
		sendInv  bool
		wantOpen bool
	}{
		{"default-normal/sent-N", DCSPolarityNormal, false, true},
		{"default-normal/sent-I", DCSPolarityNormal, true, false},
		{"inverted/sent-I", DCSPolarityInverted, true, true},
		{"inverted/sent-N", DCSPolarityInverted, false, false},
		{"both/sent-N", DCSPolarityBoth, false, true},
		{"both/sent-I", DCSPolarityBoth, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDCSDetector(DCSConfig{SampleHz: rate, Code: "023", Polarity: tc.pol})
			x := synthDCSOnAir(refDCSWord(0o023), tc.sendInv, rate, 350, 600, 25, 1.5, 11)
			always, ever := dcsPresentAfter(d, x, rate, dcsMinDwell.Seconds())
			if tc.wantOpen && !always {
				t.Fatal("gate not continuously open on the configured polarity")
			}
			if !tc.wantOpen && ever {
				t.Fatal("gate opened on the opposite polarity")
			}
			if got := d.TakeOppositeHeard(); got != !tc.wantOpen {
				t.Errorf("TakeOppositeHeard() = %v, want %v", got, !tc.wantOpen)
			}
			if d.TakeOppositeHeard() {
				t.Error("TakeOppositeHeard did not clear")
			}
		})
	}
}

// 023I and 047N are one bit pattern on air (as are 023N and 047I, which
// no receiver can tell apart). A plain "023" gate used to open on 047N;
// with the default normal polarity it must not, and a different code
// must never raise the mismatch hint.
func TestDCSDefaultGateIgnoresAliasedCode(t *testing.T) {
	const rate = 2_400_000
	d := NewDCSDetector(DCSConfig{SampleHz: rate, Code: "023"})
	x := synthDCSOnAir(refDCSWord(0o047), false, rate, 350, 600, 25, 1.5, 12)
	if _, ever := dcsPresentAfter(d, x, rate, 0); ever {
		t.Error("default 023 gate opened on 047N (= 023I)")
	}
	d = NewDCSDetector(DCSConfig{SampleHz: rate, Code: "023"})
	x = synthDCSOnAir(refDCSWord(0o754), false, rate, 350, 600, 25, 1.5, 13)
	dcsPresentAfter(d, x, rate, 0)
	if d.TakeOppositeHeard() {
		t.Error("a different code raised the opposite-polarity hint")
	}
}

func TestValidateToneRejectsBadDCSPolarity(t *testing.T) {
	if err := validateTone(ToneConfig{Mode: "dcs", DCSCode: "023", DCSPolarity: "inverted"}); err != nil {
		t.Errorf("valid polarity rejected: %v", err)
	}
	if err := validateTone(ToneConfig{Mode: "dcs", DCSCode: "023", DCSPolarity: "reverse"}); err == nil {
		t.Error("bad dcs_polarity accepted")
	}
}

func TestBuildDetectorPassesDCSPolarity(t *testing.T) {
	d, ok := buildDetector(ToneConfig{Mode: "dcs", DCSCode: "023", DCSPolarity: "inverted"}, 2_400_000, nil).(*DCSDetector)
	if !ok || d.Polarity() != DCSPolarityInverted {
		t.Fatalf("buildDetector polarity = %v, want inverted", d)
	}
}

func TestWarnDCSOppositePolarityOncePerChannel(t *testing.T) {
	var buf bytes.Buffer
	s := &Scanner{
		log:               slog.New(slog.NewTextHandler(&buf, nil)),
		dcsPolarityWarned: make(map[*DCSDetector]bool),
	}
	d := NewDCSDetector(DCSConfig{SampleHz: 48_000, Code: "025"})
	ch := Channel{FrequencyHz: 447_100_000, Label: "UHF Simplex"}
	s.warnDCSOppositePolarity(0, ch, d)
	s.warnDCSOppositePolarity(0, ch, d)
	out := buf.String()
	if n := strings.Count(out, "opposite polarity"); n != 1 {
		t.Fatalf("WARN count = %d, want 1:\n%s", n, out)
	}
	if !strings.Contains(out, "heard=inverted") || !strings.Contains(out, "dcs_polarity=normal") {
		t.Errorf("WARN missing polarity fields:\n%s", out)
	}
}
