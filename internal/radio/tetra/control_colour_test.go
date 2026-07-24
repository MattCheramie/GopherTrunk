package tetra

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func newColourTestCC(t *testing.T) *ControlChannel {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	return New(Options{
		Bus:        bus,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName: "Sys",
	})
}

// TestLearnColourCodeCorrectsMisdecodedFirstBSCH is the regression for the
// wrong-colour lock that silently broke descramble for a whole session. A single
// mis-decoded first sync burst (bit errors that slipped through the BSCH FEC)
// must NOT lock a wrong colour: the true colour, carried by every later BSCH,
// has to overtake the fluke. These two values are the real 24-Jul capture case —
// the cell's colour is 44 (…6c) but one grab mis-learned 24 (…58).
func TestLearnColourCodeCorrectsMisdecodedFirstBSCH(t *testing.T) {
	cc := newColourTestCC(t)
	const wrong = uint32(0x0fa00358) // MCC 250 / MNC 13 / colour 24 (mis-decoded)
	const right = uint32(0x0fa0036c) // MCC 250 / MNC 13 / colour 44 (the cell)

	// First BSCH is adopted provisionally so a cold receiver can start decoding.
	cc.LearnColourCode(wrong)
	if got := cc.ColourCode(); got != wrong {
		t.Fatalf("provisional colour = %#x, want %#x (first BSCH adopted)", got, wrong)
	}

	// The true colour arrives on the next sync bursts and, once corroborated,
	// must correct the fluke. (On the pre-fix lock-on-first behaviour the colour
	// stayed `wrong` here and every BNCH/SCH/TCH descramble failed all session.)
	cc.LearnColourCode(right)
	cc.LearnColourCode(right)
	if got := cc.ColourCode(); got != right {
		t.Errorf("colour = %#x, want %#x (a corroborated colour must correct a mis-decoded first BSCH)", got, right)
	}
}

// TestLearnColourCodeLocksOnceConfirmed confirms that a normally-acquired colour
// (two agreeing BSCHs) locks and a later single fluke can no longer change it —
// so the confirmation window doesn't leave the colour permanently mutable.
func TestLearnColourCodeLocksOnceConfirmed(t *testing.T) {
	cc := newColourTestCC(t)
	const good = uint32(0x0fa0036c)
	const fluke = uint32(0x0fa00358)

	cc.LearnColourCode(good) // provisional
	cc.LearnColourCode(good) // corroborated → confirmed + locked
	if got := cc.ColourCode(); got != good {
		t.Fatalf("colour = %#x, want %#x after two agreeing BSCHs", got, good)
	}
	cc.LearnColourCode(fluke) // a single later mis-decode must be ignored
	if got := cc.ColourCode(); got != good {
		t.Errorf("colour = %#x, want %#x (a locked colour must ignore a single fluke)", got, good)
	}
}

// TestSetColourCodeIsAuthoritative confirms an operator-configured colour is
// never overridden by learned BSCHs, however many agree.
func TestSetColourCodeIsAuthoritative(t *testing.T) {
	cc := newColourTestCC(t)
	cc.SetColourCode(0x12345)
	cc.LearnColourCode(0x0fa0036c)
	cc.LearnColourCode(0x0fa0036c)
	cc.LearnColourCode(0x0fa0036c)
	if got := cc.ColourCode(); got != 0x12345 {
		t.Errorf("colour = %#x, want %#x (operator-set colour must win over learned BSCHs)", got, 0x12345)
	}
}
