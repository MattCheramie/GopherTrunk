package purego

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// slowStopTransport makes StopBulkIn take a while, like a real host
// reaping in-flight URBs, so the window between a stream's ctx cancel and
// its teardown is wide enough to hit deterministically.
type slowStopTransport struct {
	*usb.MockTransport
	delay time.Duration
}

func (s *slowStopTransport) StopBulkIn() error {
	time.Sleep(s.delay)
	return s.MockTransport.StopBulkIn()
}

func resetBufferScript(n int) []usb.CtrlExchange {
	var out []usb.CtrlExchange
	for range n {
		out = append(out,
			usb.CtrlExchange{In: false, BRequest: 0, WValue: rtl2832u.USBEpaCtl, WIndex: uint16(rtl2832u.BlockUSB)<<8 | 0x10, Data: []byte{0x10, 0x02}},
			usb.CtrlExchange{In: false, BRequest: 0, WValue: rtl2832u.USBEpaCtl, WIndex: uint16(rtl2832u.BlockUSB)<<8 | 0x10, Data: []byte{0x00, 0x00}},
		)
	}
	return out
}

// TestStreamIQ_ReopenAfterCancelWaitsForTeardown is the issue #1261
// regression: the conventional scanner cancels a dwell's stream and opens
// the next channel's at once. The teardown runs on a goroutine, so the
// reopen used to race it and fail with "stream already active" (seen in
// the reporter's multi-SDR log). A cancelled stream must be waited out.
func TestStreamIQ_ReopenAfterCancelWaitsForTeardown(t *testing.T) {
	mock := usb.NewMockTransport()
	mock.Script = resetBufferScript(2)
	d := newDeviceWithFakeTuner(&slowStopTransport{MockTransport: mock, delay: 100 * time.Millisecond}, &fakeTuner{})

	ctx1, cancel1 := context.WithCancel(context.Background())
	ch1, err := d.StreamIQ(ctx1)
	if err != nil {
		t.Fatalf("first StreamIQ: %v", err)
	}
	cancel1()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	ch2, err := d.StreamIQ(ctx2)
	if err != nil {
		t.Fatalf("StreamIQ right after cancelling the previous stream: %v", err)
	}
	// The old stream's channel is closed by its own teardown, never the
	// new one's.
	select {
	case _, ok := <-ch1:
		if ok {
			for range ch1 {
			}
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled stream's channel never closed")
	}
	if ch2 == nil {
		t.Fatal("nil channel for the reopened stream")
	}
}

// TestStreamIQ_ReopenWaitHonoursCallerCtx: the wait for a previous
// teardown is bounded by the caller's ctx, so shutdown cannot hang here.
func TestStreamIQ_ReopenWaitHonoursCallerCtx(t *testing.T) {
	mock := usb.NewMockTransport()
	mock.Script = resetBufferScript(2)
	d := newDeviceWithFakeTuner(&slowStopTransport{MockTransport: mock, delay: 500 * time.Millisecond}, &fakeTuner{})

	ctx1, cancel1 := context.WithCancel(context.Background())
	if _, err := d.StreamIQ(ctx1); err != nil {
		t.Fatalf("first StreamIQ: %v", err)
	}
	cancel1()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	start := time.Now()
	if _, err := d.StreamIQ(ctx2); err == nil {
		t.Fatal("StreamIQ succeeded while the previous stream was still tearing down past the caller's deadline")
	}
	if el := time.Since(start); el > 300*time.Millisecond {
		t.Errorf("StreamIQ waited %v, want it bounded by the caller's ctx", el)
	}
}
