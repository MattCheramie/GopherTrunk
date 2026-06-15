package tetra

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// putBits writes v as n bits MSB-first into bits starting at off.
func putBits(bits []byte, off, n int, v uint32) {
	for i := 0; i < n; i++ {
		bits[off+i] = byte((v >> uint(n-1-i)) & 1)
	}
}

func TestParseSyncPDU(t *testing.T) {
	const (
		cc  = 0x2A // 6-bit
		tn  = 2
		fn  = 17
		mn  = 33
		mcc = 0x0CE // 10-bit (e.g. 206)
		mnc = 0x1ABC
	)
	info := make([]byte, 60)
	putBits(info, 4, 6, cc)
	putBits(info, 10, 2, tn)
	putBits(info, 12, 5, fn)
	putBits(info, 17, 6, mn)
	putBits(info, 31, 10, mcc)
	putBits(info, 41, 14, mnc)

	s, ok := ParseSyncPDU(info)
	if !ok {
		t.Fatal("ParseSyncPDU returned false")
	}
	if s.ColourCode != cc || s.TN != tn || s.FN != fn || s.MN != mn || s.MCC != mcc || s.MNC != mnc {
		t.Fatalf("parsed %+v, want cc=%#x tn=%d fn=%d mn=%d mcc=%#x mnc=%#x", s, cc, tn, fn, mn, mcc, mnc)
	}
	want := (uint32(mcc) << 20) | (uint32(mnc) << 6) | uint32(cc)
	if got := s.ExtendedColourCode(); got != want {
		t.Errorf("ExtendedColourCode = %#x, want %#x", got, want)
	}
}

// TestParseSyncPDUThroughBSCHChain round-trips the SYNC PDU through the
// real BSCH FEC chain (EncodeBSCH → DecodeBSCH → ParseSyncPDU), so the
// parser is exercised against the actual channel decoder, not just raw
// bits.
func TestParseSyncPDUThroughBSCHChain(t *testing.T) {
	info := make([]byte, 60)
	putBits(info, 4, 6, 0x15)
	putBits(info, 31, 10, 0x123)
	putBits(info, 41, 14, 0x2345)
	type5 := EncodeBSCH(info)
	if len(type5) != 120 {
		t.Fatalf("EncodeBSCH = %d bits, want 120", len(type5))
	}
	recovered, ok := DecodeBSCH(type5)
	if !ok {
		t.Fatal("DecodeBSCH failed CRC on a clean BSCH")
	}
	s, ok := ParseSyncPDU(recovered)
	if !ok {
		t.Fatal("ParseSyncPDU returned false")
	}
	if s.ColourCode != 0x15 || s.MCC != 0x123 || s.MNC != 0x2345 {
		t.Errorf("parsed cc=%#x mcc=%#x mnc=%#x, want 0x15/0x123/0x2345", s.ColourCode, s.MCC, s.MNC)
	}
}

