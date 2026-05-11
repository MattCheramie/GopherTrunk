package edacs

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ccwToWireBCH packs a CCW (Aux must be 0, LCN must fit in 4 bits)
// through the BCH(40,28,2) primitive and returns 40 wire bits ready
// for Process(BCHOn) to consume.
//
// Layout in the 40-bit codeword (matching framing.BCHEncodeEDACS):
//
//	bits 0..11  = 12-bit BCH parity (computed)
//	bits 12..15 = high 4 bits of LCN
//	bits 16..31 = Address
//	bits 32..35 = Status
//	bits 36..39 = Command
//
// The Aux field (low 11 bits of the legacy CCW layout) and bit 0
// of LCN are overwritten by BCH parity and must be zero in the
// input CCW for the round-trip to match.
func ccwToWireBCH(c CCW) []byte {
	// Build the 28 information bits from the CCW fields.
	info := uint32(c.Command&0xF)<<24 |
		uint32(c.Status&0xF)<<20 |
		uint32(c.Address&0xFFFF)<<4 |
		uint32((c.LCN>>1)&0xF)
	cw := framing.BCHEncodeEDACS(info)
	wire := make([]byte, 40)
	for i := 0; i < 40; i++ {
		if cw&(uint64(1)<<uint(39-i)) != 0 {
			wire[i] = 1
		}
	}
	return wire
}

// TestProcessBCHOnDecodesEncodedCCW: build a stream of 30 padding
// bits + 24 sync bits + 40 BCH-encoded codeword bits and confirm
// Process(BCHOn) recovers the original Command / Address / LCN.
func TestProcessBCHOnDecodesEncodedCCW(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 866_000_000,
	})
	cc.SetBCHMode(BCHOn)

	ccw := CCW{
		Command: CmdGroupVoiceGrant,
		Status:  0,
		Address: 0xCAFE,
		// LCN stored in the 4 high bits of the legacy 5-bit field:
		// LCN = 10 = 0b01010, of which the high 4 bits = 0b0101 = 5.
		// Under BCHOn the recovered LCN is (high 4 bits << 1) = 10.
		LCN: 10,
		Aux: 0, // Aux is BCH parity under BCHOn — must be zero in input.
	}
	wire := ccwToWireBCH(ccw)

	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, wire...)

	cc.Process(stream, 0)

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g, _ := ev.Payload.(trunking.Grant)
				if g.GroupID != uint32(ccw.Address) {
					t.Errorf("Grant.GroupID = %#x, want %#x", g.GroupID, ccw.Address)
				}
				if g.ChannelNum != uint16(ccw.LCN&0x1E) {
					// LCN's low bit is BCH parity under BCHOn; the
					// recovered LCN has bit 0 = 0.
					t.Errorf("Grant.ChannelNum = %d, want %d", g.ChannelNum, ccw.LCN&0x1E)
				}
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("BCHOn Process did not publish a Grant")
			}
			return
		}
	}
}

// TestProcessBCHOnCorrectsSingleBitError: flip one bit inside the
// 40-bit BCH-encoded CCW; the decoder must still recover the
// original GroupVoiceGrant and publish the Grant.
func TestProcessBCHOnCorrectsSingleBitError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 866_000_000,
	})
	cc.SetBCHMode(BCHOn)

	ccw := CCW{
		Command: CmdGroupVoiceGrant,
		Address: 0xBEEF,
		LCN:     8,
	}
	wire := ccwToWireBCH(ccw)
	// Flip one bit deep in the info portion (codeword position 20
	// = wire index 19).
	wire[19] ^= 1

	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, wire...)

	cc.Process(stream, 0)

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g, _ := ev.Payload.(trunking.Grant)
				if g.GroupID != uint32(ccw.Address) {
					t.Errorf("Grant.GroupID = %#x, want %#x", g.GroupID, ccw.Address)
				}
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("BCHOn Process did not publish a Grant after single-bit correction")
			}
			return
		}
	}
}

// TestProcessBCHOnCorrectsDoubleBitError: flip two bits in the
// BCH-encoded codeword; BCH(40,28,2) corrects up to 2 errors so
// the Grant must still land.
func TestProcessBCHOnCorrectsDoubleBitError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 866_000_000,
	})
	cc.SetBCHMode(BCHOn)

	ccw := CCW{
		Command: CmdGroupVoiceGrant,
		Address: 0x1234,
		LCN:     4,
	}
	wire := ccwToWireBCH(ccw)
	wire[5] ^= 1  // first error
	wire[25] ^= 1 // second error

	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, wire...)

	cc.Process(stream, 0)

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g, _ := ev.Payload.(trunking.Grant)
				if g.GroupID != uint32(ccw.Address) {
					t.Errorf("Grant.GroupID = %#x, want %#x", g.GroupID, ccw.Address)
				}
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("BCHOn Process did not publish a Grant after double-bit correction")
			}
			return
		}
	}
}

// TestProcessBCHOnDropsUncorrectableCCW: corrupt the codeword with
// three unfavourable bit flips; the BCH(40,28,2) decoder must
// reject (errs == -1) and Process must not publish events.
func TestProcessBCHOnDropsUncorrectableCCW(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:        bus,
		Log:        slog.Default(),
		SystemName: "Sys",
	})
	cc.SetBCHMode(BCHOn)

	ccw := CCW{Command: CmdGroupVoiceGrant, Address: 0xABCD, LCN: 6}
	wire := ccwToWireBCH(ccw)
	// Flip three info bits — beyond BCH(40,28,2)'s t=2 correction
	// capability.
	wire[1] ^= 1
	wire[15] ^= 1
	wire[30] ^= 1

	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, wire...)

	cc.Process(stream, 0)

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked || ev.Kind == events.KindGrant {
				t.Errorf("BCHOn accepted an uncorrectable CCW: %v", ev.Kind)
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
