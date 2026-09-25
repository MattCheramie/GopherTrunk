package phase2

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Real-air FACCH-S bursts from a P25 Phase 2 traffic channel
// (WACN 0xBEE00 / SysID 0x1FC / NAC 0x1F0), captured as baseband I/Q and
// replayed through this package's own receiver and superframe decoder.
//
// These are the load-bearing tests for the whole ACCH chain: the expected
// messages are not this decoder's output pinned against itself, they are what
// SDRtrunk independently decodes from the same capture, byte for byte. Nothing
// short of real air distinguishes a correct descramble and framing from a
// wrong one — every self-consistent property holds either way — so a
// synthesized round-trip cannot replace them.
//
// Each raw burst is 180 dibits written as digits 0-3, in transmission order,
// starting at the first dibit of the ISCH region.
var acchRealAirVectors = []struct {
	name string
	slot int
	raw  string
	want string // SDRtrunk's decode of the same burst
}{
	{
		name: "NULL INFORMATION (idle FACCH-S)",
		slot: 3,
		raw: "111311311113331333332131031010312223220302233213301032330102331121213102023321221130003233" +
			"333111113131331133112110301201210220202001312332121323132300211012222113220212330031033002",
		want: "7C0000000000000000000000000000000000CA70",
	},
	{
		name: "GROUP VOICE CHANNEL USER + ADJACENT STATUS BROADCAST, site 42",
		slot: 4,
		raw: "010113022121302220102212010223231102303300213210211012232101020201120112201023223222302113" +
			"333111113131331133111313332302301101312001212331033223312311000310213201011003310232313302",
		want: "9C0104561E0D06DD7C0031FC012A1480700022C0",
	},
	{
		name: "GROUP VOICE CHANNEL USER + ADJACENT STATUS BROADCAST, site 41",
		slot: 6,
		raw: "111311311113331333332210120130220222333030230313011102320101320123223231113320110232103003" +
			"333111113131331133112103020131131203310131213131003323012022322320002233323213322031100032",
		want: "9C0104561E0D06DD7C0031FC012900F5700063B0",
	},
}

func parseBurst(t *testing.T, raw string) []uint8 {
	t.Helper()
	if len(raw) != BurstDibits {
		t.Fatalf("burst is %d dibits, want %d", len(raw), BurstDibits)
	}
	out := make([]uint8, BurstDibits)
	for i, c := range raw {
		if c < '0' || c > '3' {
			t.Fatalf("dibit %d is %q", i, c)
		}
		out[i] = uint8(c - '0')
	}
	return out
}

func msgHex(bits []byte) string {
	return strings.ToUpper(hex.EncodeToString(framing.PackBitsMSB(bits)))
}

func TestDecodeACCHBurstRealAir(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	for _, v := range acchRealAirVectors {
		t.Run(v.name, func(t *testing.T) {
			burst := parseBurst(t, v.raw)
			if bt := BurstTypeOf(burst); bt != BurstFACCHScrambled {
				t.Fatalf("burst type = %d, want %d (scrambled FACCH)", bt, BurstFACCHScrambled)
			}
			res, ok := DecodeACCHBurst(burst, v.slot, seq)
			if !ok {
				t.Fatalf("decode failed")
			}
			if !res.Burst.IsFast() {
				t.Errorf("IsFast = false")
			}
			if !res.RSValid {
				t.Errorf("RS(63,35,29) did not close on a clean real-air burst")
			}
			if len(res.Message) != 156 {
				t.Errorf("message is %d bits, want 156", len(res.Message))
			}
			if got := msgHex(res.Message); got != v.want {
				t.Errorf("message = %s\n            want %s (SDRtrunk)", got, v.want)
			}
		})
	}
}

// TestDecodeACCHBurstWrongSlotRejected checks that the CRC-12 actually gates
// the descramble: the same burst at any other slot phase is noise, and must
// not come back as a PDU. This is what keeps a mis-phased channel from
// injecting fabricated grants into the archive.
func TestDecodeACCHBurstWrongSlotRejected(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	for _, v := range acchRealAirVectors {
		burst := parseBurst(t, v.raw)
		for slot := 0; slot < SubframesPerSuperframe; slot++ {
			if slot == v.slot {
				continue
			}
			if _, ok := DecodeACCHBurst(burst, slot, seq); ok {
				t.Errorf("%s: slot %d also decoded; only %d should", v.name, slot, v.slot)
			}
		}
	}
}

// Each 6-bit RS symbol spans three payload dibits, and the first coded window
// starts at payload dibit 1, so flipping dibits 1, 4, 7, … damages that many
// distinct symbols — the unit the outer code actually counts.
func damageSymbols(burst []uint8, n int) {
	for i := 0; i < n; i++ {
		burst[ISCHRegionDibits+1+3*i] ^= 1
	}
}