// TestProcessLearnsColourCodeFromSBBurst feeds a synchronisation-burst
// dibit stream (BSCH SYNC + STS + broadcast + BNCH SYSINFO) straight
// into Process with NO colour code configured, and asserts the decoder
// learns the colour code from the (colour-0) BSCH and then locks on the
// BNCH SYSINFO it descrambles with the learned code.
func TestProcessLearnsColourCodeFromSBBurst(t *testing.T) {
	const (
		cc6        = 0x2D
		mcc uint16 = 0x0CE
		mnc uint16 = 0x1234
		la  uint16 = 0x2B7
	)
	ext := ExtendedColourCode(mcc, mnc, cc6)

	// BSCH SYNC PDU (60 type-1 bits), colour code 0 on the wire.
	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, cc6)
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))
	bschDibits := TetraBitsToDibits(EncodeBSCH(syncInfo))

	// BNCH: MLE SYSINFO carrying the location area, SCH/HD-coded and
	// scrambled with the cell's extended colour code.
	payload := make([]byte, 11)
	payload[3] = byte((la >> 6) & 0xFF)
	payload[4] = byte((la & 0x3F) << 2)
	pdu := PDU{Disc: DiscMLE, Type: uint8(MLESystemInfo), Payload: payload}
	bnchDibits := TetraBitsToDibits(EncodeSCHHD(pduToType1Bits(pdu, 124), ext))

	// Assemble the SB burst at the spec offsets and repeat it.
	var burst []uint8
	burst = append(burst, bschDibits...)           // [0,60)
	burst = append(burst, SyncTrainingDibits()...) // [60,79) STS
	burst = append(burst, make([]uint8, 15)...)    // [79,94) broadcast
	burst = append(burst, bnchDibits...)           // [94,202) BNCH

	stream := make([]uint8, 30) // lead-in so the first STS has look-back room
	for r := 0; r < 4; r++ {
		stream = append(stream, burst...)
		stream = append(stream, make([]uint8, 40)...) // inter-burst gap
	}

	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), SystemName: "Sys", FrequencyHz: 412_000_000})
	cc.SetChannelCoding(ChannelCodingOn)
	// Deliberately do NOT SetColourCode — the decoder must learn it.

	cc.Process(stream, 0)

	if got := cc.ColourCode(); got != ext {
		t.Errorf("learned colour code = %#x, want %#x", got, ext)
	}
	var locked bool
	var ls LockState
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				locked = true
				ls, _ = ev.Payload.(LockState)
			}
			continue
		default:
		}
		break
	}
	if !locked {
		t.Fatal("no cc.locked from the SB burst (colour code never learned / BNCH never decoded)")
	}
	if ls.MCC != mcc || ls.MNC != mnc {
		t.Errorf("lock MCC/MNC = %#x/%#x, want %#x/%#x", ls.MCC, ls.MNC, mcc, mnc)
	}
	if ls.LocationArea != la {
		t.Errorf("lock LocationArea = %#x, want %#x (BNCH SYSINFO not decoded with the learned colour)", ls.LocationArea, la)
	}
}

// TestProcessLearnsColourCodeFromSBBurstUnderRotation is the live-path
// counterpart to TestRecoverColourCodeUnderRotation: it drives the full
// ControlChannel.Process state machine with a synchronisation burst whose
// dibit stream is globally rotated (the residual-CFO orientation real air
// always carries), and asserts the SB detector still fires, the colour code
// is learned from the BSCH, and the BNCH SYSINFO descrambles to a lock.
// Before the per-rotation STS detectors landed, a non-zero rotation slipped
// straight past the single fixed-orientation correlator and never locked.
func TestProcessLearnsColourCodeFromSBBurstUnderRotation(t *testing.T) {
	const (
		cc6        = 0x2D
		mcc uint16 = 0x0CE
		mnc uint16 = 0x1234
		la  uint16 = 0x2B7
	)
	ext := ExtendedColourCode(mcc, mnc, cc6)

	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, cc6)
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))
	bschDibits := TetraBitsToDibits(EncodeBSCH(syncInfo))

	payload := make([]byte, 11)
	payload[3] = byte((la >> 6) & 0xFF)
	payload[4] = byte((la & 0x3F) << 2)
	pdu := PDU{Disc: DiscMLE, Type: uint8(MLESystemInfo), Payload: payload}
	bnchDibits := TetraBitsToDibits(EncodeSCHHD(pduToType1Bits(pdu, 124), ext))

	var burst []uint8
	burst = append(burst, bschDibits...)
	burst = append(burst, SyncTrainingDibits()...)
	burst = append(burst, make([]uint8, 15)...)
	burst = append(burst, bnchDibits...)

	base := make([]uint8, 30)
	for r := 0; r < 4; r++ {
		base = append(base, burst...)
		base = append(base, make([]uint8, 40)...)
	}

	for rot := uint8(1); rot < 4; rot++ {
		stream := rotateDibits(base, rot)

		bus := events.NewBus(16)
		sub := bus.Subscribe()
		cc := New(Options{Bus: bus, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), SystemName: "Sys", FrequencyHz: 412_000_000})
		cc.SetChannelCoding(ChannelCodingOn)
		cc.Process(stream, 0)

		if got := cc.ColourCode(); got != ext {
			t.Errorf("rot=%d: learned colour code = %#x, want %#x", rot, got, ext)
		}
		var locked bool
		for {
			select {
			case ev := <-sub.C:
				if ev.Kind == events.KindCCLocked {
					locked = true
				}
				continue
			default:
			}
			break
		}
		if !locked {
			t.Errorf("rot=%d: no cc.locked from the rotated SB burst", rot)
		}
		sub.Close()
		bus.Close()
	}
}
