package mpt1327

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// codewordToWire64 maps a gophertrunk Codeword (all five fields:
// Type, Prefix, Ident, Op, Function) into the 48-bit info field
// BCHEncodeMPT1327 expects, then encodes the full 64-bit on-wire
// codeword, then unpacks it into a 64-bit wire-bit array suitable
// for feeding into Process(BCHOn).
//
// Bit layout matches what parseCodeword's inverse extraction
// expects:
//
//	wire 0..20  = Type (1) + Prefix (7) + Ident (13) — 21 bits
//	wire 21..30 = Op (10)
//	wire 31..47 = Function (17)
//	wire 48..62 = BCH check (computed)
//	wire 63     = overall even parity (computed)
//
// The 48-bit prefix of wire preserves the same MSB-first bit order
// that CodewordBits48 produces, so the round-trip back through
// CodewordFromBits48 decodes the same Codeword.
func codewordToWire64(c Codeword) []byte {
	wire48 := CodewordBits48(c)
	var info48 uint64
	for i := 0; i < 48; i++ {
		if wire48[i]&1 != 0 {
			info48 |= uint64(1) << uint(i)
		}
	}
	cw := framing.BCHEncodeMPT1327(info48)
	wire := make([]byte, 64)
	for i := 0; i < 64; i++ {
		wire[i] = byte((cw >> uint(i)) & 1)
	}
	return wire
}

// TestProcessBCHOnDecodesEncodedCodeword: build a stream of two
// BCH-encoded 64-bit codewords (Aloha → GoToChannel) and confirm
// Process with SetBCHMode(BCHOn) recovers the same trunking
// events as the BCHOff path produces from 38-bit codewords.
func TestProcessBCHOnDecodesEncodedCodeword(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 169_212_500,
	})
	cc.SetBCHMode(BCHOn)

	aloha := alohaCodeword(0x5)
	gtc := gtcCodeword(0x5, 0x123, 7)

	stream := append([]byte{}, codewordToWire64(aloha)...)
	stream = append(stream, codewordToWire64(gtc)...)

	cc.Process(stream, 0)

	var sawLock, sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				ls, _ := ev.Payload.(LockState)
				if ls.Prefix != 0x5 {
					t.Errorf("LockState.Prefix = %d, want 5", ls.Prefix)
				}
				sawLock = true
			case events.KindGrant:
				g, _ := ev.Payload.(trunking.Grant)
				if g.Protocol != "mpt1327" {
					t.Errorf("Grant.Protocol = %q, want mpt1327", g.Protocol)
				}
				if g.ChannelNum != 7 {
					t.Errorf("Grant.ChannelNum = %d, want 7", g.ChannelNum)
				}
				sawGrant = true
			}
		default:
			if !sawLock {
				t.Errorf("BCHOn Process did not publish a KindCCLocked")
			}
			if !sawGrant {
				t.Errorf("BCHOn Process did not publish a KindGrant")
			}
			return
		}
	}
}

// TestProcessBCHOnCorrectsSingleBitError: flip one bit in the
// 64-bit Aloha codeword and confirm BCH-correction still drives
// cc.locked. Picks a position in the info-bit half (0..47) where
// the BCH single-error correction guarantees exact recovery.
func TestProcessBCHOnCorrectsSingleBitError(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 169_212_500,
	})
	cc.SetBCHMode(BCHOn)

	aloha := alohaCodeword(0x5)
	wire := codewordToWire64(aloha)
	// Flip an info bit deep in the codeword (position 35 sits
	// inside Function, well clear of Prefix's syndrome-collision
	// range). Use the alignment-codeword-then-fixed-stride flow
	// by prefixing a clean recognised codeword so alignment locks
	// first.
	clean := codewordToWire64(aloha)
	corrupted := append([]byte{}, wire...)
	corrupted[35] ^= 1

	stream := append([]byte{}, clean...)
	stream = append(stream, corrupted...)

	cc.Process(stream, 0)

	var lockCount int
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				lockCount++
			}
		default:
			if lockCount == 0 {
				t.Errorf("BCHOn Process did not publish a KindCCLocked even after single-bit correction")
			}
			return
		}
	}
}

