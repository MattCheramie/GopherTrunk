package rtl2832u

import (
	"errors"
	"syscall"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// stallTransport is a minimal usb.Transport whose ControlOut stalls its
// first stallOuts calls (with stallErr) and then succeeds. Everything else
// is a benign no-op; ControlIn always succeeds (SetI2CRepeater's commit read).
type stallTransport struct {
	stallOuts int   // number of leading ControlOut calls that return stallErr
	stallErr  error // the stall error to return
	outCalls  int
	inCalls   int
}

func (s *stallTransport) ControlOut(_ uint8, _, _ uint16, _ []byte, _ int) error {
	s.outCalls++
	if s.outCalls <= s.stallOuts {
		return s.stallErr
	}
	return nil
}

func (s *stallTransport) ControlIn(_ uint8, _, _ uint16, n, _ int) ([]byte, error) {
	s.inCalls++
	return make([]byte, n), nil
}

func (s *stallTransport) ClaimInterface(int) error                                    { return nil }
func (s *stallTransport) ReleaseInterface(int) error                                  { return nil }
func (s *stallTransport) StartBulkIn(byte, int, int, func([]byte), func(error)) error { return nil }
func (s *stallTransport) StopBulkIn() error                                           { return nil }
func (s *stallTransport) Reset() error                                                { return nil }
func (s *stallTransport) Close() error                                                { return nil }

var _ usb.Transport = (*stallTransport)(nil)

// TestSetI2CRepeater_RecoversFromControlPipeStall is the #753 regression: an
// intermittent control-pipe STALL on the SetI2CRepeater demod write (the
// RTL-SDR Blog V4's R828D mid-retune) must be settled and retried once when
// runtime recovery is armed, instead of aborting the tune. Covers both the
// Linux (syscall.EPIPE) and Windows/macOS (usb.ErrPipeStalled) stall forms.
//
// Fails first without the retry: the single stalled ControlOut surfaces
// straight out of SetI2CRepeater.
func TestSetI2CRepeater_RecoversFromControlPipeStall(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"linux-EPIPE", syscall.EPIPE},
		{"pipe-stalled", usb.ErrPipeStalled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := &stallTransport{stallOuts: 1, stallErr: tc.err}
			d := New(tr)
			d.EnableControlStallRetry() // armed post-bring-up

			if err := d.SetI2CRepeater(true); err != nil {
				t.Fatalf("SetI2CRepeater(true) = %v, want nil (should recover from a single control-pipe stall)", err)
			}
			if tr.outCalls != 2 {
				t.Errorf("ControlOut called %d times, want 2 (one stall + one retry)", tr.outCalls)
			}
		})
	}
}

// TestSetI2CRepeater_StallNotRetriedDuringBringup guards that recovery stays
// OFF until EnableControlStallRetry is called: during bring-up a control-pipe
// stall is the cold-boot latch the open-time reset envelope must see, so it
// must surface, not be silently retried.
func TestSetI2CRepeater_StallNotRetriedDuringBringup(t *testing.T) {
	tr := &stallTransport{stallOuts: 1, stallErr: syscall.EPIPE}
	d := New(tr) // NOT armed

	if err := d.SetI2CRepeater(true); !errors.Is(err, syscall.EPIPE) {
		t.Fatalf("SetI2CRepeater(true) = %v, want an EPIPE stall surfaced (no retry before bring-up completes)", err)
	}
	if tr.outCalls != 1 {
		t.Errorf("ControlOut called %d times, want 1 (unarmed: no retry)", tr.outCalls)
	}
}

// TestSetI2CRepeater_PersistentStallSurfaces guards the bound: a stall that
// survives the single retry still surfaces (no silent looping on a wedged pipe).
func TestSetI2CRepeater_PersistentStallSurfaces(t *testing.T) {
	tr := &stallTransport{stallOuts: 2, stallErr: usb.ErrPipeStalled} // both attempts stall
	d := New(tr)
	d.EnableControlStallRetry()

	if err := d.SetI2CRepeater(true); !errors.Is(err, usb.ErrPipeStalled) {
		t.Fatalf("SetI2CRepeater(true) = %v, want the persistent stall surfaced", err)
	}
	if tr.outCalls != 2 {
		t.Errorf("ControlOut called %d times, want 2 (one retry, then give up)", tr.outCalls)
	}
}
