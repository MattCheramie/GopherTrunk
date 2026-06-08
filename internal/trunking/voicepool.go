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

// FrequencyChecker is implemented by Tuners that can serve only a
// limited range of centre frequencies — e.g. a virtual voice tuner
// backed by a wideband DDC tap can only follow grants inside the
// wideband dongle's IQ window. FindFreeForFrequency consults this
// interface to skip incapable tuners; physical SDRs that don't
// implement it are treated as universally tunable.
type FrequencyChecker interface {
	CanTune(hz uint32) bool
}

// FindFreeForFrequency returns the first free device whose Tuner
// either doesn't implement FrequencyChecker (physical SDR — accepted
// unconditionally) or reports CanTune(hz)=true (virtual tuner whose
// wideband window covers the target). Order matches the device list,
// so the daemon's preference (physical voice SDRs first, virtual
// taps after) is preserved. Returns nil when every free device
// rejects the target — the engine then falls back to preemption.
func (p *VoicePool) FindFreeForFrequency(hz uint32) *VoiceDevice {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, d := range p.devices {
		if _, busy := p.active[d.Serial]; busy {
			continue
		}
		if fc, ok := d.Tuner.(FrequencyChecker); ok && !fc.CanTune(hz) {
			continue
		}
		return d
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

// HasCapableDevice reports whether any device in the pool — busy or
// free — can tune hz. A device with no FrequencyChecker (physical SDR)
// counts as universally capable. The engine uses this to tell a
// coverage gap (e.g. every voice device is a wideband tap whose IQ
// window excludes hz) apart from a genuine all-busy or empty-pool
// condition, so an out-of-window grant isn't mislogged as an engine bug.
func (p *VoicePool) HasCapableDevice(hz uint32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, d := range p.devices {
		if fc, ok := d.Tuner.(FrequencyChecker); ok && !fc.CanTune(hz) {
			continue
		}
		return true
	}
	return false
}

// LowestPriorityActiveForFrequency returns the lowest-priority active
// call among devices that can tune hz, or nil when no such device has
// an active call. It mirrors LowestPriorityActive but skips devices
// whose Tuner rejects hz — preempting one of those would end a call to
// free a device that then can't bind the incoming grant. Physical SDRs
// (no FrequencyChecker) are always eligible, so for a pool with no
// frequency-constrained tuners this matches LowestPriorityActive.
func (p *VoicePool) LowestPriorityActiveForFrequency(hz uint32) *ActiveCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	var lowest *ActiveCall
	for _, ac := range p.active {
		if fc, ok := ac.Device.Tuner.(FrequencyChecker); ok && !fc.CanTune(hz) {
			continue
		}
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

// Retune follows an already-bound call to a new frequency without ending it:
// it retunes the device to g.FrequencyHz and replaces the stored Grant and
// LastHeardAt, preserving StartedAt so the call's duration and identity stay
// continuous. The engine calls this when a control channel moves an
// in-progress call to a new channel (a real handoff, or a P25 band-plan
// IdentifierUpdate re-mapping the channel) — the call is matched by
// (System, GroupID, Timeslot), so following it here is what stops a second
// tuner being bound for the same talkgroup.
//
// The SetReacquire retry applies exactly as in Bind. Returns an error if the
// device isn't currently bound or the tune fails after the retry; on error the
// caller should end the call and rebind a capable device. Note the returned
// ActiveCall pointer is the same instance Bind handed out, so the engine's
// mirror sees the updated Grant without any extra bookkeeping.
func (p *VoicePool) Retune(serial string, g Grant, now time.Time) error {
	p.mu.Lock()
	ac, ok := p.active[serial]
	if !ok {
		p.mu.Unlock()
		return errors.New("trunking: device not bound")
	}
	d := ac.Device
	reacquire := p.reacquire
	p.mu.Unlock()
	if err := d.Tuner.SetCenterFreq(g.FrequencyHz); err != nil {
		if reacquire == nil {
			return err
		}
		newTuner, rerr := reacquire(serial)
		if rerr != nil {
			return errors.Join(err, rerr)
		}
		d.Tuner = newTuner
		if err2 := d.Tuner.SetCenterFreq(g.FrequencyHz); err2 != nil {
			return err2
		}
	}
	p.mu.Lock()
	ac.Grant = g
	ac.LastHeardAt = now
	p.mu.Unlock()
	return nil
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

// UpdateEncryption backfills ALGID/KID on the active call bound to
// serial — used by the engine when an in-call Encryption Sync arrives
// after the original grant (P25 Phase 1 LDU2). Returns a copy of the
// updated Grant for the caller to publish in an enriched event, plus
// ok=true when a matching call was found. The mutation runs under the
// pool's mutex so it stays consistent with concurrent Touch / Release.
func (p *VoicePool) UpdateEncryption(serial string, algID uint8, keyID uint16) (Grant, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ac, ok := p.active[serial]
	if !ok {
		return Grant{}, false
	}
	ac.Grant.AlgorithmID = algID
	ac.Grant.KeyID = keyID
	return ac.Grant, true
}

// UpdateSource backfills SourceID + Encrypted on the active call
// bound to serial — used by the engine when an in-call
// GROUP_VOICE_CHANNEL_USER PDU arrives on the traffic channel after
// a compressed grant whose SOURCE_ID / SVC_OPTIONS were absent
// (e.g. P25 Phase 2 MMR). SourceID is only overwritten when the
// new value is non-zero so a later compressed-form update doesn't
// blank out a legitimate source. Encrypted is OR-merged so an
// in-call PDU can flip a non-encrypted grant to encrypted but
// never the other way (the spec doesn't define mid-call
// decryption). Returns a copy of the updated Grant + ok=true when
// a matching call was found.
func (p *VoicePool) UpdateSource(serial string, sourceID uint32, encrypted bool) (Grant, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ac, ok := p.active[serial]
	if !ok {
		return Grant{}, false
	}
	if sourceID != 0 {
		ac.Grant.SourceID = sourceID
	}
	if encrypted {
		ac.Grant.Encrypted = true
	}
	return ac.Grant, true
}
