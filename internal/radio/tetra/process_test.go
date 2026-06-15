package tetra

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestRecoverColourCode builds a synchronisation burst (BSCH + STS) carrying a
// known cell colour code and confirms RecoverColourCode pulls the extended
// colour code back out with no prior configuration — the path blind identify
// relies on to descramble SCH/HD (issue #648).
func TestRecoverColourCode(t *testing.T) {
	const (
		cc6        = 0x2D
		mcc uint16 = 0x0CE
		mnc uint16 = 0x1234
	)
	want := ExtendedColourCode(mcc, mnc, cc6)

	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, cc6)
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))
	bschDibits := TetraBitsToDibits(EncodeBSCH(syncInfo))

	var stream []uint8
	stream = append(stream, bschDibits...)           // [0,60) BSCH
	stream = append(stream, SyncTrainingDibits()...) // [60,79) STS
	stream = append(stream, make([]uint8, 40)...)    // trailing pad

	got, ok := RecoverColourCode(stream)
	if !ok {
		t.Fatal("RecoverColourCode found no BSCH in a clean SB burst")
	}
	if got != want {
		t.Errorf("recovered colour code = %#x, want %#x", got, want)
	}

	// A stream with no synchronisation burst yields nothing.
	if _, ok := RecoverColourCode(make([]uint8, 200)); ok {
		t.Error("RecoverColourCode reported a colour code on an empty stream")
	}
}

// TestRecoverColourCodeUnderRotation reproduces the real-air failure mode that
// the rotation-0 synthetic fixtures masked: π/4-DQPSK carries data in the
// differential phase, so a residual carrier offset adds a constant 90°/dibit
// term that cyclically rotates the entire demodulated stream by an unknown
// 0..3. Live captures essentially never sit at rotation 0 (the supplied 468.5
// MHz TETRA captures landed at rotation 1), so a rotation-blind STS correlator
// never fires and no colour code is recovered — exactly what was observed on
// air. RecoverColourCode must de-rotate and recover under every orientation.
func TestRecoverColourCodeUnderRotation(t *testing.T) {
	const (
		cc6        = 0x2D
		mcc uint16 = 0x0CE
		mnc uint16 = 0x1234
	)
	want := ExtendedColourCode(mcc, mnc, cc6)

	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, cc6)
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))
	bschDibits := TetraBitsToDibits(EncodeBSCH(syncInfo))

	var base []uint8
	base = append(base, make([]uint8, 23)...)    // leading filler
	base = append(base, bschDibits...)           // BSCH
	base = append(base, SyncTrainingDibits()...) // STS
	base = append(base, make([]uint8, 40)...)    // trailing pad

	for rot := uint8(0); rot < 4; rot++ {
		stream := rotateDibits(base, rot)
		got, ok := RecoverColourCode(stream)
		if !ok {
			t.Errorf("rot=%d: RecoverColourCode found no BSCH in a rotated SB burst", rot)
			continue
		}
		if got != want {
			t.Errorf("rot=%d: recovered colour code = %#x, want %#x", rot, got, want)
		}
	}
}

// TestProcessLocksOnSystemBroadcastAfterSync builds a dibit stream
// of 50 padding dibits + 38 normal training-sequence dibits + 48
// dibits whose raw bits decode to an MLE-SYSINFO PDU with known
// MCC / MNC / LocationArea, and confirms a KindCCLocked event
// lands on the bus.
func TestProcessLocksOnSystemBroadcastAfterSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 412_062_500,
	})

	// SystemBroadcast layout: 10-bit MCC + 14-bit MNC + 14-bit LA
	// packed MSB-first into 5 bytes (38 bits + 2 trailing pad bits).
	const (
		mcc uint16 = 0x123 // 10-bit
		mnc uint16 = 0x1FF // 14-bit
		la  uint16 = 0x0AB // 14-bit
	)
	payload := make([]byte, 11) // pad to 11 so VoiceGrant-sized PDUs also fit
	payload[0] = byte(mcc >> 2)
	payload[1] = byte((mcc&0x3)<<6) | byte((mnc>>8)&0x3F)
	payload[2] = byte(mnc & 0xFF)
	payload[3] = byte(la >> 6)
	payload[4] = byte((la & 0x3F) << 2)

	pdu := PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: payload,
	}
	pduBytes := AssemblePDU(pdu)
	pduBits := framing.UnpackBitsMSB(pduBytes, 96)
	pduDibits := TetraBitsToDibits(pduBits)
	if len(pduDibits) != 48 {
		t.Fatalf("PDU dibits = %d, want 48", len(pduDibits))
	}

	stream := make([]uint8, 50)
	stream = append(stream, NormalSyncDibits()...)
	stream = append(stream, pduDibits...)

	cc.Process(stream, 0)

	var sawLock bool
	var ls LockState
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				ls, _ = ev.Payload.(LockState)
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("Process did not publish a KindCCLocked")
				return
			}
			if ls.FrequencyHz != 412_062_500 {
				t.Errorf("LockState.FrequencyHz = %d", ls.FrequencyHz)
			}
			if ls.MCC != mcc {
				t.Errorf("LockState.MCC = %#x, want %#x", ls.MCC, mcc)
			}
			if ls.MNC != mnc {
				t.Errorf("LockState.MNC = %#x, want %#x", ls.MNC, mnc)
			}
			if ls.LocationArea != la {
				t.Errorf("LockState.LocationArea = %#x, want %#x", ls.LocationArea, la)
			}
			return
		}
	}
}

func TestProcessHandlesPDUSpanningCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	payload := make([]byte, 11)
	payload[0] = 0x40 // MCC bits — anything non-zero so SystemBroadcast parses
	pdu := PDU{Disc: DiscMLE, Type: uint8(MLESystemInfo), Payload: payload}
	pduBits := framing.UnpackBitsMSB(AssemblePDU(pdu), 96)
	pduDibits := TetraBitsToDibits(pduBits)

	chunk1 := make([]uint8, 50)
	chunk1 = append(chunk1, NormalSyncDibits()...)
	cc.Process(chunk1, 0)
	cc.Process(pduDibits, len(chunk1))

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("Process did not publish a KindCCLocked across the chunk boundary")
			}
			return
		}
	}
}

func TestProcessIgnoresGarbageWithoutSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	garbage := make([]uint8, 2000)
	for i := range garbage {
		garbage[i] = uint8(i % 4)
	}
	cc.Process(garbage, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event from garbage stream: %v", ev.Kind)
	default:
	}
}

func TestSyncDetectorReset(t *testing.T) {
	det := NewSyncDetector(NormalSyncDibits(), 0)
	junk := make([]uint8, 100)
	det.Process(nil, junk, 0)
	det.Reset()
	if det.primed != 0 {
		t.Errorf("post-Reset primed = %d, want 0", det.primed)
	}
	if det.pos != 0 {
		t.Errorf("post-Reset pos = %d, want 0", det.pos)
	}
}
