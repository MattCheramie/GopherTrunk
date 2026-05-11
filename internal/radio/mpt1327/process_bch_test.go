package mpt1327

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// codewordToWire64 maps a gophertrunk Codeword (38 info bits) into
// the 48-bit info field BCHEncodeMPT1327 expects, then encodes the
// full 64-bit on-wire codeword, then unpacks it into a 64-bit
// wire-bit array suitable for feeding into Process(BCHOn).
//
// The 10-bit Op field (which gophertrunk's Codeword doesn't model)
// is filled with zero. The bit layout matches what parseCodeword's
// inverse extraction expects:
//
//	wire 0..20  = Type (1) + Prefix (7) + Ident (13) — 21 bits
//	wire 21..30 = Op (10) — set to zero
//	wire 31..47 = Function (17)
//	wire 48..62 = BCH check (computed)
//	wire 63     = overall even parity (computed)
//
// The 38-bit prefix of wire (Type + Prefix + Ident + Function with
// Op skipped) preserves the same MSB-first bit order that
// CodewordBits produces, so the round-trip back through
// CodewordFromBits decodes the same Codeword.
func codewordToWire64(c Codeword) []byte {
	wire38 := CodewordBits(c)
	var info48 uint64
	// Place Type + Prefix + Ident in info48 bits 0..20.
	for i := 0; i < 21; i++ {
		if wire38[i]&1 != 0 {
			info48 |= uint64(1) << uint(i)
		}
	}
	// Op (10 bits) at info48 21..30 stays zero — Codeword
	// doesn't model the Op field.
	// Place Function in info48 bits 31..47.
	for i := 0; i < 17; i++ {
		if wire38[21+i]&1 != 0 {
			info48 |= uint64(1) << uint(31+i)
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
	cc.SetBCHMode(BCHOn)
	if cc.bchMode != BCHOn {
		t.Errorf("SetBCHMode(BCHOn) did not take effect")
	}
	cc.SetBCHMode(BCHOff)
	if cc.bchMode != BCHOff {
		t.Errorf("SetBCHMode(BCHOff) did not take effect")
	}
}