// TestDecodeACCHBurstRepairsSymbolErrors is what the outer RS(63,35,29) is
// for: a FACCH-S burst arrives with nine of its parity symbols punctured, and
// the remaining budget repairs up to nine damaged symbols. The message that
// comes out must still be the one SDRtrunk read off the air.
func TestDecodeACCHBurstRepairsSymbolErrors(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	v := acchRealAirVectors[1]
	for _, n := range []int{1, 5, 9} {
		burst := parseBurst(t, v.raw)
		damageSymbols(burst, n)
		res, ok := DecodeACCHBurst(burst, v.slot, seq)
		if !ok {
			t.Fatalf("%d damaged symbols: decode failed, should have been repaired", n)
		}
		if !res.RSValid {
			t.Errorf("%d damaged symbols: RSValid = false", n)
		}
		if res.RSErrors != n {
			t.Errorf("%d damaged symbols: reported %d corrected", n, res.RSErrors)
		}
		if got := msgHex(res.Message); got != v.want {
			t.Errorf("%d damaged symbols: message = %s, want %s", n, got, v.want)
		}
	}
}

// TestDecodeACCHBurstRejectsBeyondRSBudget: past the radius the burst must be
// dropped. Neither code may vouch for it — the RS fails outright, and the
// CRC-12 on the uncorrected symbols has to fail too, since a burst this
// damaged carrying a plausible MAC PDU is a fabricated grant.
func TestDecodeACCHBurstRejectsBeyondRSBudget(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	v := acchRealAirVectors[1]
	burst := parseBurst(t, v.raw)
	damageSymbols(burst, 14)
	if res, ok := DecodeACCHBurst(burst, v.slot, seq); ok {
		t.Errorf("14 damaged symbols still decoded (RSValid=%v) to %s",
			res.RSValid, msgHex(res.Message))
	}
}

// TestDecodeACCHBurstRepairsParityDamage pins the other half of the erasure
// geometry: the trailing parity symbols are not dead weight — damage to a
// transmitted parity symbol is a symbol error like any other, counted and
// repaired, and the message is unaffected either way.
func TestDecodeACCHBurstRepairsParityDamage(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	v := acchRealAirVectors[1]
	burst := parseBurst(t, v.raw)
	burst[ISCHRegionDibits+158] ^= 1 // last coded dibit: an RS parity symbol
	res, ok := DecodeACCHBurst(burst, v.slot, seq)
	if !ok {
		t.Fatalf("damaging a parity symbol broke the decode")
	}
	if !res.RSValid || res.RSErrors != 1 {
		t.Errorf("RSValid = %v, RSErrors = %d; want true, 1", res.RSValid, res.RSErrors)
	}
	if got := msgHex(res.Message); got != v.want {
		t.Errorf("message changed to %s, want %s", got, v.want)
	}
}

func TestBurstTypeOfCorrectsSingleBitError(t *testing.T) {
	burst := make([]uint8, BurstDibits)
	// FACCH-S codeword: value 9 (1001) with parity 1010 → 0x9A.
	const cw = 0x9A
	for i, pos := range duidDibitPositions {
		burst[pos] = uint8(cw >> (6 - 2*i) & 3)
	}
	if got := BurstTypeOf(burst); got != BurstFACCHScrambled {
		t.Fatalf("clean codeword decoded to %d, want %d", got, BurstFACCHScrambled)
	}
	for bit := 0; bit < 8; bit++ {
		corrupt := cw ^ (1 << bit)
		b := make([]uint8, BurstDibits)
		for i, pos := range duidDibitPositions {
			b[pos] = uint8(corrupt >> (6 - 2*i) & 3)
		}
		if got := BurstTypeOf(b); got != BurstFACCHScrambled {
			t.Errorf("bit %d flipped: decoded %d, want %d", bit, got, BurstFACCHScrambled)
		}
	}
}

func TestBurstTypeOfRejectsShortBurst(t *testing.T) {
	if got := BurstTypeOf(make([]uint8, BurstDibits-1)); got != BurstInvalid {
		t.Errorf("short burst decoded to %d, want BurstInvalid", got)
	}
}

func TestCRC12P25P2MatchesRealAirTrailer(t *testing.T) {
	// The trailer SDRtrunk carries on the idle FACCH-S PDU above.
	raw, err := hex.DecodeString("7C0000000000000000000000000000000000CA70")
	if err != nil {
		t.Fatal(err)
	}
	bits := framing.UnpackBitsMSB(raw, 156)
	if !framing.CRC12P25P2OK(bits) {
		t.Fatalf("real-air PDU failed its own CRC-12")
	}
	bits[7] ^= 1
	if framing.CRC12P25P2OK(bits) {
		t.Errorf("corrupted PDU still passed CRC-12")
	}
}

