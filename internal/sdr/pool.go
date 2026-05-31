package sdr

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// PoolEntry tracks a single discovered-and-opened device along with its role.
//
// Hint carries the per-device tuning the pool applied at Open time so a
// later Snapshot can render gain/PPM/bias-tee state without having to
// query the underlying chip.
type PoolEntry struct {
	Driver Driver
	Device Device
	Info   Info
	Role   Role
	Hint   Hint
}

// Snapshot returns the wire-format status payload for this entry. Used
// by the API's GET /api/v1/devices handler and the bus payload on the
// sdr.attached / sdr.detached events.
//
// attached == true is the normal "device is in the pool" case; the
// detached snapshot published by Pool.Close passes false.
func (e *PoolEntry) Snapshot(attached bool) SDRStatus {
	st := SDRStatus{
		Driver:       e.Info.Driver,
		Serial:       e.Info.Serial,
		Manufacturer: e.Info.Manufacturer,
		Product:      e.Info.Product,
		TunerName:    e.Info.TunerName,
		Role:         e.Role.String(),
		Attached:     attached,
		PPM:          e.Hint.PPM,
		BiasTee:      e.Hint.BiasTee,
		Gains:        append([]int(nil), e.Info.Gains...),
	}
	if e.Hint.gainSet {
		st.GainTenthDB = e.Hint.Gain
		st.GainAuto = e.Hint.Gain < 0
	} else {
		st.GainAuto = true
	}
	return st
}

// Pool holds a fleet of opened SDR devices and assigns roles.
type Pool struct {
	mu      sync.RWMutex
	entries []*PoolEntry
	log     *slog.Logger
	bus     *events.Bus
}

// NewPool constructs an empty pool. The optional bus is used to publish
// events.KindSDRAttached / events.KindSDRDetached as devices come and
// go; pass nil to disable that side effect (tests and the
// `gophertrunk sdr list` CLI both run without a bus).
func NewPool(logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pool{log: logger}
}

// SetBus attaches an events bus so the pool can publish attach/detach
// events. Idempotent; passing nil silently disables publishing.
func (p *Pool) SetBus(bus *events.Bus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bus = bus
}

// Hint guides role assignment when opening devices. Match by serial first;
// fall back to first-found.
//
// PPM, Gain, and BiasTee carry per-device tuning that Pool.Open
// applies once the device is opened. Gain follows the Device.SetGain
// convention: a negative value selects automatic gain control. PPM
// is in parts-per-million; 0 is fine for the TCXO-equipped NESDR
// Smart v5 and similar dongles.
type Hint struct {
	Serial  string
	Role    Role
	PPM     int
	Gain    int // tenths of dB; negative = auto
	BiasTee bool
	// gainSet distinguishes "Gain not configured" (apply auto) from
	// the explicit "auto" choice. The daemon sets this when it parses
	// the YAML; tests that don't care can leave Hint zero-valued and
	// pool.Open won't touch the device's gain.
	gainSet bool
}

// WithGain returns a copy of h with Gain set and the gain-set flag
// flipped so Pool.Open knows to apply it.
func (h Hint) WithGain(tenthDB int) Hint {
	h.Gain = tenthDB
	h.gainSet = true
	return h
}

// DefaultSampleRateHz mirrors librtlsdr's open-time default and is the
// rate Pool.Open programs when the caller passes 0. Matches the value
// the rtlsdr driver also programs during bring-up so the two layers
// agree on a known-good fallback.
const DefaultSampleRateHz uint32 = 2_048_000

// PoolOpenOptions parameterises Pool.OpenWith. Use this when callers
// need to engage strict mode; the historical Pool.Open(rate, hints)
// signature still works and remains the default for code paths that
// want today's open-everything behaviour.
type PoolOpenOptions struct {
	// SampleRateHz is the IQ rate to program on every opened device.
	// Zero falls back to DefaultSampleRateHz.
	SampleRateHz uint32
	// Hints carries the per-device tuning the pool applies once each
	// device is opened (PPM, gain, bias-tee, role). Hints are matched
	// to discovered devices by serial.
	Hints []Hint
	// Strict treats Hints as an allowlist: a discovered device whose
	// serial is not present in Hints is logged and skipped instead of
	// being auto-roled. The daemon engages strict mode when the user
	// has populated cfg.SDR.Devices — that's the operator's signal
	// that they want only the devices they named, not whatever else
	// happens to be on the USB bus.
	Strict bool
}

