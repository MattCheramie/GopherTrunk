package phase2

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// interleavedGrantStream builds a 30-dibit lead-in + outbound sync +
// the block-interleaved 72-dibit MAC burst carrying a group voice grant
// for talkgroup tg.
func interleavedGrantStream(tg uint16) []uint8 {
	g := grantPDU(tg, 0x00ABCD, 0x1, 0x005)
	g.Payload = append(g.Payload, make([]byte, 17-len(g.Payload))...)
	pduDibits := framing.BitsToDibits(framing.UnpackBitsMSB(AssembleMACPDU(g), 144))
	interleaved := framing.InterleaveMACBurst(pduDibits)

	stream := make([]uint8, 30)
	stream = append(stream, OutboundSyncDibits()...)
	stream = append(stream, interleaved...)
	return stream
}

func grantTalkgroups(sub *events.Subscription) []uint32 {
	var out []uint32
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				if g, ok := ev.Payload.(trunking.Grant); ok {
					out = append(out, g.GroupID)
				}
			}
		default:
			return out
		}
	}
}

// TestProcessDeinterleavesMACBurst confirms that with InterleaveOn the
// Process adapter undoes the block interleaver and recovers the grant,
// and that InterleaveOff on the same interleaved stream does not.
func TestProcessDeinterleavesMACBurst(t *testing.T) {
	const tg = 0x1234
	stream := interleavedGrantStream(tg)

	t.Run("InterleaveOn recovers the grant", func(t *testing.T) {
		bus := events.NewBus(8)
		defer bus.Close()
		sub := bus.Subscribe()
		defer sub.Close()

		cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
		cc.SetInterleaveMode(InterleaveOn)
		cc.Process(stream, 0)

		found := false
		for _, got := range grantTalkgroups(sub) {
			if got == tg {
				found = true
			}
		}
		if !found {
			t.Errorf("InterleaveOn: no grant for talkgroup %#x", tg)
		}
	})

	t.Run("InterleaveOff misdecodes the interleaved burst", func(t *testing.T) {
		bus := events.NewBus(8)
		defer bus.Close()
		sub := bus.Subscribe()
		defer sub.Close()

		cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
		// InterleaveOff is the default; feeding the interleaved burst
		// without deinterleaving must not reproduce the real grant.
		cc.Process(stream, 0)

		for _, got := range grantTalkgroups(sub) {
			if got == tg {
				t.Errorf("InterleaveOff: unexpectedly recovered talkgroup %#x from an interleaved burst", tg)
			}
		}
	})
}
