package airspy

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// TestStreamIQRecordsDeadCause pins the diagnostic fix: when the USB reaper
// dies (the transport fires onStreamDead with a concrete error — here the
// stall-watchdog's usb.ErrStreamStalled), the driver must record it so
// StreamDeadCause() surfaces the real reason. Previously onStreamDead discarded
// its argument, so an operator only ever saw the generic "IQ stream closed
// unexpectedly" with no way to tell a stall from a disconnect.
func TestStreamIQRecordsDeadCause(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
	}
	mt.BulkPackets = [][]byte{make([]byte, 64)}
	mt.BulkSimulateDeath = true
	mt.BulkDeathErr = usb.ErrStreamStalled

	if c := dev.StreamDeadCause(); c != nil {
		t.Fatalf("StreamDeadCause before streaming = %v, want nil", c)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	// Drain until the driver closes the channel — the death path stores the
	// cause before it closes `out`, so no sleep/poll is needed.
	for range ch {
	}

	if got := dev.StreamDeadCause(); !errors.Is(got, usb.ErrStreamStalled) {
		t.Fatalf("StreamDeadCause after reaper death = %v, want usb.ErrStreamStalled", got)
	}
}

// airspyDebugEnum builds a MockEnumerator whose opened transports carry the
// samplerate-table script Open reads, so openDevice succeeds. It records the
// last raw mock returned so a test can check whether the driver wrapped it.
func airspyDebugEnum(record func(*usb.MockTransport)) *usb.MockEnumerator {
	return &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 4, VID: vidAirspy, PID: pidAirspy, Serial: "AS1", Product: "Airspy R2", Path: "mock/1"},
		},
		OpenFunc: func(usb.Descriptor) (*usb.MockTransport, error) {
			mt := usb.NewMockTransport()
			count := make([]byte, 4)
			binary.LittleEndian.PutUint32(count, 1)
			list := make([]byte, 4)
			binary.LittleEndian.PutUint32(list, 10_000_000)
			mt.Script = []usb.CtrlExchange{
				{BRequest: reqReceiverMode, WValue: receiverModeOff},
				{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 0, Reply: count, N: 4},
				{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 1, Reply: list, N: 4},
			}
			record(mt)
			return mt, nil
		},
	}
}

// TestOpenWrapsTransportForUSBDebug pins that airspy honours RTLSDR_DEBUG_USB by
// wrapping its transport with usb.MaybeWrapDebug, so control transfers
// (SET_SAMPLERATE / SET_FREQ / RECEIVER_MODE / gain) are traced. The Airspy
// shares the RTL-SDR USB transport but never wrapped it, so its control setup
// was invisible even with debugging on. With the env unset the transport is the
// raw handle; with it set the device holds the wrapper, not the raw mock.
func TestOpenWrapsTransportForUSBDebug(t *testing.T) {
	openOne := func(t *testing.T) (inner usb.Transport, raw *usb.MockTransport) {
		t.Helper()
		drv := New(airspyDebugEnum(func(mt *usb.MockTransport) { raw = mt }))
		dev, err := drv.Open(0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if raw == nil {
			t.Fatal("OpenFunc never ran")
		}
		return dev.(*Device).t, raw
	}

	t.Run("debug off leaves transport unwrapped", func(t *testing.T) {
		t.Setenv("RTLSDR_DEBUG_USB", "")
		inner, raw := openOne(t)
		if inner != usb.Transport(raw) {
			t.Fatal("RTLSDR_DEBUG_USB unset but transport was wrapped anyway")
		}
	})

	t.Run("debug on wraps transport", func(t *testing.T) {
		t.Setenv("RTLSDR_DEBUG_USB", "1")
		inner, raw := openOne(t)
		if inner == usb.Transport(raw) {
			t.Fatal("RTLSDR_DEBUG_USB set but transport was not wrapped by MaybeWrapDebug")
		}
	})
}
