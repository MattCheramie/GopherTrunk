package widebandt2

import "testing"

// TestChannelFanoutDropsReadableWhileSubscribed pins the live drop read the
// voice source's StreamIQWithLoss relies on (issue #1288): the composer reads
// it at call end, before it unsubscribes, so it must reflect drops already,
// and count only this subscriber's.
func TestChannelFanoutDropsReadableWhileSubscribed(t *testing.T) {
	f := newChannelFanout(nil, "WB1", 1, 48_000)
	_, _, unsub, dropped := f.registerWithDrops(true)
	other, _, unsubOther := f.register(false)
	defer unsubOther()
	depth := channelTapBudget(48_000)
	for i := 0; i < depth+5; i++ { // nobody reads: the last 5 overflow
		f.broadcast(ramp(0, 16))
		select {
		case <-other: // the other subscriber keeps up
		default:
		}
	}
	if got := dropped(); got != 5 {
		t.Fatalf("live drop read = %d, want 5", got)
	}
	if got := unsub(); got != 5 {
		t.Fatalf("unsubscribe reports %d, want 5", got)
	}
}
