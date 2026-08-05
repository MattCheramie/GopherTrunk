package trunking

import (
	"errors"
	"sync"
	"sync/atomic"
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
	// callSeq is a process-monotonic counter that assigns each bound call a
	// unique Grant.CallID, used by the voice chain + recorder to fence
	// cross-call audio bleed when a tap serial is reused. Starts at 0; the
	// first Bind hands out CallID 1, so a real CallID is always non-zero and
	// distinguishable from an un-bound (synthetic) grant's zero.
	callSeq atomic.Uint64
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
	// EncReleaseAt is the deadline at which the encrypted-call-handling
	// policy (trunking.encrypted_calls mode: metadata) releases this
	// voice SDR. Zero means unarmed — either the call is clear, the mode
	// is follow/ignore, or the operator holds a decryption key for it.
	// Armed by ArmEncryptedRelease, cleared by DisarmEncryptedRelease,
	// and reaped by the engine watchdog via EncryptedReleasesDue. Issue
	// #711.
	EncReleaseAt time.Time
	// SignalDbFS is the call's mean received channel power in dBFS,
	// measured by the voice composer over the call's baseband IQ and
	// stamped in via UpdateSignal shortly before end-of-call. nil until
	// measured — calls ended by non-composer paths (watchdog timeout,
	// preemption, shutdown) leave it nil. See composer.boundaryTracker
	// for the semantics (channel power, not calibrated RSSI or SNR).
	SignalDbFS *float64
	// EVMPct / SNRDb are the call's demod quality (RMS EVM % and estimated
	// SNR dB), stamped in via UpdateDemod shortly before end-of-call. nil
	// until measured; only chains that feed the receiver demod taps (P25
	// Phase 1) populate them. Issue #878 follow-up.
	EVMPct *float64
	SNRDb  *float64
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
	// Stamp a fresh, process-unique CallID so the voice chain can tag this
	// call's decoded frames and the recorder can reject a stale frame from
	// the call that previously held this serial.
	g.CallID = p.callSeq.Add(1)
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
	// A handoff continues the same call, so preserve its CallID across the
	// grant replacement (the new grant from the control channel carries no
	// CallID of its own).
	g.CallID = ac.Grant.CallID
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

// UpdateSignal stamps the call's measured received channel power (dBFS)
// onto the ActiveCall bound to serial. No-op when no call is bound. A
// fresh pointer is stored per update so the value is stable once released.
func (p *VoicePool) UpdateSignal(serial string, dbfs float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ac, ok := p.active[serial]; ok {
		v := dbfs
		ac.SignalDbFS = &v
	}
}

// UpdateDemod stamps the call's demod quality (RMS EVM % and estimated SNR dB)
// onto the ActiveCall bound to serial. No-op when no call is bound. Fresh
// pointers are stored per update so the values are stable once released.
func (p *VoicePool) UpdateDemod(serial string, evmPct, snrDB float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ac, ok := p.active[serial]; ok {
		evm, snr := evmPct, snrDB
		ac.EVMPct = &evm
		ac.SNRDb = &snr
	}
}

// ArmEncryptedRelease schedules the metadata-mode teardown of the call
// bound to serial at time at. It only arms when the call is currently
// unarmed, so a repeat encryption update keeps the earliest deadline
// (the metadata window starts when encryption is first known, not on
// every in-call re-confirmation). No-op when no call is bound. Issue
// #711.
func (p *VoicePool) ArmEncryptedRelease(serial string, at time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ac, ok := p.active[serial]; ok && ac.EncReleaseAt.IsZero() {
		ac.EncReleaseAt = at
	}
}

// DisarmEncryptedRelease cancels any pending metadata-mode teardown on
// the call bound to serial — used when a later in-call update reveals the
// operator holds a decryption key for the call, so it should be followed
// after all. No-op when no call is bound. Issue #711.
func (p *VoicePool) DisarmEncryptedRelease(serial string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ac, ok := p.active[serial]; ok {
		ac.EncReleaseAt = time.Time{}
	}
}

// EncryptedReleasesDue returns every active call whose armed
// encrypted-release deadline has been reached by now. The engine
// watchdog ends each with EndReasonEncrypted. Evaluated under the pool
// lock so the deadline read stays consistent with concurrent
// ArmEncryptedRelease / Touch from the decode pipeline. Issue #711.
func (p *VoicePool) EncryptedReleasesDue(now time.Time) []*ActiveCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	var due []*ActiveCall
	for _, ac := range p.active {
		if !ac.EncReleaseAt.IsZero() && !now.Before(ac.EncReleaseAt) {
			due = append(due, ac)
		}
	}
	return due
}

