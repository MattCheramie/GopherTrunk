package trunking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// Engine is the central trunking state machine. It subscribes to
// events.KindGrant, looks up the talkgroup, dispatches to the voice pool
// (preempting lower-priority active calls when necessary), and emits
// events.KindCallStart / events.KindCallEnd.
//
// The engine deliberately knows nothing about the demod pipeline — it
// just tunes Voice devices and publishes structured events. Downstream
// consumers (the voice composer + recorder, the SQLite call log)
// subscribe to the CallStart / CallEnd events to do their work.
type Engine struct {
	bus        *events.Bus
	log        *slog.Logger
	pool       *VoicePool
	talkgroups *TalkgroupDB
	patches    *PatchRegistry
	timeout    time.Duration
	now        func() time.Time
	sub        *events.Subscription
	closeOnce  sync.Once
	// noVoiceSDROnce gates the actionable "no voice SDR" warning so
	// it logs once per Engine lifetime instead of once per grant.
	// Subsequent grants on an empty pool drop at DEBUG. Reset when
	// the engine is reconstructed (daemon reload / restart).
	noVoiceSDROnce sync.Once
	// noVoiceCoverageOnce gates the analogous warning for a pool that
	// has voice devices but none whose tuning window covers the grant
	// frequency — e.g. a wideband-only rig whose IQ window excludes the
	// repeater. Logged once, then DEBUG per grant.
	noVoiceCoverageOnce sync.Once

	// scanMode is read under modeMu so the API cockpit can flip it at
	// runtime without a daemon restart. HandleGrant takes a snapshot under
	// the read lock to avoid blocking the bus loop.
	modeMu   sync.RWMutex
	scanMode ScanMode

	// encModes / encFollows map a system name to its encrypted-call policy
	// (trunking.systems[].encrypted_calls). Per-system so an operator can
	// run "metadata" on one system and "follow" / "ignore" on another.
	// Read-only after NewEngine, so no lock; a system absent from the map
	// defaults to EncryptedFollow / defaultEncryptedMetadataFollow. Issue
	// #711.
	encModes   map[string]EncryptedMode
	encFollows map[string]time.Duration

	// configuredKeys maps a system name to the set of encryption key IDs
	// the operator supplied for it (trunking.systems[].encryption_keys).
	// A call whose KeyID is in its system's set is "decryptable" — the
	// operator intends to capture / decode it — so the encrypted-call
	// policy exempts it and always follows it. Read-only after NewEngine,
	// so no lock. nil / empty for systems with no keys. Issue #711.
	configuredKeys map[string]map[uint16]bool

	mu        sync.Mutex
	calls     map[string]*ActiveCall // by device serial; mirror of pool.active for fast access
	synthetic map[string]*ActiveCall // by device serial; calls owned by external scanners (conv FM)
	// observed tracks every voice call the control channel has announced,
	// keyed by observedKey (system|talkgroup|timeslot), regardless of whether
	// a voice tuner is following it. It is the source for ObservedCalls, which
	// lets operator UIs show ALL talkgroups currently up on a system — not just
	// the few a tuner is decoding — so a single-voice-tuner rig no longer looks
	// like only one talkgroup is ever active. Entries refresh on the control
	// channel's repeated grant TSBKs and age out via runWatchdog once the call
	// drops. Guarded by mu, like calls/synthetic.
	observed map[string]*ActiveCall
}

// observedKey identifies a logical call for the observed-call tracker:
// (System, talkgroup, timeslot) — the same identity HandleGrant's duplicate-
// grant guard uses, so a DMR Tier III carrier's two per-slot calls stay
// distinct and a band-plan re-map (same call, new frequency) updates one entry.
func observedKey(g Grant) string {
	return fmt.Sprintf("%s|%d|%d", g.System, g.GroupID, g.Timeslot)
}

// EngineOptions configure a new Engine.
type EngineOptions struct {
	Bus        *events.Bus
	Log        *slog.Logger
	VoicePool  *VoicePool
	Talkgroups *TalkgroupDB
	// CallTimeout is how long a call can run without a Touch before
	// the watchdog reaps it. Default 30 s. The end reason depends on
	// whether the call ever decoded frames: EndReasonNormal when
	// frames arrived and the carrier later dropped (P25's natural
	// end-of-call mechanism, since the CC has no explicit channel
	// release); EndReasonTimeout when no frames ever arrived (silent
	// decode failure).
	CallTimeout time.Duration
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
	// ScanMode controls whether HandleGrant respects the per-talkgroup
	// Scan flag. Default ScanModeAll keeps every non-locked-out grant
	// flowing through; ScanModeList enforces the talkgroup scan list.
	ScanMode ScanMode
	// EncryptedModes maps system name -> how encrypted calls on that
	// system are handled. EncryptedFollow (the default for a system absent
	// from the map) holds a voice SDR for the full call (legacy
	// behaviour); EncryptedMetadata follows briefly then releases;
	// EncryptedIgnore never allocates a voice SDR to an encrypted call.
	// Per-system (trunking.systems[].encrypted_calls). Issue #711.
	EncryptedModes map[string]EncryptedMode
	// EncryptedFollows maps system name -> how long an encrypted call on
	// that system is followed under EncryptedMetadata before its voice SDR
	// is released, measured from when the call is first known to be
	// encrypted. A system absent from the map (or mapped to <= 0) uses
	// defaultEncryptedMetadataFollow (1.5 s). Issue #711.
	EncryptedFollows map[string]time.Duration
	// ConfiguredKeys maps system name -> set of configured encryption key
	// IDs. Calls whose KeyID matches are exempt from the encrypted-call
	// policy (always followed). nil disables the exemption. Issue #711.
	ConfiguredKeys map[string]map[uint16]bool
}

