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

// fakeProbeOpenerDriver additionally implements sdr.ProbeOpener so the
// probe fast path can be exercised: openProbe is used by probeDevice when
// present, and open records whether the daemon path was taken instead.
type fakeProbeOpenerDriver struct {
	open      func(idx int) (sdr.Device, error)
	openProbe func(idx int) (sdr.Device, error)
}

func (d *fakeProbeOpenerDriver) Name() string                   { return "fake-probe" }
func (d *fakeProbeOpenerDriver) Enumerate() ([]sdr.Info, error) { return nil, nil }
func (d *fakeProbeOpenerDriver) Open(idx int) (sdr.Device, error) {
	return d.open(idx)
}
func (d *fakeProbeOpenerDriver) OpenProbe(idx int) (sdr.Device, error) {
	return d.openProbe(idx)
}

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

// TestProbeDevicePrefersProbeOpener is the regression for the issue #1135
// follow-up: `sdr list --probe` must use a driver's no-reset OpenProbe
// fast path (sdr.ProbeOpener) rather than the daemon Open with its
// reset+retry envelope, because on macOS that envelope re-enumerates the
// device and made probing one dongle perturb its sibling (timeouts +
// which-dongle-probes swapping). probeDevice must call OpenProbe and not
// Open when the driver implements ProbeOpener.
func TestProbeDevicePrefersProbeOpener(t *testing.T) {
	want := sdr.Info{TunerName: "R820T2", Gains: []int{0, 496}}
	var openCalled, probeCalled bool
	drv := &fakeProbeOpenerDriver{
		open: func(int) (sdr.Device, error) {
			openCalled = true
			return nil, errors.New("daemon Open must not be used for probing")
		},
		openProbe: func(int) (sdr.Device, error) {
			probeCalled = true
			return &fakeProbeDevice{info: want}, nil
		},
	}
	got, err := probeDevice(drv, 0, time.Second)
	if err != nil {
		t.Fatalf("probeDevice: unexpected error: %v", err)
	}
	if openCalled {
		t.Error("probeDevice called Driver.Open; it must use the no-reset OpenProbe fast path (issue #1135)")
	}
	if !probeCalled {
		t.Error("probeDevice did not call OpenProbe despite the driver implementing sdr.ProbeOpener")
	}
	if got.TunerName != want.TunerName {
		t.Errorf("TunerName = %q, want %q", got.TunerName, want.TunerName)
	}
}

// A driver that does NOT implement sdr.ProbeOpener must still be probed
// via Open — the fast path is an optional extension, not a requirement.
func TestProbeDeviceFallsBackToOpen(t *testing.T) {
	want := sdr.Info{TunerName: "FC0013"}
	var openCalled bool
	drv := &fakeProbeDriver{open: func(int) (sdr.Device, error) {
		openCalled = true
		return &fakeProbeDevice{info: want}, nil
	}}
	got, err := probeDevice(drv, 0, time.Second)
	if err != nil {
		t.Fatalf("probeDevice: unexpected error: %v", err)
	}
	if !openCalled {
		t.Error("probeDevice did not fall back to Driver.Open for a driver without ProbeOpener")
	}
	if got.TunerName != want.TunerName {
		t.Errorf("TunerName = %q, want %q", got.TunerName, want.TunerName)
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
