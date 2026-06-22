package phase2

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestDecodedFramesCountsMACPDUs verifies DecodedFrames bumps once per MAC
// PDU handed to Ingest — every such PDU has already cleared trellis + RS +
// CRC in DecodeSuperframeMACPDUs, so each is a decode success regardless of
// whether its opcode is recognised. This is the decode-activity signal the
// wideband engine gates power logging on.
func TestDecodedFramesCountsMACPDUs(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	if got := cc.DecodedFrames(); got != 0 {
		t.Fatalf("fresh DecodedFrames() = %d, want 0", got)
	}

	payload := []byte{
		0x00,       // service options
		0x10, 0x07, // channel ID 1 + channel 7
		0xCA, 0xFE, // group address
		0x12, 0x34, 0x56, // source ID
	}
	cc.Ingest(MACPDU{Opcode: OpGroupVoiceChannelGrant, Payload: payload})
	cc.Ingest(MACPDU{Opcode: OpGroupVoiceChannelGrant, Payload: payload})
	if got := cc.DecodedFrames(); got != 2 {
		t.Fatalf("after 2 PDUs DecodedFrames() = %d, want 2", got)
	}

	// Even a strict-dropped unknown opcode counts as decode activity: it
	// cleared FEC, the opcode is simply unrecognised.
	cc.SetStrictValidation(true)
	cc.Ingest(MACPDU{Opcode: Opcode(0x7E), Payload: make([]byte, 8)})
	if got := cc.DecodedFrames(); got != 3 {
		t.Fatalf("after a strict-dropped PDU DecodedFrames() = %d, want 3", got)
	}
}