// defaultEncryptedMetadataFollow is the metadata-mode follow window
// applied when trunking.encrypted_calls.metadata_follow_ms is unset /
// zero. Long enough for a P25 Phase 2 talker-alias reassembly + a couple
// of MAC PDU repeats, short enough to free the tuner quickly. Issue #711.
const defaultEncryptedMetadataFollow = 1500 * time.Millisecond

// algorithmClear is the encryption Algorithm ID a clear (unencrypted)
// P25 call advertises; anything else means encrypted. Mirrors
// p25.AlgorithmClear, kept local to avoid a radio-package import (same
// pattern as internal/voice/recorder.go). Issue #711.
const algorithmClear uint8 = 0x80

// NewEngine validates opts and returns a ready-to-Run engine.
func NewEngine(opts EngineOptions) (*Engine, error) {
	if opts.Bus == nil {
		return nil, errors.New("trunking/engine: events.Bus is required")
	}
	if opts.VoicePool == nil {
		return nil, errors.New("trunking/engine: VoicePool is required")
	}
	if opts.Talkgroups == nil {
		opts.Talkgroups = NewTalkgroupDB()
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.CallTimeout <= 0 {
		opts.CallTimeout = 30 * time.Second
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	// Surface the resolved watchdog timeout once at startup so an operator
	// can confirm from logs that the configured trunking.call_timeout_ms
	// is the value the engine is actually using — issue #356 follow-up
	// where a field log showed calls dying well under the configured
	// 5 s, and there was no log line to verify what the engine had
	// applied.
	opts.Log.Info("engine: configured", "call_timeout", opts.CallTimeout)
	e := &Engine{
		bus:            opts.Bus,
		log:            opts.Log,
		pool:           opts.VoicePool,
		talkgroups:     opts.Talkgroups,
		patches:        NewPatchRegistry(),
		timeout:        opts.CallTimeout,
		now:            opts.Now,
		scanMode:       opts.ScanMode,
		encModes:       opts.EncryptedModes,
		encFollows:     opts.EncryptedFollows,
		configuredKeys: opts.ConfiguredKeys,
		calls:          make(map[string]*ActiveCall),
		synthetic:      make(map[string]*ActiveCall),
		observed:       make(map[string]*ActiveCall),
	}
	// Subscribe at construction time so callers can publish grants
	// before Run starts without losing them.
	e.sub = opts.Bus.Subscribe()
	return e, nil
}

// Close releases the engine's subscription. Safe to call concurrently
// with Run; idempotent on repeat calls. Subscription.Close is itself
// idempotent so we don't need to nil the field — that nil-write was
// previously a race with Run's read of e.sub.C.
func (e *Engine) Close() {
	e.closeOnce.Do(func() {
		e.sub.Close()
	})
}

// Run drains grant events from the bus and runs the watchdog until ctx
// cancels. Returns ctx.Err(). Run does NOT close the engine's
// subscription; call Close when you're done with the engine.
func (e *Engine) Run(ctx context.Context) error {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			e.shutdown()
			return ctx.Err()
		case ev, ok := <-e.sub.C:
			if !ok {
				return nil
			}
			switch ev.Kind {
			case events.KindGrant:
				if g, ok := ev.Payload.(Grant); ok {
					e.HandleGrant(g)
				}
			case events.KindPatch:
				if p, ok := ev.Payload.(Patch); ok {
					e.handlePatch(p)
				}
			case events.KindCallEncryption:
				if c, ok := ev.Payload.(CallEncryption); ok {
					e.handleCallEncryption(c)
				}
			case events.KindCallSourceUpdate:
				if c, ok := ev.Payload.(CallSourceUpdate); ok {
					e.handleCallSourceUpdate(c)
				}
			case events.KindTalkerAlias:
				if a, ok := ev.Payload.(TalkerAlias); ok {
					e.handleTalkerAlias(a)
				}
			}
		case <-tick.C:
			e.runWatchdog()
		}
	}
}