// TestProcessBCHOnDropsUncorrectableCodeword: flip two info bits
// in unfavourable positions to produce an uncorrectable codeword
// and confirm Process drops it (alignment falls back to search).
func TestProcessBCHOnDropsUncorrectableCodeword(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 0,
	})
	cc.SetBCHMode(BCHOn)

	aloha := alohaCodeword(0x5)
	wire := codewordToWire64(aloha)
	// Flip two info bits with non-colliding syndromes — the
	// decoder can't correct both.
	wire[20] ^= 1
	wire[33] ^= 1

	cc.Process(wire, 0)

	// The state machine should NOT lock from a single corrupted
	// codeword. (Search-and-retry is allowed; we just verify no
	// KindCCLocked landed.)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				t.Errorf("BCHOn Process locked on an uncorrectable codeword: %v", ev)
			}
		default:
			return
		}
	}
}

func TestSetBCHModeDefault(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(1)})
	if cc.bchMode != BCHOff {
		t.Errorf("default bchMode = %v, want BCHOff", cc.bchMode)
	}
	if got := cc.BCHMode(); got != BCHOff {
		t.Errorf("BCHMode() = %v, want BCHOff", got)
	}
	cc.SetBCHMode(BCHOn)
	if cc.bchMode != BCHOn {
		t.Errorf("SetBCHMode(BCHOn) did not take effect")
	}
	if got := cc.BCHMode(); got != BCHOn {
		t.Errorf("BCHMode() = %v, want BCHOn", got)
	}
	cc.SetBCHMode(BCHOff)
	if cc.bchMode != BCHOff {
		t.Errorf("SetBCHMode(BCHOff) did not take effect")
	}
}

// TestParseBCHMode covers the config-string → BCHMode mapping the
// ccdecoder connector uses to translate the `mpt1327_bch_mode`
// YAML field into a SetBCHMode call.
func TestParseBCHMode(t *testing.T) {
	cases := []struct {
		in   string
		want BCHMode
		ok   bool
	}{
		{"", BCHOff, true},
		{"off", BCHOff, true},
		{"false", BCHOff, true},
		{"0", BCHOff, true},
		{"on", BCHOn, true},
		{"ON", BCHOn, true},
		{"true", BCHOn, true},
		{"1", BCHOn, true},
		{" on ", BCHOn, true},
		{"nonsense", BCHOff, false},
	}
	for _, tc := range cases {
		got, ok := ParseBCHMode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseBCHMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestParseCodewordBCHOnSurfacesOp: encode a Codeword whose Op
// field is non-zero through the 64-bit on-wire form, run it back
// through parseCodeword under BCHOn, and confirm Op survives the
// round-trip (along with Type / Prefix / Ident / Function).
func TestParseCodewordBCHOnSurfacesOp(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(1)})
	in := Codeword{
		Type:     TypeAddress,
		Prefix:   0x05,
		Ident:    0x123,
		Op:       0x2AA, // 10-bit non-zero
		Function: 0x1ABCD,
	}
	wire := codewordToWire64(in)
	got, ok := cc.parseCodeword(wire, BCHOn)
	if !ok {
		t.Fatalf("parseCodeword rejected a clean codeword under BCHOn")
	}
	if got != in {
		t.Errorf("BCHOn round-trip lost a field:\n  got  %+v\n  want %+v", got, in)
	}
}

// TestParseCodewordBCHOffPreservesLegacyOp: under BCHOff the
// adapter takes a 38-bit wire window and parses via CodewordFromBits
// (legacy 38-bit path). Op stays at zero because the 38-bit layout
// doesn't include it — confirms back-compat for callers that still
// use the legacy fixture-generation path.
func TestParseCodewordBCHOffPreservesLegacyOp(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(1)})
	in := Codeword{
		Type:     TypeAddress,
		Prefix:   0x05,
		Ident:    0x123,
		Op:       0x000, // legacy 38-bit fixtures don't populate Op
		Function: 0x1ABCD,
	}
	wire := CodewordBits(in) // 38-bit, legacy
	got, ok := cc.parseCodeword(wire, BCHOff)
	if !ok {
		t.Fatalf("parseCodeword rejected a clean 38-bit codeword under BCHOff")
	}
	if got != in {
		t.Errorf("BCHOff round-trip mismatch:\n  got  %+v\n  want %+v", got, in)
	}
}
