package phase2

import (
	"encoding/hex"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

func mustMsg(t *testing.T, h string, bits int) []byte {
	t.Helper()
	b, err := hex.DecodeString(h)
	if err != nil {
		t.Fatal(err)
	}
	return framing.UnpackBitsMSB(b, bits)
}

// Real MAC PDUs from a P25 Phase 2 traffic channel, with the channel state
// and structure sequence SDRtrunk independently reports for each. The full
// corpus of 189 walks in TestMACStructureOracle (integration); these three
// keep the walk covered by an ordinary go test.
func TestParseACCHMessageRealAir(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hex   string
		bits  int
		typ   MACPDUType
		off   uint8
		ops   []Opcode
		trunc bool
	}{
		{
			// ACTIVE channel: an in-call source RID followed by a neighbour
			// broadcast, both packed into one 144-bit FACCH-S payload.
			name: "active: group voice channel user + adjacent status",
			hex:  "9C0104561E0D06DD7C0031FC012A1480700022C0",
			bits: 156,
			typ:  MACActive,
			off:  7,
			ops:  []Opcode{0x01, 0x7C},
		},
		{
			// IDLE with nothing but padding: NULL INFORMATION fills the PDU
			// and must not be reported as a structure.
			name: "idle: null information only",
			hex:  "7C0000000000000000000000000000000000CA70",
			bits: 156,
			typ:  MACIdle,
			off:  7,
			ops:  nil,
		},
		{
			// PTT is a single structure spanning the whole PDU, and its
			// opcode is the header byte itself.
			name: "push-to-talk",
			hex:  "240000000000000000008000000D06DD561EE940",
			bits: 156,
			typ:  MACPushToTalk,
			off:  1,
			ops:  []Opcode{0x24},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseACCHMessage(mustMsg(t, tc.hex, tc.bits))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got.Type != tc.typ {
				t.Errorf("type = %v, want %v", got.Type, tc.typ)
			}
			if got.VoiceOffset != tc.off {
				t.Errorf("voice offset = %d, want %d", got.VoiceOffset, tc.off)
			}
			if len(got.Structures) != len(tc.ops) {
				t.Fatalf("%d structures, want %d", len(got.Structures), len(tc.ops))
			}
			for i, op := range tc.ops {
				if got.Structures[i].Opcode != op {
					t.Errorf("structure %d opcode = 0x%02X, want 0x%02X", i,
						byte(got.Structures[i].Opcode), byte(op))
				}
			}
			if got.Truncated != tc.trunc {
				t.Errorf("truncated = %v, want %v", got.Truncated, tc.trunc)
			}
		})
	}
}

// TestParseACCHMessageStopsAtUnknownLength pins the safety property: an opcode
// with no known length ends the walk. The structure itself is still reported —
// its opcode is readable and its bytes are vouched for by the RS and the CRC,
// and MAC fields are laid out from the structure's start — but nothing after
// it is, because continuing would resynchronise onto the wrong byte and
// manufacture a structure that was never transmitted.
func TestParseACCHMessageStopsAtUnknownLength(t *testing.T) {
	// ACTIVE, a known 7-byte structure, then opcode 0x51 (no known length),
	// then a byte that would look like another known opcode if the walk ran on.
	msg := mustMsg(t, "9C0104561E0D06DD51FF7C0031FC012A14800000", 156)
	got, err := ParseACCHMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Structures) != 2 {
		t.Fatalf("%d structures, want 2 (the known one, then the unknown-length one)", len(got.Structures))
	}
	if got.Structures[0].Opcode != 0x01 || got.Structures[1].Opcode != 0x51 {
		t.Errorf("opcodes = 0x%02X, 0x%02X; want 0x01, 0x51",
			byte(got.Structures[0].Opcode), byte(got.Structures[1].Opcode))
	}
	if !got.Truncated {
		t.Error("Truncated = false; an unknown length must be reported, not hidden")
	}
}

func TestParseACCHMessageManufacturerStructure(t *testing.T) {
	// ACTIVE, then a Motorola structure: opcode 0x80, MFID 0x90.
	got, err := ParseACCHMessage(mustMsg(t, "9C809000CBAB262590859009B26A6F38E1C0DD90", 156))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Structures) == 0 {
		t.Fatal("no structures")
	}
	s := got.Structures[0]
	if s.Opcode != 0x80 {
		t.Errorf("opcode = 0x%02X, want 0x80", byte(s.Opcode))
	}
	if !s.Opcode.IsManufacturerSpecific() {
		t.Error("opcode 0x80 should be manufacturer-specific")
	}
	if s.MFID != 0x90 {
		t.Errorf("MFID = 0x%02X, want 0x90 (Motorola)", s.MFID)
	}
}

func TestParseACCHMessageRejectsShort(t *testing.T) {
	if _, err := ParseACCHMessage(make([]byte, 12)); err == nil {
		t.Error("a message with no information bits was accepted")
	}
}

func TestMACPDUTypeStrings(t *testing.T) {
	for _, tc := range []struct {
		t     MACPDUType
		s     string
		voice bool
	}{
		{MACSignal, "SIGNAL", false},
		{MACPushToTalk, "PTT", true},
		{MACEndPushToTalk, "END_PTT", false},
		{MACIdle, "IDLE", false},
		{MACActive, "ACTIVE", true},
		{MACHangtime, "HANGTIME", false},
		{MACPDUType(7), "RESERVED", false},
	} {
		if got := tc.t.String(); got != tc.s {
			t.Errorf("%d.String() = %q, want %q", tc.t, got, tc.s)
		}
		if got := tc.t.CarriesVoice(); got != tc.voice {
			t.Errorf("%s.CarriesVoice() = %v, want %v", tc.s, got, tc.voice)
		}
	}
}