// HandleGrant is the engine's grant-dispatch entrypoint. It is exported
// so tests can drive it directly without a running event loop.
// discoverTalkgroup adds a placeholder record for a talkgroup heard for the
// first time on the control channel, so the Talkgroups database self-populates
// from live traffic instead of requiring a hand-maintained CSV. The record
// inherits the loaders' Stream/Record defaults and is tagged "Discovered" so
// operator UIs can tell auto-learned entries from catalogued ones. Scan tracks
// the engine's scan mode: in ScanModeList a discovered TG is catalogued but not
// auto-scanned, so learning a new TG never silently widens a curated scan list
// — the operator opts it in from the UI. The record lives in memory only (it is
// gone on restart, like the rest of the in-memory TalkgroupDB).
func (e *Engine) discoverTalkgroup(g Grant) *TalkGroup {
	tg := &TalkGroup{
		ID:     g.GroupID,
		Tag:    "Discovered",
		Mode:   "D",
		Scan:   e.ScanMode() != ScanModeList,
		Stream: true,
		Record: true,
	}
	e.talkgroups.Add(tg)
	e.log.Info("talkgroup discovered on control channel",
		"tg", g.GroupID, "system", g.System, "scan", tg.Scan)
	return tg
}

func (e *Engine) HandleGrant(g Grant) {
	if g.At.IsZero() {
		g.At = e.now()
	}
	if g.FrequencyHz == 0 {
		e.log.Warn("dropping grant with zero frequency", "grant", g.String())
		return
	}
	tg := e.talkgroups.Lookup(g.GroupID)
	if tg == nil && g.GroupID != 0 && !g.Individual {
		// First time this talkgroup has been heard: catalogue it so
		// Database → Talkgroups fills in from the air (the trunk-recorder
		// behaviour the operator asked for). Individual (unit-to-unit /
		// interconnect) grants are skipped — their 24-bit destination is a
		// subscriber address, not a talkgroup.
		tg = e.discoverTalkgroup(g)
	}
	if tg != nil && tg.Lockout && !g.Emergency {
		e.log.Info("grant locked out", "grant", g.String(), "tg", tg.AlphaTag)
		return
	}
	// Patch attribution: when GroupID is an active patch super-group,
	// the call is physically the shared traffic of its member
	// talkgroups. Tag the grant with the members so the call is
	// attributed to each of them.
	if members := e.patches.MembersOf(g.GroupID); len(members) > 0 {
		g.PatchedGroups = members
	}
	// Scan list gate: in ScanModeList, drop grants whose TG is missing
	// or has Scan==false (Emergency bypasses, matching Lockout's
	// emergency exception above). A patched super-group passes if the
	// super-group OR any member talkgroup is scanned. In ScanModeAll
	// the gate is a no-op.
	if e.ScanMode() == ScanModeList && !g.Emergency {
		scanned := tg != nil && tg.Scan
		for _, m := range g.PatchedGroups {
			if mt := e.talkgroups.Lookup(m); mt != nil && mt.Scan {
				scanned = true
				break
			}
		}
		if !scanned {
			e.log.Debug("grant not in scan list", "grant", g.String())
			return
		}
	}

	// Encrypted-call policy (mode: ignore): never tie up a voice SDR on a
	// grant already flagged encrypted. Only dropped here when the grant
	// itself signals encryption (P25 Phase 2 grants carry it; Phase 1
	// discovers it mid-call, handled in applyEncryptedPolicy). A system
	// the operator has keys for is left to the in-call handlers, which
	// know the KeyID and can exempt a decryptable call — dropping here
	// would discard a call the operator may want to capture. Emergency
	// grants follow the existing lockout/scan-list precedent and bypass
	// the policy. Issue #711.
	if g.Encrypted && !g.Emergency && e.encModeFor(g.System) == EncryptedIgnore &&
		!e.keyConfigured(g.System, g.KeyID) && !e.systemHasKeys(g.System) {
		e.log.Debug("dropping encrypted grant (encrypted_calls mode: ignore)",
			"grant", g.String())
		return
	}

	// Record the call on the system-activity tracker before the tuner-
	// allocation logic below, so a talkgroup that keys up while every voice
	// device is busy still surfaces in ObservedCalls (the P25 "only one active
	// talkgroup" report). Repeated grant TSBKs for a live call refresh the
	// entry; data grants are not voice calls and are skipped. ObservedCalls
	// filters out whichever of these a tuner ends up following.
	if !g.DataCall {
		e.observeCall(g, tg)
	}

	// Suppress duplicate grants. The Phase 1 CC repeats voice-grant
	// TSBKs while a call is active (the user's issue #356 log shows
	// two grants for tg=32181 freq=773431250 arriving 20 ms apart),
	// and without this guard the engine binds a second voice SDR to
	// the same call — wasting a tuner, producing a duplicate WAV, and
	// confusing the operator's view of which device is serving the
	// call. Treat a repeat grant as the CC re-asserting "this call is
	// still going" and refresh the existing bind's LastHeardAt. Skip
	// when GroupID is zero (grants without a TG can legitimately
	// share a frequency).
	//
	// Timeslot is part of the match: a DMR Tier III carrier runs two
	// independent calls, one per TDMA slot, so a TS2 grant must NOT be
	// folded into an active TS1 call on the same frequency (even when
	// both slots happen to carry the same talkgroup) — that would drop
	// the second call. For non-slotted protocols Timeslot is 0 on both
	// sides, so the comparison is a no-op.
	if g.GroupID != 0 {
		for _, ac := range e.pool.Active() {
			// A logical call is identified by (System, talkgroup, timeslot),
			// NOT by frequency: a call's frequency can change mid-call (a
			// band-plan IdentifierUpdate re-maps the channel, or the system
			// hands the call to a new channel). Matching on frequency made
			// such a re-grant miss, so the engine bound a *second* tap to the
			// same talkgroup — two "Active calls" rows for one call. System
			// keeps two systems' identical TG numbers apart; Timeslot keeps a
			// DMR Tier III carrier's two per-slot calls apart (issue #356).
			if ac.Grant.System != g.System || ac.Grant.GroupID != g.GroupID ||
				ac.Grant.Timeslot != g.Timeslot {
				continue
			}
			if g.FrequencyHz == ac.Grant.FrequencyHz {
				// Same channel: the CC is repeating the grant while the call
				// runs (issue #356). Refresh LastHeardAt and we're done.
				e.pool.Touch(ac.Device.Serial, e.now())
				e.log.Debug("grant already active; refreshed",
					"grant", g.String(), "device", ac.Device.Serial)
				return
			}
			// Same call, new frequency — follow it. Retune the bound device
			// in place when it can still reach the new channel; otherwise end
			// the stale bind and fall through to allocate a capable device.
			if fc, ok := ac.Device.Tuner.(FrequencyChecker); !ok || fc.CanTune(g.FrequencyHz) {
				if err := e.pool.Retune(ac.Device.Serial, g, e.now()); err != nil {
					e.log.Warn("voice retune failed; rebinding",
						"err", err, "grant", g.String(), "device", ac.Device.Serial)
					e.endCall(ac, EndReasonNormal)
					break
				}
				e.log.Info("call followed to new frequency",
					"device", ac.Device.Serial, "grant", g.String())
				return
			}
			e.log.Info("call moved beyond device window; rebinding",
				"device", ac.Device.Serial, "grant", g.String())
			e.endCall(ac, EndReasonNormal)
			break
		}
	}

	// 1) Free device available? Allocate. FindFreeForFrequency skips
	// virtual voice tuners whose wideband window doesn't cover the
	// grant — so a P25 voice grant outside the wideband band falls
	// through to a physical role: voice SDR (when one is configured)
	// instead of bouncing on a tap that would reject it at Bind time.
	if free := e.pool.FindFreeForFrequency(g.FrequencyHz); free != nil {
		e.startCall(free, g, tg)
		return
	}
	// 2) No free device can serve this frequency. Look at the lowest-
	// priority active call *on a device that can tune the grant* —
	// preempting a device whose window excludes the frequency would
	// end an existing call to free a tuner that then can't bind the
	// incoming grant.
	victim := e.pool.LowestPriorityActiveForFrequency(g.FrequencyHz)
	if victim == nil {
		// No capable device is busy with a preemptable call. Work out
		// which of three situations we're in so the operator gets an
		// actionable message instead of a misleading one.
		if len(e.pool.Devices()) == 0 {
			// Pool has zero devices — trunking is configured but no
			// `role: voice` SDR (or wideband voice tap) is attached, so
			// every grant is dropped. Log loudly once, then DEBUG for
			// the rest of the daemon's life so we don't spam per grant.
			e.noVoiceSDROnce.Do(func() {
				e.log.Warn("no voice SDR available; voice grants will be dropped — add a role: voice device, or a role: wideband device with voice_taps (see docs/hardware.md)",
					"grant", g.String())
			})
			e.log.Debug("dropping grant: no voice SDR", "grant", g.String())
			return
		}
		if !e.pool.HasCapableDevice(g.FrequencyHz) {
			// Devices exist but none can tune this frequency — e.g.
			// every voice device is a wideband tap and the grant falls
			// outside its IQ window. A coverage gap, not an engine bug.
			e.noVoiceCoverageOnce.Do(func() {
				e.log.Warn("voice grant frequency outside every voice device's tuning window; widen sdr.sample_rate / adjust center_freq_hz, or add a role: voice SDR (see docs/hardware.md)",
					"grant", g.String())
			})
			e.log.Debug("dropping grant: no voice device covers frequency", "grant", g.String())
			return
		}
		// A capable device exists, none is free (step 1 failed) and
		// none is active — unreachable unless the active-tracking
		// invariant broke. Surface as Error so the bug is visible.
		e.log.Error("voice pool full but no actives (engine bug)", "grant", g.String())
		return
	}
	if !CanPreempt(victim.Grant, victim.Talkgroup, g, tg) {
		e.log.Info("no voice device available for grant", "grant", g.String())
		return
	}
	// 3) Preempt: end victim, allocate freed device.
	e.endCall(victim, EndReasonPreempted)
	e.startCall(victim.Device, g, tg)
}

