package trunking

import (
	"errors"
	"sync"
	"time"
)

// VoiceDevice is one Voice-role SDR available to the engine. The engine
// retunes it via the embedded Tuner interface and tracks an optional
// active call.
type VoiceDevice struct {
	Tuner  Tuner
	Serial string
}

// VoicePool manages the set of Voice-role devices and the call currently
// (if any) bound to each. It is safe for concurrent use.
type VoicePool struct {
	mu      sync.Mutex
	devices []*VoiceDevice
	active  map[string]*ActiveCall // by device serial
	// reacquire, when set, is called by Bind on a SetCenterFreq
	// failure to ask the SDR pool to re-open the device by serial
	// (typically after a transient USB disconnect / re-enumerate).
	// The returned Tuner replaces the VoiceDevice's stale handle and
	// Bind retries SetCenterFreq once. Wired in by the daemon via
	// SetReacquire; nil = no retry, current behaviour. See issue #345.
	reacquire ReacquireFunc
}

// ReacquireFunc asks the SDR pool to re-open the device with the
// given serial and return its fresh Tuner handle. Implementations
// (typically the daemon's bridge to sdr.Pool.Reacquire) close the
// stale handle, re-enumerate the driver, open the matching serial,
// re-apply per-device tuning, and swap the entry in place — see
// sdr.Pool.Reacquire for the contract.
type ReacquireFunc func(serial string) (Tuner, error)

// ActiveCall describes a grant currently being followed on a specific
// Voice device. The engine creates these via VoicePool.Bind.
type ActiveCall struct {
	Device      *VoiceDevice
	Grant       Grant
	Talkgroup   *TalkGroup
	StartedAt   time.Time
	LastHeardAt time.Time
}

// NewVoicePool returns a pool over the supplied devices. The order of
// devices determines allocation preference (first-fit).
func NewVoicePool(devices []*VoiceDevice) *VoicePool {
	return &VoicePool{devices: devices, active: make(map[string]*ActiveCall)}
}

// SetReacquire installs the SDR-pool reacquire callback. After this
// is set, Bind retries SetCenterFreq once via the callback when the
// initial tune fails — recovering from a USB disconnect / re-
// enumerate without dropping the call. Idempotent; passing nil
// disables the retry (matches the legacy behaviour).
func (p *VoicePool) SetReacquire(fn ReacquireFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reacquire = fn
}

// Devices returns a snapshot of the device list.
func (p *VoicePool) Devices() []*VoiceDevice {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]*VoiceDevice, len(p.devices))
	copy(out, p.devices)
	return out
}

// FindFree returns the first device with no active call, or nil if every
// device is busy. The pool lock is held only during the scan.
func (p *VoicePool) FindFree() *VoiceDevice {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, d := range p.devices {
		if _, busy := p.active[d.Serial]; !busy {
			return d
		}
	}
	return nil
}

// LowestPriorityActive returns the active call with the lowest priority
// among all devices, or nil if no calls are active. Used by the engine
// when deciding which call to preempt.
func (p *VoicePool) LowestPriorityActive() *ActiveCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	var lowest *ActiveCall
	for _, ac := range p.active {
		if lowest == nil ||
			EffectivePriority(ac.Grant, ac.Talkgroup) > EffectivePriority(lowest.Grant, lowest.Talkgroup) {
			lowest = ac
		}
	}
	return lowest
}

// Bind retunes the device to grant.FrequencyHz and records an active call.
// Returns an error if the device is already busy or the tune fails. When
// SetReacquire is wired, a SetCenterFreq failure triggers one reacquire
// attempt against the SDR pool — the stale handle is swapped for a fresh
// one and the tune is retried. Recovers from a USB disconnect that
// happened while the device was idle between calls (issue #345).
func (p *VoicePool) Bind(d *VoiceDevice, g Grant, tg *TalkGroup, now time.Time) (*ActiveCall, error) {
	if d == nil {
		return nil, errors.New("trunking: nil device")
	}
	p.mu.Lock()
	if _, busy := p.active[d.Serial]; busy {
		p.mu.Unlock()
		return nil, errors.New("trunking: device already busy")
	}
	reacquire := p.reacquire
	p.mu.Unlock()
	if err := d.Tuner.SetCenterFreq(g.FrequencyHz); err != nil {
		if reacquire == nil {
			return nil, err
		}
		// First tune failed — most often a USB disconnect/re-
		// enumerate that left this VoiceDevice's Tuner handle dead.
		// Ask the SDR pool to re-open the same serial; if that
		// succeeds, swap the live handle in and retry the tune
		// once. Any retry failure surfaces the second error so the
		// caller logs the genuine cause rather than the stale one.
		newTuner, rerr := reacquire(d.Serial)
		if rerr != nil {
			return nil, errors.Join(err, rerr)
		}
		d.Tuner = newTuner
		if err2 := d.Tuner.SetCenterFreq(g.FrequencyHz); err2 != nil {
			return nil, err2
		}
	}
	ac := &ActiveCall{
		Device:      d,
		Grant:       g,
		Talkgroup:   tg,
		StartedAt:   now,
		LastHeardAt: now,
	}
	p.mu.Lock()
	p.active[d.Serial] = ac
	p.mu.Unlock()
	return ac, nil
}

// Release marks the device free. Returns the freed ActiveCall (or nil if
// the device wasn't busy).
func (p *VoicePool) Release(serial string) *ActiveCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	ac, ok := p.active[serial]
	if !ok {
		return nil
	}
	delete(p.active, serial)
	return ac
}

// Active returns a snapshot of every currently-bound call.
func (p *VoicePool) Active() []*ActiveCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]*ActiveCall, 0, len(p.active))
	for _, ac := range p.active {
		out = append(out, ac)
	}
	return out
}

// Touch updates the LastHeardAt timestamp for the given device. The engine
// watchdog uses this to detect calls that have ended without an explicit
// release announcement.
func (p *VoicePool) Touch(serial string, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ac, ok := p.active[serial]; ok {
		ac.LastHeardAt = now
	}
}
