package receiver

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// readCS16 loads an interleaved-int16 IQ file (unit-scaled to ±1).
func readCS16(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}

// TestReceiverRealAirKeyupDecodes is a real-air decode smoke test for the
// production conventional-DMR CC receiver config (channel filter on, the
// Tier II pipeline's clock gain). The fixture is a 1.9 s DDC'd slice of the
// operator's 17 Sep 442.3875 MHz IPSC capture spanning an idle-beacon gap and
// a repeater keyup (tg 11, src 199): the receiver must lock through the gap,
// re-acquire on the keyup and decode the repeated Voice LC Header train that
// grants the call. It guards the combined onset path (channel filter warm-up
// hold, discriminator-transient drop, two-window timing seed) against a gross
// regression on genuine air — the off-channel-neighbour rejection that the
// filter exists for is a multi-gap live-tap effect this short slice can't
// exhibit, and is pinned separately by TestReceiverIgnoresStrongOffChannelNeighbour.
func TestReceiverRealAirKeyupDecodes(t *testing.T) {
	iq := readCS16(t, "testdata/dmr-ipsc-442.3875-keyup-17sep-48k.cs16")
	var got []uint8
	r := New(Options{
		SampleRateHz:    48_000,
		DeviationHz:     1944.0,
		ClockGain:       0.015,
		ChannelFilterHz: DefaultChannelFilterHz,
		DibitSink:       func(d []uint8, _ int) { got = append(got, d...) },
	})
	const chunk = 4096 // RTL-sized
	for i := 0; i < len(iq); i += chunk {
		e := i + chunk
		if e > len(iq) {
			e = len(iq)
		}
		r.Process(iq[i:e])
	}

	det := dmr.NewSyncDetector([]dmr.SyncPattern{dmr.BSData}, 2)
	matches, _ := det.Process(nil, got, 0)
	if len(matches) < 25 {
		t.Fatalf("recovered %d BS-Data syncs from the real-air keyup, want >= 25", len(matches))
	}
	// The keyup repeats its Voice LC Header; a granting receiver decodes
	// several BPTC-clean copies (this fixture: 4). Fewer than 2 means the
	// header train was lost and the call would only late-enter.
	const lb = dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1
	headers := 0
	for _, m := range matches {
		st := m.Index - lb
		if st < 0 || st+dmr.BurstDibits > len(got) {
			continue
		}
		var b dmr.Burst
		copy(b.Dibits[:], got[st:st+dmr.BurstDibits])
		slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
		if err != nil || slot.DataType != dmr.DTVoiceLCHeader {
			continue
		}
		if _, errs := framing.DecodeBPTC196_96(b.PayloadBits()); errs == 0 {
			headers++
		}
	}
	if headers < 2 {
		t.Fatalf("decoded %d BPTC-clean Voice LC Headers from the keyup, want >= 2 — the header train that grants the call was lost", headers)
	}
}