// handleCallEncryption updates the bound ActiveCall's Grant with the
// recovered ALGID/KID from an in-call Encryption Sync, then republishes
// the event with the call's system / protocol / group context so SSE +
// TUI consumers can patch their live view. Skipped when System is
// already populated — that's the engine's own re-publish coming back
// through the subscription. Logged-and-dropped when no active call
// matches the device serial (e.g. the call already ended).
func (e *Engine) handleCallEncryption(c CallEncryption) {
	if c.System != "" {
		return
	}
	// Trunked calls are bound through the voice pool; synthetic calls
	// (conventional FM scanner) live on the engine. Try both, pool first.
	g, ok := e.pool.UpdateEncryption(c.DeviceSerial, c.AlgorithmID, c.KeyID)
	if !ok {
		e.mu.Lock()
		if ac, sok := e.synthetic[c.DeviceSerial]; sok {
			ac.Grant.AlgorithmID = c.AlgorithmID
			ac.Grant.KeyID = c.KeyID
			g = ac.Grant
			ok = true
		}
		e.mu.Unlock()
	}
	if !ok {
		e.log.Debug("call encryption update for unknown call",
			"device", c.DeviceSerial)
		return
	}
	enriched := CallEncryption{
		DeviceSerial:     c.DeviceSerial,
		System:           g.System,
		Protocol:         g.Protocol,
		GroupID:          g.GroupID,
		AlgorithmID:      c.AlgorithmID,
		KeyID:            c.KeyID,
		MessageIndicator: c.MessageIndicator,
		At:               c.At,
	}
	e.bus.Publish(events.Event{
		Kind:    events.KindCallEncryption,
		Payload: enriched,
	})
	e.log.Info("call encryption update",
		"device", c.DeviceSerial,
		"system", enriched.System,
		"tg", enriched.GroupID,
		"alg", c.AlgorithmID, "key", c.KeyID)
	e.applyEncryptedPolicy(c.DeviceSerial, g, c.AlgorithmID != algorithmClear)
}

