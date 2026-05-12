package nxdn

import (
	"encoding/binary"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestProcessViterbiSpecDecodesEncodedCACFrame builds a dibit
// stream matching the NXDN-TS-1-A §4.6 RCCH outbound layout (LICH
// 16 bits + CAC 300 bits) and confirms Process with ViterbiSpec
// recovers a SITE_INFO RCCH message through the full
// deinterleave + depuncture + K=5 Viterbi + outer-CRC chain.
//
// Layout post-sync (158 dibits = postSyncDibitsViterbiSpec):
//
//	dibits  0..8   LICH wire (RFCh = Control)
//	dibits  8..158 CAC, spec-encoded (150 dibits = 300 channel bits)
func TestProcessViterbiSpecDecodesEncodedCACFrame(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, slog.Default(), 851_062_500, Rate9600)
	cc.SetViterbiMode(ViterbiSpec)

	lichInfo := AssembleLICH(LICH{RFCh: RFChControl})
	lichWire := EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)
	if len(lichDibits) != 8 {
		t.Fatalf("LICH wire dibits = %d, want 8", len(lichDibits))
	}

	// Build a 9-byte L3 prefix that the legacy ParseCAC will read:
	// 1 byte RCCH type + 8 bytes payload. The trailing 16 inner-CRC
	// bits of the existing 11-byte block aren't part of the spec
	// layout — Process re-synthesizes them locally so ParseCAC's
	// sentinel passes after DecodeCACChannel verifies the outer CRC.
	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA)
	binary.BigEndian.PutUint16(payload[2:4], 0xBEEF) // site id
	binary.BigEndian.PutUint16(payload[4:6], 0x0042) // system id
	l3 := make([]byte, 9)
	l3[0] = byte(RCCHSITEINFO)
	copy(l3[1:9], payload[:])

	// Build the 155-bit spec info block: 8 bits SR (zero) + 72 bits
	// of the L3 prefix above + 72 bits of zero L3 trailer + 3 bits
	// of Null. Total 155 ✓.
	info := make([]byte, CACInfoBits)
	// SR: leave zero.
	l3Bits := framing.UnpackBitsMSB(l3, 72)
	copy(info[8:8+72], l3Bits)
	// Remaining 144-72 = 72 L3 trailer bits and 3 Null bits left
	// zero by make().

	channel := EncodeCACChannel(info)
	if len(channel) != CACChannelBits {
		t.Fatalf("EncodeCACChannel returned %d channel bits, want %d", len(channel), CACChannelBits)
	}
	cacDibits := framing.BitsToDibits(channel)
	if len(cacDibits) != 150 {
		t.Fatalf("CAC dibits = %d, want 150", len(cacDibits))
	}

	stream := make([]uint8, 30)
	stream = append(stream, FSWDibitsOutbound...)
	stream = append(stream, lichDibits...)
	stream = append(stream, cacDibits...)

	cc.Process(stream, 0)

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				ls, ok := ev.Payload.(LockState)
				if !ok {
					t.Fatalf("CCLocked payload type = %T, want LockState", ev.Payload)
				}
				if ls.SiteID != 0xBEEF {
					t.Errorf("LockState.SiteID = %#x, want 0xBEEF", ls.SiteID)
				}
				if ls.SystemID != 0x0042 {
					t.Errorf("LockState.SystemID = %#x, want 0x0042", ls.SystemID)
				}
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("ViterbiSpec did not publish KindCCLocked for spec-encoded SITE_INFO frame")
			}
			return
		}
	}
}

// TestProcessViterbiSpecRejectsHeavilyCorruptedFrame: 60 adjacent
// bit flips in the channel-bit stream defeat the inner Viterbi;
// the outer CRC fires; no event reaches the bus.
func TestProcessViterbiSpecRejectsHeavilyCorruptedFrame(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, slog.Default(), 851_062_500, Rate9600)
	cc.SetViterbiMode(ViterbiSpec)

	lichInfo := AssembleLICH(LICH{RFCh: RFChControl})
	lichWire := EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)

	info := make([]byte, CACInfoBits)
	info[8] = byte(RCCHSITEINFO) & 1 // any non-zero pattern
	channel := EncodeCACChannel(info)

	// Heavy corruption: 60 adjacent flips guaranteed to overwhelm
	// the Viterbi corrector even with the 25×12 interleaver
	// spreading errors across 5 independent decoder steps each.
	for i := 100; i < 160; i++ {
		channel[i] ^= 1
	}
	cacDibits := framing.BitsToDibits(channel)

	stream := make([]uint8, 30)
	stream = append(stream, FSWDibitsOutbound...)
	stream = append(stream, lichDibits...)
	stream = append(stream, cacDibits...)

	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind == events.KindCCLocked || ev.Kind == events.KindGrant {
			t.Errorf("ViterbiSpec accepted heavily-corrupted frame: %v", ev.Kind)
		}
	default:
	}
}

// TestParseViterbiModeRecognisesSpec covers the new "spec" string
// the ccdecoder connector forwards from `nxdn_viterbi_mode: spec`.
func TestParseViterbiModeRecognisesSpec(t *testing.T) {
	cases := []struct {
		in   string
		want ViterbiMode
		ok   bool
	}{
		{"spec", ViterbiSpec, true},
		{"SPEC", ViterbiSpec, true},
		{" spec ", ViterbiSpec, true},
	}
	for _, c := range cases {
		got, ok := ParseViterbiMode(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseViterbiMode(%q) = (%v, %v), want (%v, %v)",
				c.in, got, ok, c.want, c.ok)
		}
	}
}
