package purego

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/tuners"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// wedgedTransport models the issue #753 failure: once wedged, every control
// transfer fails with EPIPE, as on the reporter's Pi 5, until a USB reset
// (the software replug) clears it.
type wedgedTransport struct {
	mu      sync.Mutex
	wedged  bool
	resets  int
	claimed int
}

func (w *wedgedTransport) wedge() {
	w.mu.Lock()
	w.wedged = true
	w.mu.Unlock()
}

func (w *wedgedTransport) ctrlErr() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wedged {
		return syscall.EPIPE
	}
	return nil
}

func (w *wedgedTransport) ControlIn(_ uint8, _, _ uint16, n int, _ int) ([]byte, error) {
	if err := w.ctrlErr(); err != nil {
		return nil, err
	}
	return make([]byte, n), nil
}

func (w *wedgedTransport) ControlOut(_ uint8, _, _ uint16, _ []byte, _ int) error {
	return w.ctrlErr()
}

func (w *wedgedTransport) ClaimInterface(int) error {
	w.mu.Lock()
	w.claimed++
	w.mu.Unlock()
	return nil
}
func (w *wedgedTransport) ReleaseInterface(int) error { return nil }
func (w *wedgedTransport) StartBulkIn(byte, int, int, func([]byte), func(error)) error {
	return nil
}
func (w *wedgedTransport) StopBulkIn() error { return nil }
func (w *wedgedTransport) Close() error      { return nil }
func (w *wedgedTransport) Reset() error {
	w.mu.Lock()
	w.resets++
	w.wedged = false
	w.mu.Unlock()
	return nil
}

func (w *wedgedTransport) resetCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.resets
}

// i2cTuner is a fakeTuner whose SetFreq talks to its chip over the demod's
// I2C bridge, as a real R82xx retune does, so a wedged control pipe fails it.
type i2cTuner struct {
	*fakeTuner
	demod *rtl2832u.Demod
}

func (t *i2cTuner) SetFreq(hz uint32) error {
	if err := t.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer t.demod.SetI2CRepeater(false)
	return t.fakeTuner.SetFreq(hz)
}

// newWedgeableDevice builds a Device on a wedgedTransport whose recovery hook
// brings up a fresh demod + i2cTuner, returning every tuner it made.
func newWedgeableDevice(t *testing.T) (*Device, *wedgedTransport, func() []*i2cTuner) {
	t.Helper()
	old := controlRecoverySettle
	controlRecoverySettle = 0
	t.Cleanup(func() { controlRecoverySettle = old })

	tr := &wedgedTransport{}
	var mu sync.Mutex
	var made []*i2cTuner
	mk := func() (*rtl2832u.Demod, tuners.Tuner) {
		demod := rtl2832u.New(tr)
		tn := &i2cTuner{fakeTuner: &fakeTuner{}, demod: demod}
		mu.Lock()
		made = append(made, tn)
		mu.Unlock()
		return demod, tn
	}
	demod, tn := mk()
	d := &Device{transport: tr, demod: demod, tuner: tn}
	d.rebringup = func() (*rtl2832u.Demod, tuners.Tuner, error) {
		dm, tt := mk()
		return dm, tt, nil
	}
	return d, tr, func() []*i2cTuner {
		mu.Lock()
		defer mu.Unlock()
		return append([]*i2cTuner(nil), made...)
	}
}

// TestControlStallWithoutStreamRecoversByReset is the issue #753 regression:
// a control pipe that stays stalled used to fail every retune until the
// dongle was replugged. With no stream running, the device now resets,
// re-runs bring-up, restores what was programmed, and the retune succeeds.
func TestControlStallWithoutStreamRecoversByReset(t *testing.T) {
	d, tr, tunersMade := newWedgeableDevice(t)
	if err := d.SetGain(200); err != nil {
		t.Fatalf("SetGain: %v", err)
	}
	if err := d.SetCenterFreq(146_520_000); err != nil {
		t.Fatalf("first SetCenterFreq: %v", err)
	}

	tr.wedge()
	if err := d.SetCenterFreq(154_130_000); err != nil {
		t.Fatalf("SetCenterFreq on a wedged pipe: %v (want recovery by USB reset)", err)
	}
	if got := tr.resetCount(); got != 1 {
		t.Errorf("USB resets = %d, want 1", got)
	}
	if got := d.ControlRecoveries(); got != 1 {
		t.Errorf("ControlRecoveries = %d, want 1", got)
	}
	made := tunersMade()
	if len(made) != 2 {
		t.Fatalf("bring-ups = %d, want the original plus one recovery", len(made))
	}
	fresh := made[1]
	if got := fresh.gainCalls; len(got) != 1 || got[0] != 200 {
		t.Errorf("gain restored on the re-opened tuner = %v, want [200]", got)
	}
	// The last frequency is restored, then the failed retune is retried.
	if got := fresh.freqCalls; len(got) != 2 || got[0] != 146_520_000 || got[1] != 154_130_000 {
		t.Errorf("re-opened tuner freq calls = %v, want [146520000 154130000]", got)
	}
}