// handleCallSourceUpdate backfills SourceID + Encrypted on the bound
// ActiveCall when an in-call GROUP_VOICE_CHANNEL_USER PDU arrives on
// the voice channel — used by P25 Phase 2 systems whose CC grant
// arrives in a compressed form (src=0 + enc=false) and the source
// RID + encryption state only surface in-call on the traffic
// channel. Republishes the event with the call's system / protocol
// / group context so SSE + TUI consumers can patch their live view.
// Skipped when System is already populated — the engine's own
// re-publish coming back through the subscription. Logged-and-
// dropped when no active call matches the device serial (e.g. the
// call already ended).
func (e *Engine) handleCallSourceUpdate(c CallSourceUpdate) {
	if c.System != "" {
		return
	}
	g, ok := e.pool.UpdateSource(c.DeviceSerial, c.SourceID, c.Encrypted)
	if !ok {
		e.mu.Lock()
		if ac, sok := e.synthetic[c.DeviceSerial]; sok {
			if c.SourceID != 0 {
				ac.Grant.SourceID = c.SourceID
			}
			if c.Encrypted {
				ac.Grant.Encrypted = true
			}
			g = ac.Grant
			ok = true
		}
		e.mu.Unlock()
	}
	if !ok {
		e.log.Debug("call source update for unknown call",
			"device", c.DeviceSerial)
		return
	}
	enriched := CallSourceUpdate{
		DeviceSerial: c.DeviceSerial,
		System:       g.System,
		Protocol:     g.Protocol,
		GroupID:      g.GroupID,
		SourceID:     g.SourceID,
		Encrypted:    g.Encrypted,
		At:           c.At,
	}
	e.bus.Publish(events.Event{
		Kind:    events.KindCallSourceUpdate,
		Payload: enriched,
	})
	e.log.Info("call source update",
		"device", c.DeviceSerial,
		"system", enriched.System,
		"tg", enriched.GroupID,
		"src", enriched.SourceID,
		"enc", enriched.Encrypted)
	e.applyEncryptedPolicy(c.DeviceSerial, g, g.Encrypted)
}

// handlePatch applies a patch announcement to the registry: an Add
// records the super-group → members mapping so later grants on the
// super-group are attributed to its members; a cancel drops it.
func (e *Engine) handlePatch(p Patch) {
	if p.Add {
		e.patches.Apply(PatchGroup{
			SuperGroup: p.SuperGroup,
			Members:    p.Members,
			Vendor:     p.Vendor,
			UpdatedAt:  e.now(),
		})
		e.log.Debug("patch group active",
			"super", p.SuperGroup, "members", p.Members, "vendor", p.Vendor)
		return
	}
	e.patches.Delete(p.SuperGroup)
	e.log.Debug("patch group cancelled", "super", p.SuperGroup)
}

