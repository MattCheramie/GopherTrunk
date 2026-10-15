package purego

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// TestStreamErrNamesUSBDeathCause is the #1256 regression: on the
// reporter's phone a second `wmbus` run died with a bare "IQ stream
// ended". The driver records why the bulk-IN reaper died
// (StreamDeadCause), but the CLI tools ask sdr.StreamErr, which only
// knew the rtl_tcp extension, so the USB error was never printed.
func TestStreamErrNamesUSBDeathCause(t *testing.T) {
	mock := usb.NewMockTransport()
	mock.Script = resetBufferScript(1)
	mock.BulkPackets = [][]byte{{127, 128, 127, 128}}
	mock.BulkSimulateDeath = true
	cause := errors.New("bulk-in: no such device")
	mock.BulkDeathErr = cause
	d := newDeviceWithFakeTuner(mock, &fakeTuner{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := d.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	deadline := time.After(time.Second)
	for open := true; open; {
		select {
		case _, open = <-ch:
		case <-deadline:
			t.Fatal("channel did not close after the reaper died")
		}
	}
	if got := sdr.StreamErr(d); !errors.Is(got, cause) {
		t.Fatalf("sdr.StreamErr after a USB stream death = %v, want it to wrap %q", got, cause)
	}
}

// TestStreamErrIsNilAfterCallerCancel: a stream the caller cancelled ended
// normally, so there is no reason to report.
func TestStreamErrIsNilAfterCallerCancel(t *testing.T) {
	mock := usb.NewMockTransport()
	mock.Script = resetBufferScript(1)
	d := newDeviceWithFakeTuner(mock, &fakeTuner{})
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := d.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	cancel()
	for range ch {
	}
	if got := sdr.StreamErr(d); got != nil {
		t.Fatalf("sdr.StreamErr after a caller cancel = %v, want nil", got)
	}
}
