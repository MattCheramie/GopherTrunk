package mpt1327

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func gtcCodeword(prefix uint8, ident uint16, channel uint16) Codeword {
	return Codeword{
		Type:     TypeAddress,
		Prefix:   prefix,
		Ident:    ident,
		Function: (uint32(KindGoToChan) << 13) | uint32(channel&0x1FFF),
	}
}

func alohaCodeword(prefix uint8) Codeword {
	return Codeword{
		Type:     TypeAddress,
		Prefix:   prefix,
		Function: uint32(KindAloha) << 13,
	}
}

// TestProcessLocksOnFirstAlohaAndDecodesGTC: build a back-to-back
// stream of (Aloha, GoToChannel) codewords. The state machine must
// lock on the Aloha and publish a Grant for the GoToChannel.
func TestProcessLocksOnFirstAlohaAndDecodesGTC(t *testing.T) {
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

	aloha := alohaCodeword(0x5)
	gtc := gtcCodeword(0x5, 0x123, 7)

	stream := append([]byte{}, CodewordBits(aloha)...)
	stream = append(stream, CodewordBits(gtc)...)

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
				t.Errorf("Process did not publish a KindCCLocked")
			}
			if !sawGrant {
				t.Errorf("Process did not publish a KindGrant for the GTC codeword")
			}
			return
		}
	}
}

// TestProcessHandlesCodewordSpanningCalls: an Aloha codeword split
// across two Process calls still locks the channel.
func TestProcessHandlesCodewordSpanningCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	bits := CodewordBits(alohaCodeword(0x3))
	cc.Process(bits[:20], 0)
	cc.Process(bits[20:], 20)

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("Process did not publish a KindCCLocked across the chunk boundary")
			}
			return
		}
	}
}

// TestIsRecognisedAddressCodeword: Data codewords and unknown
// kinds are rejected; recognised Address codewords are accepted.
func TestIsRecognisedAddressCodeword(t *testing.T) {
	if isRecognisedAddressCodeword(Codeword{Type: TypeData}) {
		t.Errorf("Data codeword should not be recognised")
	}
	if isRecognisedAddressCodeword(Codeword{Type: TypeAddress, Function: 0}) {
		t.Errorf("Address codeword with KindUnknown should not be recognised")
	}
	if !isRecognisedAddressCodeword(alohaCodeword(0)) {
		t.Errorf("Aloha codeword should be recognised")
	}
	if !isRecognisedAddressCodeword(gtcCodeword(0, 0, 0)) {
		t.Errorf("GoToChannel codeword should be recognised")
	}
}
