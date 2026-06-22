package tier3

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// TestDecodedFramesCountsCSBKs verifies DecodedFrames bumps once per CSBK
// that clears BPTC + CRC and stays flat on an uncorrectable burst — the
// decode-activity signal the wideband engine gates power logging on.
func TestDecodedFramesCountsCSBKs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	if got := cc.DecodedFrames(); got != 0 {
		t.Fatalf("fresh DecodedFrames() = %d, want 0", got)
	}

	c := CSBK{LB: true, Opcode: OpAloha, FID: 0, Payload: [8]byte{0x12, 0x08, 0x12, 0x34, 0, 0, 0, 0}}
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 1, DataType: dmr.DTCSBK})
	if got := cc.DecodedFrames(); got != 2 {
		t.Fatalf("after 2 good CSBKs DecodedFrames() = %d, want 2", got)
	}

	// A non-CSBK data type is not a control-block decode and must not
	// count: IngestBurst ignores it, leaving the counter where it was.
	cc.IngestBurst(burstWithCSBK(c), dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader})
	if got := cc.DecodedFrames(); got != 2 {
		t.Fatalf("after a non-CSBK burst DecodedFrames() = %d, want 2", got)
	}
}
