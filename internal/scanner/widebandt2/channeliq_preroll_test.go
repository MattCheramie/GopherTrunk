package widebandt2

import (
	"testing"
)

// ramp returns n samples whose real part is their stream position, so a
// delivered chunk shows exactly which samples it holds.
func ramp(from, n int) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(float32(from+i), 0)
	}
	return out
}

func firstChunk(t *testing.T, ch <-chan []complex64) []complex64 {
	t.Helper()
	select {
	case c := <-ch:
		return c
	default:
		return nil
	}
}

// The conventional pre-roll (issue #1288): a counted subscriber learns the
// stream position of its first sample, an onset mark hands the next preroll
// subscriber exactly the samples from that position on (across the ring's
// wrap), once, and plain subscribers never take it.
func TestChannelFanoutPrerollFromOnset(t *testing.T) {
	f := newChannelFanout(nil, "WB1", 1, 48_000)
	f.enablePreroll(100)

	f.broadcast(ramp(0, 70))
	_, start, unsubC := f.register(false)
	defer unsubC()
	if start != 70 {
		t.Fatalf("counted subscriber starts at %d, want 70", start)
	}
	f.broadcast(ramp(70, 60)) // 130 samples so far: the ring wrapped

	f.arm(50)
	plain, unsubP := f.subscribe()
	defer unsubP()
	if c := firstChunk(t, plain); c != nil {
		t.Fatalf("plain subscriber got a pre-roll of %d samples", len(c))
	}
	pre, _, unsub := f.register(true)
	defer unsub()
	c := firstChunk(t, pre)
	if len(c) != 80 || real(c[0]) != 50 || real(c[79]) != 129 {
		t.Fatalf("pre-roll = %d samples from %v to %v, want positions 50..129", len(c), first(c), last(c))
	}
	again, _, unsub2 := f.register(true)
	defer unsub2()
	if c := firstChunk(t, again); c != nil {
		t.Fatalf("the onset mark was handed out twice")
	}

	// An onset older than the ring hands out what the ring still holds.
	f.arm(0)
	old, _, unsub3 := f.register(true)
	defer unsub3()
	if c := firstChunk(t, old); len(c) != 100 || real(c[0]) != 30 {
		t.Fatalf("old onset: %d samples from %v, want the ring's 100 from 30", len(c), first(c))
	}

	// A mark nobody collected before a ring's worth of IQ went by is stale.
	f.arm(129)
	f.broadcast(ramp(130, 101))
	stale, _, unsub4 := f.register(true)
	defer unsub4()
	if c := firstChunk(t, stale); c != nil {
		t.Fatalf("a stale onset mark replayed %d samples", len(c))
	}
}

// Without enablePreroll (every non-conventional channel) a mark is inert.
func TestChannelFanoutNoPrerollByDefault(t *testing.T) {
	f := newChannelFanout(nil, "WB1", 1, 48_000)
	f.broadcast(ramp(0, 50))
	f.arm(10)
	ch, _, unsub := f.register(true)
	defer unsub()
	if c := firstChunk(t, ch); c != nil {
		t.Fatalf("a channel without a ring delivered %d pre-roll samples", len(c))
	}
}

func first(c []complex64) any {
	if len(c) == 0 {
		return nil
	}
	return real(c[0])
}

func last(c []complex64) any {
	if len(c) == 0 {
		return nil
	}
	return real(c[len(c)-1])
}
