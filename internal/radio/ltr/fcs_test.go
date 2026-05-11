package ltr

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// makeStatusWithValidFCS builds a Status with the trunking-relevant
// fields populated and Status.FCS set to the CRC-7 of the
// resulting 24-bit message vector. Used by FCSOn tests.
func makeStatusWithValidFCS() Status {
	s := Status{
		Sync:    true,
		Area:    1,
		Group:   true,
		Channel: 5,
		Home:    7,
		GroupID: 42,
		Free:    3,
	}
	s.FCS = uint16(ComputeStatusFCS(s))
	return s
}

// TestFCSOnAcceptsValidChecksum: a Status whose FCS field carries
// the correctly-computed CRC-7 must pass the Ingest check under
// SetFCSMode(FCSOn) and reach the lock / grant path.
func TestFCSOnAcceptsValidChecksum(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetFCSMode(FCSOn)

	cc.Ingest(makeStatusWithValidFCS())

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("FCSOn dropped a Status with a valid CRC")
			}
			return
		}
	}
}

// TestFCSOnDropsCorruptedChecksum: flipping a bit in the FCS
// trailer must cause FCSOn to drop the frame.
func TestFCSOnDropsCorruptedChecksum(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetFCSMode(FCSOn)

	s := makeStatusWithValidFCS()
	s.FCS ^= 1 // corrupt the LSB
	cc.Ingest(s)

	select {
	case ev := <-sub.C:
		t.Errorf("FCSOn accepted a Status with a bad CRC: %v", ev.Kind)
	default:
	}
}

// TestFCSOnDropsCorruptedMessage: flipping a bit in one of the
// FCS-protected fields (Channel, Home, GroupID, Free) must also
// cause FCSOn to drop the frame (the precomputed FCS no longer
// matches the recomputed CRC).
func TestFCSOnDropsCorruptedMessage(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetFCSMode(FCSOn)

	s := makeStatusWithValidFCS()
	s.Channel ^= 1 // flip a single Channel bit
	cc.Ingest(s)

	select {
	case ev := <-sub.C:
		t.Errorf("FCSOn accepted a Status with a corrupted message field: %v", ev.Kind)
	default:
	}
}

// TestFCSOffIgnoresChecksum: FCSOff (the default) must NOT verify
// the CRC — a Status with a bogus FCS field still drives the
// state machine.
func TestFCSOffIgnoresChecksum(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	// Default mode is FCSOff; don't call SetFCSMode.

	s := makeStatusWithValidFCS()
	s.FCS = 0x7F // bogus checksum
	cc.Ingest(s)

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("FCSOff dropped a frame despite bypassing the FCS check")
			}
			return
		}
	}
}

func TestSetFCSModeDefault(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(1)})
	if cc.fcsMode != FCSOff {
		t.Errorf("default fcsMode = %v, want FCSOff", cc.fcsMode)
	}
	if got := cc.FCSMode(); got != FCSOff {
		t.Errorf("FCSMode() = %v, want FCSOff", got)
	}
	cc.SetFCSMode(FCSOn)
	if cc.fcsMode != FCSOn {
		t.Errorf("SetFCSMode(FCSOn) did not take effect")
	}
	if got := cc.FCSMode(); got != FCSOn {
		t.Errorf("FCSMode() = %v, want FCSOn", got)
	}
	cc.SetFCSMode(FCSOff)
	if cc.fcsMode != FCSOff {
		t.Errorf("SetFCSMode(FCSOff) did not take effect")
	}
}

// TestParseFCSMode covers the config-string → FCSMode mapping the
// ccdecoder connector uses to translate the `ltr_fcs_mode` YAML
// field into a SetFCSMode call.
func TestParseFCSMode(t *testing.T) {
	cases := []struct {
		in   string
		want FCSMode
		ok   bool
	}{
		{"", FCSOff, true},
		{"off", FCSOff, true},
		{"OFF", FCSOff, true},
		{"false", FCSOff, true},
		{"0", FCSOff, true},
		{"on", FCSOn, true},
		{"ON", FCSOn, true},
		{"true", FCSOn, true},
		{"1", FCSOn, true},
		{" on ", FCSOn, true}, // whitespace tolerated
		{"nonsense", FCSOff, false},
	}
	for _, tc := range cases {
		got, ok := ParseFCSMode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseFCSMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestParseManchesterMode covers the config-string →
// ManchesterDecodeMode mapping for `ltr_manchester_mode`.
func TestParseManchesterMode(t *testing.T) {
	cases := []struct {
		in   string
		want ManchesterDecodeMode
		ok   bool
	}{
		{"", ManchesterOff, true},
		{"off", ManchesterOff, true},
		{"nrz", ManchesterOff, true},
		{"NRZ", ManchesterOff, true},
		{"strict", ManchesterStrict, true},
		{"Strict", ManchesterStrict, true},
		{"soft", ManchesterSoft, true},
		{"on", ManchesterSoft, true},
		{"Soft", ManchesterSoft, true},
		{" strict ", ManchesterStrict, true},
		{"nonsense", ManchesterOff, false},
	}
	for _, tc := range cases {
		got, ok := ParseManchesterMode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseManchesterMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestManchesterModeAccessor mirrors TestSetFCSModeDefault for the
// Manchester mode getter.
func TestManchesterModeAccessor(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(1)})
	if got := cc.ManchesterMode(); got != ManchesterOff {
		t.Errorf("default ManchesterMode() = %v, want ManchesterOff", got)
	}
	cc.SetManchesterMode(ManchesterSoft)
	if got := cc.ManchesterMode(); got != ManchesterSoft {
		t.Errorf("ManchesterMode() = %v, want ManchesterSoft", got)
	}
}

func TestComputeStatusFCSMatchesPrimitive(t *testing.T) {
	// Two distinct Status configurations should produce distinct
	// CRC values (sanity check that the message-bit mapping
	// actually varies across fields).
	s1 := Status{Group: true, Channel: 1, Home: 1, GroupID: 1, Free: 1}
	s2 := Status{Group: false, Channel: 1, Home: 1, GroupID: 1, Free: 1}
	if ComputeStatusFCS(s1) == ComputeStatusFCS(s2) {
		t.Errorf("CRC didn't change when the Group bit flipped: both = %#x", ComputeStatusFCS(s1))
	}
}