// Open is a backwards-compatible shim over OpenWith. It preserves the
// historical "open every enumerated device" behaviour; callers that
// want allowlist semantics should construct PoolOpenOptions and call
// OpenWith directly.
func (p *Pool) Open(sampleRateHz uint32, hints []Hint) error {
	return p.OpenWith(PoolOpenOptions{SampleRateHz: sampleRateHz, Hints: hints})
}

// OpenWith enumerates every registered driver, opens the devices the
// options select, programs the IQ sample rate on each one (issue #275 —
// without this the chip streams at whatever rate its resampler powered
// up at), and assigns roles. The first opened device gets RoleControl
// unless a hint says otherwise; subsequent devices get RoleVoice.
//
// When opts.Strict is false, every discovered device is opened. A
// non-hinted device gets an auto-assigned role and runs with the
// driver's default PPM / gain.
//
// When opts.Strict is true, only devices whose serial matches a hint
// are opened. Discovered devices without a matching hint are logged at
// INFO and skipped. Hints whose serial doesn't match any discovered
// device produce a WARN. Hints with empty serial are dropped at ingest
// with a WARN — an empty-serial hint in strict mode is ambiguous (no
// way to honour an allowlist entry that doesn't name anything).
//
// Strict mode is how operators get "only the devices I listed in
// config.yaml are touched"; rtl_tcp and baseband replay always
// originate from explicit config entries, so strict mode applies to
// them uniformly. An rtl_tcp endpoint without a serial: in config is
// therefore skipped in strict mode — set serial: on the endpoint to
// keep it.
//
// A device whose SetSampleRate fails is closed and skipped — a
// wrong-rate radio produces silent decoder failures, which is worse
// than no radio at all.
func (p *Pool) OpenWith(opts PoolOpenOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.entries) > 0 {
		return errors.New("pool already populated; close first")
	}

	rate := opts.SampleRateHz
	if rate == 0 {
		rate = DefaultSampleRateHz
	}

	type discovered struct {
		drv  Driver
		info Info
	}
	var all []discovered
	for _, drv := range Drivers() {
		infos, err := drv.Enumerate()
		if err != nil {
			p.log.Warn("driver enumerate failed", "driver", drv.Name(), "err", err)
			continue
		}
		for _, info := range infos {
			all = append(all, discovered{drv, info})
		}
	}
	if len(all) == 0 {
		return errors.New("no SDR devices discovered")
	}

	hintBySerial := map[string]Hint{}
	controlClaimed := false
	for _, h := range opts.Hints {
		if h.Serial == "" {
			if opts.Strict {
				p.log.Warn("ignoring hint with empty serial in strict mode; set serial: in config to use this entry",
					"role", h.Role.String())
			}
			continue
		}
		hintBySerial[h.Serial] = h
		if h.Role == RoleControl {
			controlClaimed = true
		}
	}

	openedSerials := map[string]struct{}{}
	for _, d := range all {
		hint, hinted := hintBySerial[d.info.Serial]
		if opts.Strict && !hinted {
			p.log.Info("skipping non-configured SDR; add its serial to sdr.devices to use it",
				"driver", d.drv.Name(), "serial", d.info.Serial)
			continue
		}
		role := RoleAuto
		if hinted {
			role = hint.Role
		}
		if role == RoleAuto {
			if !controlClaimed {
				role = RoleControl
				controlClaimed = true
			} else {
				role = RoleVoice
			}
		}
		dev, err := d.drv.Open(d.info.Index)
		if err != nil {
			p.log.Error("open device failed",
				"driver", d.drv.Name(),
				"index", d.info.Index,
				"serial", d.info.Serial,
				"err", err)
			continue
		}
		if err := dev.SetSampleRate(rate); err != nil {
			p.log.Error("set sample rate failed",
				"driver", d.drv.Name(), "serial", d.info.Serial, "rate_hz", rate, "err", err)
			_ = dev.Close()
			continue
		}
		// Apply per-device tuning supplied via Hint. Failures are
		// non-fatal — the device is still usable, just possibly
		// with the driver's defaults — but they get logged so an
		// operator who put bias_tee: true in config sees that the
		// device rejected it.
		if hinted {
			p.applyHintSettings(dev, d.info, hint)
		}
		entry := &PoolEntry{Driver: d.drv, Device: dev, Info: d.info, Role: role, Hint: hint}
		p.entries = append(p.entries, entry)
		openedSerials[d.info.Serial] = struct{}{}
		// Include the per-device tuning in the open log so an
		// operator can grep the boot log to confirm the value they
		// put in config.yaml actually landed on this serial (issue
		// #264 surfaced as "ppm of -4 is not adopted" because the
		// device that received the SetPPM call wasn't the one
		// driving cchunt).
		p.log.Info("device opened",
			"driver", d.drv.Name(),
			"serial", d.info.Serial,
			"role", role.String(),
			"rate_hz", rate,
			"ppm", hint.PPM,
			"bias_tee", hint.BiasTee)
		p.publish(events.KindSDRAttached, entry.Snapshot(true))
	}
	for serial := range hintBySerial {
		if _, ok := openedSerials[serial]; !ok {
			p.log.Warn("configured SDR not present on the bus; check the cable / dmesg / lsusb",
				"serial", serial)
		}
	}
	if len(p.entries) == 0 {
		return errors.New("no SDR devices opened")
	}
	return nil
}

