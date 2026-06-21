package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// fakeProbeDriver is a minimal sdr.Driver whose Open behaviour is supplied
// per-test so probeDevice's timeout/error/success paths can be exercised
// without real hardware.
type fakeProbeDriver struct {
	open func(idx int) (sdr.Device, error)
}

func (d *fakeProbeDriver) Name() string                     { return "fake" }
func (d *fakeProbeDriver) Enumerate() ([]sdr.Info, error)   { return nil, nil }
func (d *fakeProbeDriver) Open(idx int) (sdr.Device, error) { return d.open(idx) }

// fakeProbeDevice reports a fixed Info and records that Close was called.
type fakeProbeDevice struct {
	info   sdr.Info
	closed chan struct{}
}

func (d *fakeProbeDevice) Info() sdr.Info             { return d.info }
func (d *fakeProbeDevice) SetCenterFreq(uint32) error { return nil }
func (d *fakeProbeDevice) SetSampleRate(uint32) error { return nil }
func (d *fakeProbeDevice) SetGain(int) error          { return nil }
func (d *fakeProbeDevice) SetPPM(int) error           { return nil }
func (d *fakeProbeDevice) SetBiasTee(bool) error      { return nil }
func (d *fakeProbeDevice) StreamIQ(context.Context) (<-chan []complex64, error) {
	return nil, nil
}
func (d *fakeProbeDevice) Close() error {
	if d.closed != nil {
		close(d.closed)
	}
	return nil
}

func TestProbeDeviceSuccess(t *testing.T) {
	want := sdr.Info{TunerName: "MAX2839+MAX5864", Gains: []int{0, 80, 160}}
	closed := make(chan struct{})
	drv := &fakeProbeDriver{open: func(int) (sdr.Device, error) {
		return &fakeProbeDevice{info: want, closed: closed}, nil
	}}
	got, err := probeDevice(drv, 0, time.Second)
	if err != nil {
		t.Fatalf("probeDevice: unexpected error: %v", err)
	}
	if got.TunerName != want.TunerName {
		t.Errorf("TunerName = %q, want %q", got.TunerName, want.TunerName)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Error("Close was not called on the probed device")
	}
}

func TestProbeDeviceOpenError(t *testing.T) {
	wantErr := errors.New("permission denied")
	drv := &fakeProbeDriver{open: func(int) (sdr.Device, error) {
		return nil, wantErr
	}}
	if _, err := probeDevice(drv, 0, time.Second); !errors.Is(err, wantErr) {
		t.Fatalf("probeDevice error = %v, want %v", err, wantErr)
	}
}

// TestProbeDeviceTimeout is the regression for `sdr list --probe` hanging on a
// wedged device: Open blocks forever, and probeDevice must still return a
// timeout error promptly rather than blocking the caller.
func TestProbeDeviceTimeout(t *testing.T) {
	drv := &fakeProbeDriver{open: func(int) (sdr.Device, error) {
		select {} // never returns
	}}
	start := time.Now()
	_, err := probeDevice(drv, 0, 50*time.Millisecond)
	if err == nil {
		t.Fatal("probeDevice: expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want it to mention a timeout", err.Error())
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("probeDevice took %s; should return near the 50ms timeout", elapsed)
	}
}

// TestFormatSDRTableFullSerial is the regression for the reported bug: a
// HackRF's full 32-hex serial (and the full tuner/product strings) must print
// untruncated rather than being cut to the all-zero 16-char prefix.
func TestFormatSDRTableFullSerial(t *testing.T) {
	const serial = "0000000000000000457863c8284a625f"
	out := formatSDRTable([]sdr.Info{{
		Driver:    "hackrf",
		Index:     0,
		Serial:    serial,
		TunerName: "MAX2839+MAX5864",
		Product:   "HackRF One",
		Gains:     []int{0, 80, 160, 240, 320, 400, 480, 560},
	}})
	for _, want := range []string{serial, "MAX2839+MAX5864", "HackRF One"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q; got:\n%s", want, out)
		}
	}
	// The all-zero prefix must not appear as a standalone (truncated) column.
	if strings.Contains(out, "0000000000000000  ") {
		t.Errorf("serial appears truncated to its zero prefix; got:\n%s", out)
	}
}
