package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/powersweep"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtltcp"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// hangUpRTLTCP is an rtl_tcp server that sends its header and a little IQ,
// then closes the connection: the #1230 reporter's driver app ending the
// session as `power` started its first sweep.
func hangUpRTLTCP(t *testing.T, iqBytes int) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		var hdr [12]byte
		copy(hdr[:4], rtltcp.Magic[:])
		binary.BigEndian.PutUint32(hdr[4:8], 5)
		binary.BigEndian.PutUint32(hdr[8:12], 29)
		_, _ = c.Write(hdr[:])
		iq := make([]byte, iqBytes)
		for i := range iq {
			iq[i] = 127
		}
		_, _ = c.Write(iq)
		time.Sleep(100 * time.Millisecond) // let the client's setters land, then hang up
	}()
	return ln.Addr().String()
}

// TestPowerExplainsRTLTCPHangUp is the #1230 regression: the sweep failed with
// a bare "unexpected EOF" when the rtl_tcp server dropped the connection. The
// error `power` reports must now say the server closed it.
func TestPowerExplainsRTLTCPHangUp(t *testing.T) {
	const rate = 240_000
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dev, _, err := openPowerDevice(hangUpRTLTCP(t, 4096), "", log)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	r, _ := powersweep.ParseRange("100.0M:100.6M:2k")
	plan, err := powersweep.NewPlan(r, rate, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	src, err := newPowerIQSource(ctx, dev, rate, 10*time.Millisecond, log)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = runPowerSweeps(ctx, src, plan, powersweep.Integration{Samples: 20 * plan.FFTSize}, powerSchedule{every: time.Second, single: true},
		bufio.NewWriter(&out), nil, time.Now, sleepCtx)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("sweep against a hung-up server = %v, want io.ErrUnexpectedEOF from the short stream", err)
	}
	err = iqStreamEnded(dev, err)
	if !errors.Is(err, rtltcp.ErrServerClosed) {
		t.Fatalf("power would report %q; want it to say the rtl_tcp server closed the connection", err)
	}
	if !strings.Contains(err.Error(), "4096 bytes") {
		t.Errorf("error %q does not say how much IQ arrived", err)
	}
}

// TestCaptureOverRTLTCPReportsHangUp covers `capture -rtltcp` (#1256: the
// reporter could only reach the dongle through an rtl_tcp driver app on the
// phone): it records what the server sent, and a hang-up before -seconds is
// reported as the server closing, not just "IQ stream ended".
func TestCaptureOverRTLTCPReportsHangUp(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dev, _, err := openPowerDevice(hangUpRTLTCP(t, 20000), "", log)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	stream, err := dev.StreamIQ(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/meters.cu8"
	written, _, err := captureToFiles(context.Background(), []string{path}, siglab.FormatU8, stream, 240_000, 10, []*ccdecoder.Downconverter{nil})
	if !errors.Is(err, errCaptureStreamEnded) {
		t.Fatalf("capture against a hung-up server = %v, want errCaptureStreamEnded", err)
	}
	if written[0] != 10000 {
		t.Fatalf("wrote %d samples, want the 10000 the server sent", written[0])
	}
	if se := sdr.StreamErr(dev); !errors.Is(se, rtltcp.ErrServerClosed) {
		t.Fatalf("StreamErr = %v, want ErrServerClosed", se)
	}
}

// TestCaptureDeliveredNote: a capture that took longer than its sample count
// says the source could not sustain the rate (#1256's "how can the maximum
// sample rate be determined?"); one that kept up says nothing.
func TestCaptureDeliveredNote(t *testing.T) {
	if n := captureDeliveredNote(24_000_000, 10*time.Second, 2_400_000); n != "" {
		t.Fatalf("kept-up capture noted %q", n)
	}
	n := captureDeliveredNote(24_000_000, 14*time.Second, 2_400_000)
	if !strings.Contains(n, "1714286 S/s of the requested 2400000") {
		t.Fatalf("slow capture note = %q", n)
	}
	if n := captureDeliveredNote(1000, 100*time.Millisecond, 2_400_000); n != "" {
		t.Fatalf("too-short capture noted %q", n)
	}
}