// Patches returns a snapshot of the engine's active patch groups.
func (e *Engine) Patches() []PatchGroup { return e.patches.Active() }

// EndCall is the explicit external signal that a call has ended (e.g.
// the protocol decoder saw a channel-release announcement, or an upstream
// test wants to release the device). reason is published in the CallEnd
// event payload.
func (e *Engine) EndCall(deviceSerial string, reason EndReason) bool {
	e.mu.Lock()
	ac, ok := e.calls[deviceSerial]
	e.mu.Unlock()
	if !ok {
		return false
	}
	e.endCall(ac, reason)
	return true
}

// Touch refreshes the LastHeardAt timestamp on the active call bound to
// deviceSerial. The protocol decoder calls this every time it sees voice
// activity on the followed frequency so the watchdog doesn't time it out.
func (e *Engine) Touch(deviceSerial string) {
	e.pool.Touch(deviceSerial, e.now())
}

// ActiveCalls returns a snapshot of every active call — trunked
// calls allocated through the voice pool plus synthetic calls owned
// by external scanners (the conventional FM scanner publishes these
// through HandleSyntheticCall).
func (e *Engine) ActiveCalls() []*ActiveCall {
	out := e.pool.Active()
	e.mu.Lock()
	for _, ac := range e.synthetic {
		out = append(out, ac)
	}
	e.mu.Unlock()
	return out
}

// observeCall upserts the system-activity tracker entry for grant g. Called
// for every voice grant HandleGrant accepts, whether or not a tuner follows it.
func (e *Engine) observeCall(g Grant, tg *TalkGroup) {
	key := observedKey(g)
	e.mu.Lock()
	defer e.mu.Unlock()
	if ac, ok := e.observed[key]; ok {
		ac.Grant = g
		ac.Talkgroup = tg
		ac.LastHeardAt = g.At
		return
	}
	e.observed[key] = &ActiveCall{
		Grant:       g,
		Talkgroup:   tg,
		StartedAt:   g.At,
		LastHeardAt: g.At,
		// Device intentionally nil: this call is known only from the control
		// channel, not bound to a voice tuner.
	}
}

// ObservedCalls returns the voice calls the control channel has announced that
// are NOT currently being followed by a voice tuner — e.g. talkgroups that
// keyed up while every voice device was busy. They carry a nil Device. Combined
// with ActiveCalls at the API layer, they let an operator see every talkgroup
// up on the system (the P25 "only one active talkgroup" report), while audio
// stays limited to the number of voice tuners. Entries age out via runWatchdog.
func (e *Engine) ObservedCalls() []*ActiveCall {
	// Build the set of (system|tg|timeslot) keys a tuner is actively following
	// so they aren't double-reported as observed-only.
	followed := make(map[string]bool)
	for _, ac := range e.pool.Active() {
		followed[observedKey(ac.Grant)] = true
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ac := range e.synthetic {
		followed[observedKey(ac.Grant)] = true
	}
	out := make([]*ActiveCall, 0, len(e.observed))
	for key, ac := range e.observed {
		if followed[key] {
			continue
		}
		out = append(out, ac)
	}
	return out
}

func (e *Engine) startCall(d *VoiceDevice, g Grant, tg *TalkGroup) {
	ac, err := e.pool.Bind(d, g, tg, e.now())
	if err != nil {
		e.log.Warn("voice bind failed", "err", err, "grant", g.String())
		return
	}
	e.mu.Lock()
	e.calls[d.Serial] = ac
	e.mu.Unlock()
	e.bus.Publish(events.Event{
		Kind: events.KindCallStart,
		// Publish ac.Grant, not the caller's g: Bind stamped it with a
		// fresh CallID the voice chain + recorder use to fence cross-call
		// audio bleed on a reused tap serial.
		Payload: CallStart{
			Grant:        ac.Grant,
			Talkgroup:    tg,
			DeviceSerial: d.Serial,
			StartedAt:    ac.StartedAt,
		},
	})
	e.log.Info("call started",
		"device", d.Serial,
		"grant", g.String(),
		"priority", EffectivePriority(g, tg))
	// Apply the encrypted-call policy for grants that already signal
	// encryption (P25 Phase 2). Phase 1 / compressed grants surface
	// encryption mid-call via handleCallEncryption / handleCallSourceUpdate.
	e.applyEncryptedPolicy(d.Serial, g, g.Encrypted)
}

// applyEncryptedPolicy enforces the trunking.encrypted_calls policy on
// the pool-bound call on serial once it is known to be encrypted.
// encrypted is computed by the caller from the context it has (grant
// flag, encryption-sync ALGID, or source-update flag). A decryptable
// call (operator holds a matching key) is always followed — any pending
// metadata release is cancelled. Otherwise: ignore ends the call now;
// metadata arms the watchdog to release it after the follow window;
// follow does nothing. Synthetic (conventional-FM) calls aren't pool-
// bound, so this is a no-op for them. Issue #711.
func (e *Engine) applyEncryptedPolicy(serial string, g Grant, encrypted bool) {
	if !encrypted {
		return
	}
	if e.keyConfigured(g.System, g.KeyID) {
		e.pool.DisarmEncryptedRelease(serial)
		return
	}
	switch e.encModeFor(g.System) {
	case EncryptedIgnore:
		e.mu.Lock()
		ac := e.calls[serial]
		e.mu.Unlock()
		if ac != nil {
			e.log.Info("releasing encrypted call (encrypted_calls mode: ignore)",
				"device", serial, "grant", g.String())
			e.endCall(ac, EndReasonEncrypted)
		}
	case EncryptedMetadata:
		e.pool.ArmEncryptedRelease(serial, e.now().Add(e.encMetadataFollowFor(g.System)))
	}
}

// handleTalkerAlias short-circuits the metadata-mode hold: once a system's
// talker alias has fully reassembled (P25 Phase 2 FACCH-S header + data
// blocks, decoded during call hangtime), the reason we held the tuner is
// satisfied, so release it right away rather than waiting out the rest of
// the metadata_follow_ms window. Only calls already armed for an encrypted
// release are affected — a follow-mode call, a clear call, or one not yet
// known encrypted is left running. Issue #711.
func (e *Engine) handleTalkerAlias(a TalkerAlias) {
	ac := e.pool.ArmedCallBySource(a.System, a.SourceID)
	if ac == nil {
		return
	}
	e.log.Info("releasing encrypted call after talker alias completed (encrypted_calls mode: metadata)",
		"device", ac.Device.Serial, "grant", ac.Grant.String(), "alias", a.Alias)
	e.endCall(ac, EndReasonEncrypted)
}

func (e *Engine) endCall(ac *ActiveCall, reason EndReason) {
	released := e.pool.Release(ac.Device.Serial)
	if released == nil {
		return // already released elsewhere
	}
	e.mu.Lock()
	delete(e.calls, ac.Device.Serial)
	e.mu.Unlock()
	e.bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: CallEnd{
			Grant:        released.Grant,
			Talkgroup:    released.Talkgroup,
			DeviceSerial: ac.Device.Serial,
			StartedAt:    released.StartedAt,
			EndedAt:      e.now(),
			Reason:       reason,
		},
	})
	e.log.Info("call ended",
		"device", ac.Device.Serial,
		"grant", released.Grant.String(),
		"reason", reason.String())
}

