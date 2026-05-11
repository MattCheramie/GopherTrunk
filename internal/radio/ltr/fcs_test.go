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
	cc.SetFCSMode(FCSOn)
	if cc.fcsMode != FCSOn {
		t.Errorf("SetFCSMode(FCSOn) did not take effect")
	}
	cc.SetFCSMode(FCSOff)
	if cc.fcsMode != FCSOff {
		t.Errorf("SetFCSMode(FCSOff) did not take effect")
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
