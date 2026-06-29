package phase1

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestPDUSinkEmitsDecodedBlock confirms a registered PDUSink receives one
// SignalingBlock per decoded TSBK, carrying the opcode name, MFID, NAC, raw
// payload, CRC-OK flag and a plausible dibit offset.
func TestPDUSinkEmitsDecodedBlock(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()

	var blocks []SignalingBlock
	groupPayload := [8]byte{0x00, (1 << 4) | 0x00, 0x10, 0x00, 0xCC, 0x00, 0x04, 0xD2}
	tsbk := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: groupPayload}

	cc := New(Options{
		Bus:         bus,
		FrequencyHz: 450_125_000,
		PDUSink:     func(b SignalingBlock) { blocks = append(blocks, b) },
	})
	stream := buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, tsbk)
	cc.Process(stream, 0)

	if len(blocks) == 0 {
		t.Fatal("PDUSink received no signaling blocks")
	}
	b := blocks[0]
	if b.Opcode != uint8(OpGroupVoiceChannelGrant) {
		t.Errorf("opcode = %d, want %d", b.Opcode, OpGroupVoiceChannelGrant)
	}
	if b.OpcodeName != OpGroupVoiceChannelGrant.String() {
		t.Errorf("opcode name = %q, want %q", b.OpcodeName, OpGroupVoiceChannelGrant.String())
	}
	if b.NAC != 0x293 {
		t.Errorf("NAC = %#x, want 0x293", b.NAC)
	}
	if !b.CRCOK {
		t.Error("CRCOK = false, want true for a clean block")
	}
	if !b.LastBlock {
		t.Error("LastBlock = false, want true")
	}
	if len(b.RawPayload) != 8 || b.RawPayload[4] != 0xCC {
		t.Errorf("RawPayload = %v, want the 8 opcode octets with byte4=0xCC", b.RawPayload)
	}
	if b.DibitLen != 98 || b.DibitStart < 0 {
		t.Errorf("DibitStart=%d DibitLen=%d, want start>=0 len=98", b.DibitStart, b.DibitLen)
	}
}

// TestPDUSinkEmitsCorruptedBlock confirms a CRC-failed block is still surfaced
// (with CRCOK=false) — the protocol-analyzer view includes the bad frames.
func TestPDUSinkEmitsCorruptedBlock(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()

	var blocks []SignalingBlock
	tsbk := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant}
	cc := New(Options{
		Bus:         bus,
		FrequencyHz: 450_125_000,
		PDUSink:     func(b SignalingBlock) { blocks = append(blocks, b) },
	})
	// Corrupt the TSBK channel region of an otherwise-valid frame so the CRC
	// fails while the NID still aligns.
	stream := buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, tsbk)
	for i := len(stream) - 40; i < len(stream)-20 && i >= 0; i++ {
		stream[i] ^= 0x3 // flip dibits inside the TSBK payload
	}
	cc.Process(stream, 0)

	if len(blocks) == 0 {
		t.Fatal("no signaling block emitted for the corrupted frame")
	}
	if blocks[0].CRCOK {
		t.Error("CRCOK = true for a corrupted block, want false")
	}
}
