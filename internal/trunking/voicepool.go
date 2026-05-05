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
}

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
// Returns an error if the device is already busy or the tune fails.
func (p *VoicePool) Bind(d *VoiceDevice, g Grant, tg *TalkGroup, now time.Time) (*ActiveCall, error) {
	if d == nil {
		return nil, errors.New("trunking: nil device")
	}
	p.mu.Lock()
	if _, busy := p.active[d.Serial]; busy {
		p.mu.Unlock()
		return nil, errors.New("trunking: device already busy")
	}
	p.mu.Unlock()
	if err := d.Tuner.SetCenterFreq(g.FrequencyHz); err != nil {
		return nil, err
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
