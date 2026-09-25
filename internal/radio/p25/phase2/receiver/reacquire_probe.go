//go:build integration

package receiver

// Granular re-acquisition hooks, for attributing a lost channel to one piece
// of receiver state. Integration-only: production callers use Reset,
// ReacquireCarrier or ReacquireTiming.

// ReacquirePart resets exactly one component. Unknown names are a no-op.
func (r *Receiver) ReacquirePart(name string) {
	switch name {
	case "gardner":
		if r.gardner != nil {
			r.gardner.Reset()
		}
	case "agc":
		if r.agc != nil {
			r.agc.Reset()
		}
	case "dq":
		r.dq.Reset()
	case "pending":
		r.pending = r.pending[:0]
		r.rxOffset = 0
	case "eq":
		if r.eq != nil {
			r.eq.Reset()
		}
	case "costas":
		if r.costas != nil {
			r.costas.Reset()
		}
	case "nco":
		if r.nco != nil {
			r.nco.Reset()
			r.nco.SetOffset(0, r.fs)
		}
		r.seeded = false
		r.seedHz = 0
		r.seedBuf = nil
	}
}

// AGCGain reports the AGC's current gain, 0 when there is no AGC.
func (r *Receiver) AGCGain() float64 {
	if r.agc == nil {
		return 0
	}
	return float64(r.agc.Gain())
}

// TimingMu reports the Gardner loop's sub-sample phase, 0 when there is no
// Gardner loop. Diagnostic for the re-acquisition work.
func (r *Receiver) TimingMu() float64 {
	if r.gardner == nil {
		return 0
	}
	return r.gardner.Mu()
}
