package nxdn

import (
	"encoding/binary"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestProcessViterbiOnDecodesEncodedCACFrame builds a dibit stream
// that mimics on-air NXDN CAC framing with the K=5 ½-rate
// convolutional encoder applied to the 88 CAC information bits +
// 4 zero tail bits, and confirms Process with SetViterbiMode(
// ViterbiOn) recovers the CAC and publishes cc.locked.
//
// Layout (post-sync):
//
//	dibits  0..8    LICH wire (RFCh = Control)
//	dibits  8..40   SACCH (junk; skipped)
//	dibits  40..132 CAC K=5 ½-rate-encoded (92 dibits = 184 wire bits)
//
// = 132 post-sync dibits, matching postSyncDibitsViterbi.
func TestProcessViterbiOnDecodesEncodedCACFrame(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, slog.Default(), 851_062_500, Rate9600)
	cc.SetViterbiMode(ViterbiOn)

	lichInfo := AssembleLICH(LICH{RFCh: RFChControl})
	lichWire := EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)
	if len(lichDibits) != 8 {
		t.Fatalf("LICH wire dibits = %d, want 8", len(lichDibits))
	}

	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA)
	binary.BigEndian.PutUint16(payload[2:4], 0x1234)
	binary.BigEndian.PutUint16(payload[4:6], 0x5678)
	cacBytes := AssembleCAC(CACMessage{
		Type:    RCCHSITEINFO,
		Payload: payload,
	})
	cacBits := framing.UnpackBitsMSB(cacBytes, 88)

	// Append 4 zero tail bits to flush the encoder, then K=5
	// ½-rate convolutional encode → 184 channel bits = 92 dibits.
	source := make([]byte, 92)
	copy(source, cacBits)
	channelBits := framing.EncodeK5(source)
	if len(channelBits) != 184 {
		t.Fatalf("EncodeK5 produced %d bits, want 184", len(channelBits))
	}
	encodedCACDibits := framing.BitsToDibits(channelBits)
	if len(encodedCACDibits) != 92 {
		t.Fatalf("encoded CAC dibits = %d, want 92", len(encodedCACDibits))
	}

	stream := make([]uint8, 30)
	stream = append(stream, FSWDibitsOutbound...)
	stream = append(stream, lichDibits...)
	stream = append(stream, make([]uint8, 32)...) // SACCH junk
	stream = append(stream, encodedCACDibits...)

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
				if ls.SiteID != 0x1234 {
					t.Errorf("LockState.SiteID = %#x, want 0x1234", ls.SiteID)
				}
				if ls.SystemID != 0x5678 {
					t.Errorf("LockState.SystemID = %#x, want 0x5678", ls.SystemID)
				}
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("ViterbiOn Process did not publish a KindCCLocked for an encoded frame")
			}
			return
		}
	}
}

// TestProcessViterbiOnRejectsRawCACFrame confirms that a stream
// formatted for ViterbiOff (raw CAC bits on the wire) does NOT
// pass the CRC under ViterbiOn — because the K=5 decoder treats
// the wire bits as convolution-encoded and produces garbage that
// fails the CAC CRC-CCITT-16. The two modes are not interchangeable.
func TestProcessViterbiOnRejectsRawCACFrame(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, slog.Default(), 851_062_500, Rate9600)
	cc.SetViterbiMode(ViterbiOn)

	lichInfo := AssembleLICH(LICH{RFCh: RFChControl})
	lichWire := EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)

	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA)
	binary.BigEndian.PutUint16(payload[2:4], 0x1234)
	cacBytes := AssembleCAC(CACMessage{
		Type:    RCCHSITEINFO,
		Payload: payload,
	})
	cacBits := framing.UnpackBitsMSB(cacBytes, 88)

	// Pack the raw CAC bits into 44 dibits + 48 junk dibits to fill
	// the 92-dibit encoded-CAC window. ViterbiOn will run K=5 over
	// these 184 wire bits and produce non-CAC output that fails CRC.
	rawCACDibits := framing.BitsToDibits(cacBits)
	encodedSlot := make([]uint8, 92)
	copy(encodedSlot, rawCACDibits)

	stream := make([]uint8, 30)
	stream = append(stream, FSWDibitsOutbound...)
	stream = append(stream, lichDibits...)
	stream = append(stream, make([]uint8, 32)...)
	stream = append(stream, encodedSlot...)

	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind == events.KindCCLocked {
			t.Errorf("ViterbiOn Process accepted a raw (un-encoded) CAC frame as locked")
		}
	default:
	}
}

// TestSetViterbiModeRetainsViterbiOffDefault confirms the zero-value
// ControlChannel uses the legacy raw-CAC path.
func TestSetViterbiModeRetainsViterbiOffDefault(t *testing.T) {
	cc := NewControlChannel(nil, slog.Default(), 0, Rate9600)
	if cc.viterbiMode != ViterbiOff {
		t.Errorf("default viterbiMode = %v, want ViterbiOff", cc.viterbiMode)
	}
	if got := cc.ViterbiMode(); got != ViterbiOff {
		t.Errorf("ViterbiMode() = %v, want ViterbiOff", got)
	}
	cc.SetViterbiMode(ViterbiOn)
	if cc.viterbiMode != ViterbiOn {
		t.Errorf("SetViterbiMode(ViterbiOn) did not take effect")
	}
	if got := cc.ViterbiMode(); got != ViterbiOn {
		t.Errorf("ViterbiMode() = %v, want ViterbiOn", got)
	}
	cc.SetViterbiMode(ViterbiOff)
	if cc.viterbiMode != ViterbiOff {
		t.Errorf("SetViterbiMode(ViterbiOff) did not take effect")
	}
}

// TestParseViterbiMode covers the config-string → ViterbiMode
// mapping the ccdecoder connector uses to translate the
// `nxdn_viterbi_mode` YAML field into a SetViterbiMode call.
func TestParseViterbiMode(t *testing.T) {
	cases := []struct {
		in   string
		want ViterbiMode
		ok   bool
	}{
		{"", ViterbiOff, true},
		{"off", ViterbiOff, true},
		{"false", ViterbiOff, true},
		{"0", ViterbiOff, true},
		{"on", ViterbiOn, true},
		{"ON", ViterbiOn, true},
		{"true", ViterbiOn, true},
		{"1", ViterbiOn, true},
		{" on ", ViterbiOn, true},
		{"nonsense", ViterbiOff, false},
	}
	for _, tc := range cases {
		got, ok := ParseViterbiMode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseViterbiMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
