package soapyremote

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// capHandler is a minimal slog.Handler that records WARN+ messages and the
// integer attributes we assert on, so the overrun throttle's rate-limited
// summaries can be inspected without a real socket.
type capHandler struct {
	records []capRecord
}

type capRecord struct {
	level    slog.Level
	msg      string
	devOver  int64
	hostDrop int64
}

func (h *capHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *capHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *capHandler) WithGroup(string) slog.Handler            { return h }
func (h *capHandler) Handle(_ context.Context, r slog.Record) error {
	rec := capRecord{level: r.Level, msg: r.Message}
	asInt := func(v slog.Value) int64 {
		if v.Kind() == slog.KindUint64 {
			return int64(v.Uint64())
		}
		return v.Int64()
	}
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "device_overflows":
			rec.devOver = asInt(a.Value)
		case "host_drops":
			rec.hostDrop = asInt(a.Value)
		}
		return true
	})
	h.records = append(h.records, rec)
	return nil
}

func (h *capHandler) warns() []capRecord {
	var w []capRecord
	for _, r := range h.records {
		if r.level == slog.LevelWarn {
			w = append(w, r)
		}
	}
	return w
}

// TestOverrunThrottleRateLimits is the visibility regression: a sustained
// overrun storm must surface as a periodic WARN with running counts (it used to
// be a silent per-datagram DEBUG line), and it must NOT log one line per drop.
func TestOverrunThrottleRateLimits(t *testing.T) {
	h := &capHandler{}
	now := time.Unix(0, 0)
	tr := &overrunThrottle{
		log:      slog.New(h),
		addr:     "test",
		interval: 5 * time.Second,
		now:      func() time.Time { return now },
	}

	// First overflow logs immediately so the operator sees it without delay.
	tr.deviceOverflow()
	if got := len(h.warns()); got != 1 {
		t.Fatalf("after first overflow: %d WARNs, want 1", got)
	}
	if w := h.warns()[0]; w.devOver != 1 {
		t.Errorf("first WARN device_overflows = %d, want 1", w.devOver)
	}

	// A burst within the interval accumulates silently (no new WARNs).
	for i := 0; i < 50; i++ {
		tr.deviceOverflow()
	}
	for i := 0; i < 10; i++ {
		tr.hostDrop()
	}
	if got := len(h.warns()); got != 1 {
		t.Fatalf("during interval: %d WARNs, want 1 (throttled)", got)
	}

	// After the interval elapses the next event flushes the accumulated batch.
	now = now.Add(6 * time.Second)
	tr.deviceOverflow()
	w := h.warns()
	if len(w) != 2 {
		t.Fatalf("after interval: %d WARNs, want 2", len(w))
	}
	// 50 mid-interval + 1 that triggered the flush = 51 overflows, 10 host drops.
	if w[1].devOver != 51 || w[1].hostDrop != 10 {
		t.Errorf("second WARN counts = (overflows %d, drops %d), want (51, 10)", w[1].devOver, w[1].hostDrop)
	}
}

// TestOverrunThrottleFlush verifies a trailing batch that never reaches the
// interval is still reported on teardown (maybeWarn(true)).
func TestOverrunThrottleFlush(t *testing.T) {
	h := &capHandler{}
	now := time.Unix(100, 0)
	tr := &overrunThrottle{
		log:      slog.New(h),
		addr:     "test",
		interval: 5 * time.Second,
		now:      func() time.Time { return now },
	}
	tr.deviceOverflow()  // logs immediately (count reset)
	tr.deviceOverflow()  // within interval, accumulates silently
	tr.hostDrop()        // ditto
	if got := len(h.warns()); got != 1 {
		t.Fatalf("before flush: %d WARNs, want 1", got)
	}
	tr.maybeWarn(true) // teardown flush ignores the interval
	w := h.warns()
	if len(w) != 2 {
		t.Fatalf("after flush: %d WARNs, want 2", len(w))
	}
	if w[1].devOver != 1 || w[1].hostDrop != 1 {
		t.Errorf("flush WARN counts = (overflows %d, drops %d), want (1, 1)", w[1].devOver, w[1].hostDrop)
	}

	// A flush with nothing pending is a no-op (no spurious teardown WARN).
	tr.maybeWarn(true)
	if got := len(h.warns()); got != 2 {
		t.Errorf("empty flush logged: %d WARNs, want 2", got)
	}
}

// TestSendOrDropNeverBlocks is the drop-oldest regression: when the DSP consumer
// stalls and the bounded channel fills, the read loop must NOT block (a blocked
// reader stalls ACKs and drives device-side overruns). sendOrDrop must shed the
// oldest chunk, count it, and keep the freshest data — returning promptly.
func TestSendOrDropNeverBlocks(t *testing.T) {
	d := &device{addr: "test"}
	h := &capHandler{}
	tr := &overrunThrottle{log: slog.New(h), addr: "test", interval: time.Hour, now: time.Now}
	out := make(chan []complex64, 2) // tiny bounded channel, never drained

	chunk := func(tag float32) []complex64 { return []complex64{complex(tag, 0)} }

	// Fill the channel to capacity, then push two more with no consumer.
	done := make(chan bool, 1)
	go func() {
		ok := d.sendOrDrop(context.Background(), out, chunk(0), tr) &&
			d.sendOrDrop(context.Background(), out, chunk(1), tr) &&
			d.sendOrDrop(context.Background(), out, chunk(2), tr) && // channel now full
			d.sendOrDrop(context.Background(), out, chunk(3), tr) && // drops oldest (0)
			d.sendOrDrop(context.Background(), out, chunk(4), tr) //   drops oldest (1)
		done <- ok
	}()

	select {
	case ok := <-done:
		if !ok {
			t.Fatal("sendOrDrop returned false without ctx cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("sendOrDrop blocked with a full channel — the read loop would stall the radio")
	}

	if tr.hostDrops != 2 {
		t.Errorf("host_drops = %d, want 2 (two oldest chunks shed)", tr.hostDrops)
	}
	// The channel retains exactly its capacity, holding the two freshest chunks
	// (3 then 4) in FIFO order — the stale 0 and 1 were the ones shed.
	if len(out) != cap(out) {
		t.Fatalf("channel holds %d, want %d (full)", len(out), cap(out))
	}
	if v := real((<-out)[0]); v != 3 {
		t.Errorf("oldest retained chunk = %v, want 3 (chunks 0 and 1 should have been dropped)", v)
	}
	if v := real((<-out)[0]); v != 4 {
		t.Errorf("newest retained chunk = %v, want 4", v)
	}
}

// TestSendOrDropCtxCancelUnblocks confirms a cancelled context lets the loop
// unwind even when the channel is full (no consumer will ever drain it).
func TestSendOrDropCtxCancelUnblocks(t *testing.T) {
	d := &device{addr: "test"}
	tr := &overrunThrottle{log: slog.New(&capHandler{}), addr: "test", interval: time.Hour, now: time.Now}
	out := make(chan []complex64) // unbuffered, no consumer ⇒ send never ready

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan bool, 1)
	go func() { done <- d.sendOrDrop(ctx, out, []complex64{0}, tr) }()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("sendOrDrop returned true on a cancelled ctx with no consumer")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("sendOrDrop did not unwind on ctx cancel")
	}
}