// TestDUIDCodewordsPinned pins all sixteen DUID codewords. The code is
// generated from four parity rows rather than copied, so a transcription slip
// cannot be caught by inspection — and a bit-reversed row order still decodes
// the four self-reversing types (0, 6, 9, 15) correctly, which covers voice
// and both FACCH forms and hides the error behind a working decoder.
func TestDUIDCodewordsPinned(t *testing.T) {
	want := [16]byte{
		0x00, 0x17, 0x2E, 0x39, 0x4B, 0x5C, 0x65, 0x72,
		0x8D, 0x9A, 0xA3, 0xB4, 0xC6, 0xD1, 0xE8, 0xFF,
	}
	for v, cw := range want {
		if got := duidDecodeTable[cw]; got != BurstType(v) {
			t.Errorf("codeword 0x%02X decoded to %d, want %d", cw, got, v)
		}
	}
	// The code has distance 3, so exactly 16*9 = 144 of the 256 received
	// bytes are within one bit of a codeword and the rest must be rejected.
	valid := 0
	for _, bt := range duidDecodeTable {
		if bt != BurstInvalid {
			valid++
		}
	}
	if valid != 144 {
		t.Errorf("%d decodable codewords, want 144 (16 codewords x 9)", valid)
	}
}

// TestBurstTypeSACCHRoundTrip guards the specific pair the reversed row order
// swapped: scrambled SACCH must not be read as the unscrambled form, or its
// payload is never descrambled and the burst silently yields nothing.
func TestBurstTypeSACCHRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		cw        byte
		want      BurstType
		scrambled bool
	}{
		{0x39, BurstSACCHScrambled, true},
		{0xC6, BurstSACCHUnscrambled, false},
		{0x9A, BurstFACCHScrambled, true},
		{0xFF, BurstFACCHUnscrambled, false},
	} {
		burst := make([]uint8, BurstDibits)
		for i, pos := range duidDibitPositions {
			burst[pos] = tc.cw >> (6 - 2*i) & 3
		}
		got := BurstTypeOf(burst)
		if got != tc.want {
			t.Errorf("codeword 0x%02X → burst type %d, want %d", tc.cw, got, tc.want)
		}
		if got.IsScrambled() != tc.scrambled {
			t.Errorf("codeword 0x%02X: IsScrambled = %v, want %v", tc.cw, got.IsScrambled(), tc.scrambled)
		}
	}
}

// TestEncodeDecodeACCHBurstRoundTrip exercises the generator against the
// decoder across both burst kinds and every slot phase. It cannot replace the
// real-air vectors — an encoder built from the same assumptions as the decoder
// agrees with it whether or not those assumptions match the air — but it does
// cover the payload widths and structure layouts real captures never happened
// to contain.
func TestEncodeDecodeACCHBurstRoundTrip(t *testing.T) {
	seq := ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	structures := []MACPDU{
		{Opcode: 0x01, Payload: []byte{0x04, 0x56, 0x1E, 0x0D, 0x06, 0xDD}},
		{Opcode: 0x7C, Payload: []byte{0x00, 0x31, 0xFC, 0x01, 0x2A, 0x14, 0x80, 0x70}},
	}
	for _, bt := range []BurstType{BurstFACCHScrambled, BurstSACCHScrambled,
		BurstFACCHUnscrambled, BurstSACCHUnscrambled} {
		for slot := 0; slot < SubframesPerSuperframe; slot++ {
			burst := EncodeACCHBurst(bt, MACActive, 7, structures, slot, seq)
			if got := BurstTypeOf(burst); got != bt {
				t.Fatalf("burst type %d round-tripped as %d", bt, got)
			}
			res, ok := DecodeACCHBurst(burst, slot, seq)
			if !ok {
				t.Fatalf("type %d slot %d: decode failed", bt, slot)
			}
			if !res.RSValid || res.RSErrors != 0 {
				t.Errorf("type %d slot %d: RSValid=%v errors=%d, want true/0",
					bt, slot, res.RSValid, res.RSErrors)
			}
			msg, err := ParseACCHMessage(res.Message)
			if err != nil {
				t.Fatalf("type %d slot %d: %v", bt, slot, err)
			}
			if msg.Type != MACActive || msg.VoiceOffset != 7 {
				t.Errorf("type %d slot %d: header %v/%d", bt, slot, msg.Type, msg.VoiceOffset)
			}
			if len(msg.Structures) != 2 ||
				msg.Structures[0].Opcode != 0x01 || msg.Structures[1].Opcode != 0x7C {
				t.Errorf("type %d slot %d: structures = %+v", bt, slot, msg.Structures)
			}
		}
	}
}