func (e *Engine) runWatchdog() {
	now := e.now()
	// Encrypted-call policy (mode: metadata): release any call whose
	// metadata-follow window has elapsed. Done first so a call due for an
	// encrypted release isn't also reaped as an inactivity timeout in the
	// same tick. Issue #711.
	for _, ac := range e.pool.EncryptedReleasesDue(now) {
		e.log.Debug("watchdog: releasing encrypted call after metadata window",
			"device", ac.Device.Serial, "grant", ac.Grant.String())
		e.endCall(ac, EndReasonEncrypted)
	}
	cutoff := now.Add(-e.timeout)
	for _, ac := range e.pool.Active() {
		if ac.LastHeardAt.Before(cutoff) {
			// Distinguish carrier-drop natural end from silent-from-
			// start decode failure. P25 trunking has no explicit
			// channel-release message on the CC for most calls, so
			// the only natural end-of-call signal IS the grace
			// timeout after the last LDU. A call whose LastHeardAt
			// advanced past StartedAt received frames at least once
			// — its end is "carrier dropped, watchdog reaped after
			// the grace window" → EndReasonNormal. A call whose
			// LastHeardAt is still equal to StartedAt never decoded
			// a single frame → EndReasonTimeout (the real failure
			// mode issue #356 wants to surface). Issue #356
			// follow-up: a field log showed three healthy calls all
			// reported as reason=timeout, leading the operator to
			// believe the decode was still broken when it was
			// actually a terminology problem.
			reason := EndReasonTimeout
			if ac.LastHeardAt.After(ac.StartedAt) {
				reason = EndReasonNormal
			}
			e.log.Debug("watchdog: reaping call",
				"device", ac.Device.Serial,
				"grant", ac.Grant.String(),
				"last_heard_at", ac.LastHeardAt,
				"started_at", ac.StartedAt,
				"now", now,
				"elapsed", now.Sub(ac.LastHeardAt),
				"timeout", e.timeout,
				"reason", reason)
			e.endCall(ac, reason)
		}
	}
	// Age out system-activity entries the control channel has stopped
	// repeating: once a call drops, its grant TSBKs stop, so the same grace
	// window that reaps a followed call clears its observed-only siblings.
	e.mu.Lock()
	for key, ac := range e.observed {
		if ac.LastHeardAt.Before(cutoff) {
			delete(e.observed, key)
		}
	}
	e.mu.Unlock()
}

func (e *Engine) shutdown() {
	for _, ac := range e.pool.Active() {
		e.endCall(ac, EndReasonNormal)
	}
}

