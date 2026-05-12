package mpt1327

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// cwscBitsSlice returns the 16-bit CWSC pattern (`1100010011010111`)
// as a byte slice with one bit per element — the format the Process
// adapter expects from the receiver.
func cwscBitsSlice() []byte {
	out := make([]byte, cwscBits)
	for i := 0; i < cwscBits; i++ {
		out[i] = cwscPattern[i]
	}
	return out
}

// TestFindCWSCFindsExactPattern verifies the bit-by-bit matcher
// locates the 16-bit sync sequence anywhere in the buffer.
func TestFindCWSCFindsExactPattern(t *testing.T) {
	for _, lead := range []int{0, 1, 7, 31, 63} {
		buf := make([]byte, lead+cwscBits+8)
		// Fill the leader with 0s and 1s alternating so it doesn't
		// accidentally match.
		for i := 0; i < lead; i++ {
			buf[i] = byte(i & 1)
		}
		copy(buf[lead:], cwscBitsSlice())
		got, ok := findCWSC(buf, 0)
		if !ok {
			t.Errorf("lead=%d: findCWSC returned !ok", lead)
			continue
		}
		if got != lead {
			t.Errorf("lead=%d: findCWSC = %d, want %d", lead, got, lead)
		}
	}
}

// TestFindCWSCSkipsPartialMatches confirms a near-miss in the pattern
// doesn't false-positive.
func TestFindCWSCSkipsPartialMatches(t *testing.T) {
	pattern := cwscBitsSlice()
	// Flip exactly one bit in the middle so the sequence no longer
	// matches.
	buf := make([]byte, len(pattern))
	copy(buf, pattern)
	buf[7] ^= 1
	if _, ok := findCWSC(buf, 0); ok {
		t.Errorf("findCWSC accepted a 1-bit-flipped pattern; expected exact-match rejection")
	}
}

// TestFindCWSCRespectsFromOffset confirms the scan begins at the
// supplied start index, so a CWSC that appears before the cursor
// position isn't re-matched.
func TestFindCWSCRespectsFromOffset(t *testing.T) {
	first := cwscBitsSlice()
	second := cwscBitsSlice()
	gap := make([]byte, 8)
	buf := append(append(first, gap...), second...)
	got, ok := findCWSC(buf, len(first))
	if !ok {
		t.Fatalf("findCWSC didn't find the second copy")
	}
	want := len(first) + len(gap)
	if got != want {
		t.Errorf("findCWSC = %d, want %d", got, want)
	}
}

// TestProcessLocksOnCWSCBeforeRecognisedCodeword exercises the
// two-stage alignment: a CWSC-prefixed Aloha codeword must lock the
// state machine at the byte immediately after the sync pattern, even
// when the preceding bits in the buffer are noise that would
// otherwise drive the fallback search.
func TestProcessLocksOnCWSCBeforeRecognisedCodeword(t *testing.T) {
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

	// Noise prefix that's deliberately not a parseable codeword and
	// doesn't contain CWSC.
	noise := make([]byte, 40)
	for i := range noise {
		noise[i] = byte(i&1) ^ 1
	}

	aloha := alohaCodeword(0x5)
	stream := append([]byte{}, noise...)
	stream = append(stream, cwscBitsSlice()...)
	stream = append(stream, CodewordBits(aloha)...)

	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event = %v, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload = %T, want LockState", ev.Payload)
		}
		if ls.Prefix != 0x5 {
			t.Errorf("LockState.Prefix = %#x, want 0x5", ls.Prefix)
		}
	default:
		t.Fatal("no cc.locked event after CWSC-prefixed Aloha")
	}
}

// TestProcessCWSCPrefixedGTCEmitsGrant confirms a grant publishes
// when an Aloha + GoToChan pair follows the sync.
func TestProcessCWSCPrefixedGTCEmitsGrant(t *testing.T) {
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
	stream := append([]byte{}, cwscBitsSlice()...)
	stream = append(stream, CodewordBits(aloha)...)
	stream = append(stream, CodewordBits(gtc)...)

	cc.Process(stream, 0)

	var sawLock, sawGrant bool
	for i := 0; i < 4; i++ {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				sawLock = true
			case events.KindGrant:
				sawGrant = true
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Errorf("grant payload = %T, want trunking.Grant", ev.Payload)
					continue
				}
				if g.ChannelNum != 7 {
					t.Errorf("grant.ChannelNum = %d, want 7", g.ChannelNum)
				}
			}
		default:
		}
	}
	if !sawLock {
		t.Error("did not see cc.locked")
	}
	if !sawGrant {
		t.Error("did not see grant")
	}
}

// TestProcessFallsBackWhenNoCWSC confirms the legacy "first parseable
// codeword wins" path still works when the stream doesn't include the
// sync sequence — protects synthesized-fixture tests that pre-date
// this PR.
func TestProcessFallsBackWhenNoCWSC(t *testing.T) {
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
	// No CWSC prefix; the existing fallback path should still
	// recognise the Aloha codeword and lock.
	cc.Process(CodewordBits(aloha), 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event = %v, want cc.locked", ev.Kind)
		}
	default:
		t.Fatal("fallback path did not lock on bare Aloha codeword")
	}
}