func (p *Pool) Entries() []*PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*PoolEntry, len(p.entries))
	copy(out, p.entries)
	return out
}

// Snapshot returns a status payload for every entry currently in the
// pool. Safe to call concurrently with Open / Close.
func (p *Pool) Snapshot() []SDRStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]SDRStatus, 0, len(p.entries))
	for _, e := range p.entries {
		out = append(out, e.Snapshot(true))
	}
	return out
}

// FirstByRole returns the first device with the given role, or nil.
func (p *Pool) FirstByRole(r Role) *PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, e := range p.entries {
		if e.Role == r {
			return e
		}
	}
	return nil
}

// AllByRole returns every device with the given role.
func (p *Pool) AllByRole(r Role) []*PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []*PoolEntry
	for _, e := range p.entries {
		if e.Role == r {
			out = append(out, e)
		}
	}
	return out
}

// FindBySerial returns the entry whose info.Serial matches, or nil.
// Used by the demod-pipeline composer to look up a Voice device that
// the engine has just bound to a call.
func (p *Pool) FindBySerial(serial string) *PoolEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, e := range p.entries {
		if e.Info.Serial == serial {
			return e
		}
	}
	return nil
}

// applyHintSettings runs the per-device tuners after Open. Caller
// holds p.mu.
func (p *Pool) applyHintSettings(dev Device, info Info, h Hint) {
	if h.PPM != 0 {
		if err := dev.SetPPM(h.PPM); err != nil {
			p.log.Warn("set ppm failed", "serial", info.Serial, "ppm", h.PPM, "err", err)
		}
	}
	if h.gainSet {
		if err := dev.SetGain(h.Gain); err != nil {
			p.log.Warn("set gain failed", "serial", info.Serial, "gain", h.Gain, "err", err)
		} else if h.Gain < 0 {
			p.log.Info("sdr: gain set to automatic (AGC)", "serial", info.Serial, "role", h.Role.String())
		} else {
			// Surface the *applied* gain in dB at INFO. The driver snaps
			// h.Gain to the nearest rung on the tuner's ladder, so a
			// units mistake (gain: "32" → 3.2 dB, see parseGain) ends up
			// near the bottom of the ladder and silently deaf. Logging
			// the dB figure on the normal startup path lets the operator
			// catch it without enabling debug.
			p.log.Info("sdr: gain set",
				"serial", info.Serial,
				"role", h.Role.String(),
				"gain_db", float64(h.Gain)/10.0)
		}
	} else {
		// Surface the "no `gain:` in config" path explicitly. Without
		// this warn, a device whose driver default happens to be too
		// low for the user's antenna + LNA chain reads as completely
		// deaf — the field symptom in issue #356 (v0.2.4 follow-up,
		// reporter @v2maldo). The driver-default gain varies between
		// chips and even firmware revisions; surfacing it once at open
		// gives the operator a chance to set `gain: auto` (AGC) or a
		// specific tenth-dB value before chasing harder hypotheses
		// like a broken voice chain.
		p.log.Warn("sdr: no gain configured for device; using driver default — set `gain: auto` for AGC or a specific tenth-dB value (e.g. \"496\" = 49.6 dB) if reception is weak",
			"serial", info.Serial, "role", h.Role.String())
	}
	if h.BiasTee {
		if err := dev.SetBiasTee(true); err != nil {
			p.log.Warn("set bias_tee failed", "serial", info.Serial, "err", err)
		}
	}
}

// publish is a non-blocking helper that fans an event to the optional
// bus. Caller holds p.mu (read or write — Bus.Publish is internally
// safe, and the snapshot is constructed from already-copied data).
func (p *Pool) publish(kind events.Kind, payload any) {
	if p.bus == nil {
		return
	}
	p.bus.Publish(events.Event{Kind: kind, Payload: payload})
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var errs []error
	for _, e := range p.entries {
		p.publish(events.KindSDRDetached, e.Snapshot(false))
		if err := e.Device.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Info.Serial, err))
		}
	}
	p.entries = nil
	return errors.Join(errs...)
}