// ArmedCallBySource returns the active call from system originated by
// sourceID that is currently armed for a metadata-mode encrypted release
// (EncReleaseAt set), or nil. The engine uses it to short-circuit the
// metadata-follow window the moment a system's talker alias completes:
// the metadata we held the tuner for has arrived, so the call can be torn
// down early. Only armed calls match, so a completed alias never disturbs
// a follow-mode or clear call. Evaluated under the pool lock so the read
// stays consistent with concurrent ArmEncryptedRelease / Touch. Issue
// #711.
func (p *VoicePool) ArmedCallBySource(system string, sourceID uint32) *ActiveCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ac := range p.active {
		if ac.EncReleaseAt.IsZero() {
			continue
		}
		if ac.Grant.System == system && ac.Grant.SourceID == sourceID {
			return ac
		}
	}
	return nil
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
func (p *VoicePool) UpdateSource(serial string, sourceID uint32, encrypted, emergency bool, priority uint8) (Grant, bool) {
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
	if emergency {
		ac.Grant.Emergency = true
	}
	if priority != 0 {
		ac.Grant.Priority = priority
	}
	return ac.Grant, true
}

// BackfillSourceFromGrant fills the bound call's source RID from a control-
// channel grant, but only when the call has no source yet: the initiating
// GRP_VCH_GRANT's RID is a fallback, so an in-call GROUP_VOICE_CHANNEL_USER
// update (UpdateSource) — which reflects the radio actually keyed on the
// traffic channel — must be able to set the source over this, never the
// reverse, and a later grant must never clobber an already-known RID. Returns the
// updated Grant with filled=true only when a previously-zero source was set,
// so the engine republishes the source-update event once per call instead of
// on every repeat grant. No-op (filled=false) when no call is bound, sourceID
// is zero, or the source is already known. Issue #915.
func (p *VoicePool) BackfillSourceFromGrant(serial string, sourceID uint32) (Grant, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ac, ok := p.active[serial]
	if !ok || sourceID == 0 || ac.Grant.SourceID != 0 {
		return Grant{}, false
	}
	ac.Grant.SourceID = sourceID
	return ac.Grant, true
}

// BackfillSourceForChannel folds a control-channel grant's source RID onto the
// active call occupying a physical channel — identified by (system, frequency,
// timeslot) — regardless of the grant's talkgroup label. It is the wider net
// BackfillSourceFromGrant (keyed on the call's talkgroup) can't cast: on a
// heavily-compressed Phase 2 system the RID-bearing GRP_VCH_GRANT for a call
// frequently arrives under a *different* talkgroup than the source-less grant
// that bound the call (a mis-aliased compressed grant, or a super-group / patch
// remap), so it never matches the talkgroup dedup — the residual ~88% of the
// #915 coverage gap the reporter measured. A frequency + timeslot hosts exactly
// one in-progress transmission (see Grant.Timeslot, which documents
// (FrequencyHz, Timeslot) as the engine's call identity), so a source-carrying
// grant on an active call's exact channel belongs to that call.
//
// Fill-only-when-zero, like BackfillSourceFromGrant: an in-call
// GROUP_VOICE_CHANNEL_USER (UpdateSource) still wins, and a later grant never
// clobbers a known RID. Returns the bound device serial, the updated Grant, and
// filled=true only on the first fill of a previously-zero source — so the engine
// republishes the source-update once per call. Returns ("", zero, false) when
// sourceID or freqHz is zero, or no channel-matching call has an unset source.
// Issue #915.
func (p *VoicePool) BackfillSourceForChannel(system string, freqHz uint32, timeslot uint8, sourceID uint32) (serial string, g Grant, filled bool) {
	if sourceID == 0 || freqHz == 0 {
		return "", Grant{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for s, ac := range p.active {
		if ac.Grant.System != system || ac.Grant.FrequencyHz != freqHz ||
			ac.Grant.Timeslot != timeslot || ac.Grant.SourceID != 0 {
			continue
		}
		ac.Grant.SourceID = sourceID
		return s, ac.Grant, true
	}
	return "", Grant{}, false
}