// A stall while an IQ stream is running is never reset: the reset would kill
// the stream mid-call. The error surfaces unchanged (still matchable).
func TestControlStallWithStreamRunningDoesNotReset(t *testing.T) {
	d, tr, _ := newWedgeableDevice(t)
	d.streaming.Store(true)
	tr.wedge()
	err := d.SetCenterFreq(154_130_000)
	if !errors.Is(err, syscall.EPIPE) {
		t.Fatalf("err = %v, want EPIPE", err)
	}
	if got := tr.resetCount(); got != 0 {
		t.Errorf("USB resets = %d with a stream running, want 0", got)
	}
}

// Recoveries are rate-limited, so a dongle a reset cannot fix is not
// reset-stormed on every scanner lap.
func TestControlStallRecoveryIsRateLimited(t *testing.T) {
	d, tr, _ := newWedgeableDevice(t)
	tr.wedge()
	if err := d.SetCenterFreq(154_130_000); err != nil {
		t.Fatalf("first recovery: %v", err)
	}
	tr.wedge()
	if err := d.SetCenterFreq(155_000_000); !errors.Is(err, syscall.EPIPE) {
		t.Fatalf("second stall inside the cooldown: err = %v, want EPIPE", err)
	}
	if got := tr.resetCount(); got != 1 {
		t.Errorf("USB resets = %d, want 1 (cooldown)", got)
	}
}

// A non-control error (a validation failure, a gone device) is not a reason
// to reset.
func TestControlNonStallErrorDoesNotReset(t *testing.T) {
	d, tr, _ := newWedgeableDevice(t)
	d.tuner.(*i2cTuner).freqErr = usb.ErrDeviceGone
	if err := d.SetCenterFreq(154_130_000); !errors.Is(err, usb.ErrDeviceGone) {
		t.Fatalf("err = %v, want ErrDeviceGone", err)
	}
	if got := tr.resetCount(); got != 0 {
		t.Errorf("USB resets = %d, want 0", got)
	}
}

// stopFlagTransport records when StopBulkIn has finished.
type stopFlagTransport struct {
	*slowStopTransport
	stopped atomic.Bool
}

func (s *stopFlagTransport) StopBulkIn() error {
	err := s.slowStopTransport.StopBulkIn()
	s.stopped.Store(true)
	return err
}

// orderTuner notes whether the stream teardown had finished when it was
// retuned.
type orderTuner struct {
	*fakeTuner
	tr                  *stopFlagTransport
	tunedDuringTeardown atomic.Bool
}

func (o *orderTuner) SetFreq(hz uint32) error {
	if !o.tr.stopped.Load() {
		o.tunedDuringTeardown.Store(true)
	}
	return o.fakeTuner.SetFreq(hz)
}

// TestRetuneWaitsForCancelledStreamTeardown is the other half of #753: the
// conventional scanner cancels a dwell's stream and retunes at once, and the
// teardown (StopBulkIn reaping the URBs) runs on its own goroutine, so the
// retune's control transfers raced it. They now wait for it.
func TestRetuneWaitsForCancelledStreamTeardown(t *testing.T) {
	mock := usb.NewMockTransport()
	mock.Script = resetBufferScript(1)
	tr := &stopFlagTransport{slowStopTransport: &slowStopTransport{MockTransport: mock, delay: 150 * time.Millisecond}}
	tn := &orderTuner{fakeTuner: &fakeTuner{}, tr: tr}
	d := newDeviceWithFakeTuner(tr, tn.fakeTuner)
	d.tuner = tn

	ctx, cancel := context.WithCancel(context.Background())
	if _, err := d.StreamIQ(ctx); err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	cancel()
	if err := d.SetCenterFreq(154_130_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if tn.tunedDuringTeardown.Load() {
		t.Error("retune ran while the cancelled stream's StopBulkIn was still in progress")
	}
}