// HandleSyntheticCall registers a call originated by a non-trunked
// source (the conventional FM scanner is the canonical example) that
// already owns its SDR — no VoicePool binding, no re-tune, no
// preemption logic. The engine publishes CallStart and adds the call
// to ActiveCalls() so the API + TUI surfaces light up like any
// other call. Pair with EndSyntheticCall to release.
//
// deviceSerial must be unique across the daemon's call set so the
// recorder can route WritePCM samples to the right WAV.
func (e *Engine) HandleSyntheticCall(g Grant, deviceSerial string) {
	if g.At.IsZero() {
		g.At = e.now()
	}
	tg := e.talkgroups.Lookup(g.GroupID)
	ac := &ActiveCall{
		Device:      &VoiceDevice{Serial: deviceSerial},
		Grant:       g,
		Talkgroup:   tg,
		StartedAt:   e.now(),
		LastHeardAt: e.now(),
	}
	e.mu.Lock()
	e.synthetic[deviceSerial] = ac
	e.mu.Unlock()
	e.bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: CallStart{
			Grant:        g,
			Talkgroup:    tg,
			DeviceSerial: deviceSerial,
			StartedAt:    ac.StartedAt,
		},
	})
	e.log.Info("synthetic call started",
		"device", deviceSerial,
		"grant", g.String())
}

// EndSyntheticCall is the conventional scanner's "carrier dropped"
// signal. Publishes CallEnd and forgets the call. Returns false if
// the engine has no synthetic call bound to deviceSerial.
func (e *Engine) EndSyntheticCall(deviceSerial string, reason EndReason) bool {
	e.mu.Lock()
	ac, ok := e.synthetic[deviceSerial]
	if ok {
		delete(e.synthetic, deviceSerial)
	}
	e.mu.Unlock()
	if !ok {
		return false
	}
	e.bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: CallEnd{
			Grant:        ac.Grant,
			Talkgroup:    ac.Talkgroup,
			DeviceSerial: deviceSerial,
			StartedAt:    ac.StartedAt,
			EndedAt:      e.now(),
			Reason:       reason,
		},
	})
	e.log.Info("synthetic call ended",
		"device", deviceSerial,
		"reason", reason.String())
	return true
}

// TalkgroupForDevice returns the talkgroup of the active call bound to
// deviceSerial, or nil when no call is active on that device. The live
// audio path uses it to honour the per-talkgroup Mute flag. Safe to
// call from any goroutine.
func (e *Engine) TalkgroupForDevice(deviceSerial string) *TalkGroup {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ac, ok := e.calls[deviceSerial]; ok {
		return ac.Talkgroup
	}
	if ac, ok := e.synthetic[deviceSerial]; ok {
		return ac.Talkgroup
	}
	return nil
}

// ScanMode returns the engine's current scan mode. Safe to call from
// any goroutine.
func (e *Engine) ScanMode() ScanMode {
	e.modeMu.RLock()
	defer e.modeMu.RUnlock()
	return e.scanMode
}

// SetScanMode swaps the engine's scan mode at runtime — the API
// cockpit calls this when the operator flips the global scan_mode
// from the TUI. Returns the previous mode so the caller can log /
// audit the change.
func (e *Engine) SetScanMode(m ScanMode) ScanMode {
	e.modeMu.Lock()
	defer e.modeMu.Unlock()
	prev := e.scanMode
	e.scanMode = m
	return prev
}

// encModeFor returns the encrypted-call handling mode configured for
// system (trunking.systems[].encrypted_calls.mode). A system with no
// override defaults to EncryptedFollow — the pre-issue-711 behaviour.
// The map is read-only after NewEngine, so no lock. Issue #711.
func (e *Engine) encModeFor(system string) EncryptedMode {
	return e.encModes[system] // zero value is EncryptedFollow
}

// encMetadataFollowFor returns the metadata-mode follow window configured
// for system. A system with no override (or a non-positive value) uses
// defaultEncryptedMetadataFollow. Read-only after NewEngine. Issue #711.
func (e *Engine) encMetadataFollowFor(system string) time.Duration {
	if d, ok := e.encFollows[system]; ok && d > 0 {
		return d
	}
	return defaultEncryptedMetadataFollow
}

// keyConfigured reports whether the operator supplied an encryption key
// whose key ID matches keyID for system — i.e. the call is "decryptable"
// (captured for in-process or out-of-band decode) and must be exempt
// from the encrypted-call policy. Issue #711.
func (e *Engine) keyConfigured(system string, keyID uint16) bool {
	ids := e.configuredKeys[system]
	return ids != nil && ids[keyID]
}

// systemHasKeys reports whether the operator supplied any encryption key
// for system. HandleGrant uses it to avoid dropping an encrypted grant at
// grant time (mode: ignore) on a system the operator has keys for — the
// grant may not carry a KeyID yet, so the decision is deferred to the
// in-call handlers where the KeyID is known. Issue #711.
func (e *Engine) systemHasKeys(system string) bool {
	return len(e.configuredKeys[system]) > 0
}