// Reacquire releases the existing device handle for the given serial
// and tries to re-open the same serial against the entry's original
// driver. On success the PoolEntry's Device is swapped in place —
// Role, Hint, and serial identity are preserved, Info.Index updates to
// reflect the new enumeration — and KindSDRDetached + KindSDRAttached
// events are published so consumers (and the API/web snapshot) observe
// the swap. The configured sample rate plus the original Hint
// (PPM / gain / bias-tee) are re-applied to the fresh handle.
//
// Designed for recovery from transient USB disconnect/re-enumerate
// cycles: the kernel assigns a new device number but the dongle
// reports the same serial. The caller (typically the daemon's
// ccdecoder retry loop) drives the backoff between attempts. Closing
// the existing handle is best-effort — a dead handle's Close may
// return errors which are logged but not surfaced. See issue #345.
//
// Returns the refreshed PoolEntry on success, or an error if the
// serial is unknown to the pool, the driver re-enumerate misses the
// serial, or open / sample-rate programming fails.
func (p *Pool) Reacquire(serial string, sampleRateHz uint32) (*PoolEntry, error) {
	if serial == "" {
		return nil, errors.New("sdr: Reacquire requires a non-empty serial")
	}
	rate := sampleRateHz
	if rate == 0 {
		rate = DefaultSampleRateHz
	}

	// Snapshot entry identity under the lock, then drop it before the
	// slow driver enumerate / open calls (USB I/O, potentially
	// hundreds of ms). The entry itself is preserved — only its Device
	// handle and Info.Index change — so concurrent readers that hold a
	// pointer keep working; the actual handle swap is done under the
	// lock at the end.
	p.mu.RLock()
	var (
		entry *PoolEntry
	)
	for _, e := range p.entries {
		if e.Info.Serial == serial {
			entry = e
			break
		}
	}
	if entry == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("sdr: Reacquire: serial %q not in pool", serial)
	}
	drv := entry.Driver
	oldInfo := entry.Info
	hint := entry.Hint
	oldDev := entry.Device
	p.mu.RUnlock()

	// Best-effort close of the (likely dead) handle. The purego
	// Device.Close is idempotent and safe to call against a
	// transport whose USB endpoint has already disappeared.
	if oldDev != nil {
		if err := oldDev.Close(); err != nil {
			p.log.Debug("sdr: Reacquire: close of stale handle returned error",
				"serial", serial, "err", err)
		}
	}
	// Tell the bus the device went away so the API snapshot and any
	// UI can show the gap. The Attached event below republishes the
	// fresh state.
	p.publish(events.KindSDRDetached, entry.Snapshot(false))

	infos, err := drv.Enumerate()
	if err != nil {
		return nil, fmt.Errorf("sdr: Reacquire: %s enumerate: %w", drv.Name(), err)
	}
	var freshInfo Info
	found := false
	for _, info := range infos {
		if info.Serial == serial {
			freshInfo = info
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("sdr: Reacquire: serial %q not present after %s re-enumerate", serial, drv.Name())
	}

	dev, err := drv.Open(freshInfo.Index)
	if err != nil {
		return nil, fmt.Errorf("sdr: Reacquire: open serial %q: %w", serial, err)
	}
	if err := dev.SetSampleRate(rate); err != nil {
		_ = dev.Close()
		return nil, fmt.Errorf("sdr: Reacquire: set sample rate on serial %q: %w", serial, err)
	}
	// Re-apply per-device tuning (PPM, gain, bias-tee). applyHintSettings
	// only logs failures — it is non-fatal in the original Open path and
	// stays non-fatal here for the same reason.
	p.applyHintSettings(dev, freshInfo, hint)

	// Carry forward identity-stable Info fields (Serial, Driver,
	// Manufacturer/Product/TunerName, Gains list) but accept the
	// possibly-changed Index from the fresh enumerate.
	mergedInfo := oldInfo
	mergedInfo.Index = freshInfo.Index
	mergedInfo.Gains = freshInfo.Gains
	mergedInfo.TunerName = freshInfo.TunerName

	p.mu.Lock()
	entry.Device = dev
	entry.Info = mergedInfo
	p.mu.Unlock()

	p.publish(events.KindSDRAttached, entry.Snapshot(true))
	p.log.Info("sdr: reacquired",
		"driver", drv.Name(), "serial", serial, "role", entry.Role.String(),
		"old_index", oldInfo.Index, "new_index", freshInfo.Index)
	return entry, nil
}
