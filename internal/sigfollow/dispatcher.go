// Package sigfollow decodes the MAC signalling that rides a P25 Phase 2
// traffic channel — talker alias, in-call source / encryption — off the
// channel's signalling stream, independent of whether a voice tuner is
// recording the call.
//
// On a busy multi-site Phase 2 system most grants never get a voice
// tuner (the voice pool can't cover the full voice spread), and
// encrypted calls are torn down before hangtime — so the talker-alias
// FACCH-S decode wired inside the voice composer almost never runs
// (issue #376). SDRTrunk harvests these aliases from the traffic
// channel's signalling stream with just two SDRs, without following the
// call as voice. This package mirrors that: a Manager spins up a
// signalling-only DDC tap on the wideband IQ broker for each in-window
// Phase 2 grant and feeds the decoded superframes through a shared
// MACDispatcher — the same dispatch the voice composer uses, so the two
// paths never diverge on what they decode.
package sigfollow

import (
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// MACDispatcher decodes the MAC PDUs interleaved with voice / signalling
// on one P25 Phase 2 traffic channel and publishes the talker-alias
// events they carry. Construct one per call / follow: the alias
// assemblers buffer fragments per source unit for the life of the
// channel. Not safe for concurrent use — drive it from a single
// goroutine (the receiver's DibitSink).
//
// The talker-alias path is identical for the voice composer and the
// signalling follower, so it lives here and publishes directly. The
// in-call source / encryption PDUs are call-bound (the engine backfills
// the active call keyed by device serial, and ignores updates whose
// System is already set), so they cannot be published identically by a
// follower that has no bound call. Callers handle those via the
// OnCallSource / OnCallEncryption hooks: the voice composer wires them to
// its engine-backfill publishers; the follower leaves them nil.
type MACDispatcher struct {
	bus       *events.Bus
	log       *slog.Logger
	logPrefix string
	system    string
	serial    string

	aliasAsm         *p25p2.TalkerAliasAssembler
	motorolaAliasAsm *p25p2.MotorolaAliasAssembler
	macSeen          map[uint16]struct{}

	onCallSource     func(p25p2.GroupVoiceChannelUser)
	onCallEncryption func(p25p2.EncryptionSync)
}

// MACDispatcherOptions configures a MACDispatcher.
type MACDispatcherOptions struct {
	// Bus is where decoded talker-alias events are published. Required
	// for any event to surface; a nil bus makes Dispatch a decode-only
	// no-op (useful in unit tests).
	Bus *events.Bus
	// Log labels diagnostics. Optional; defaults to slog.Default().
	Log *slog.Logger
	// LogPrefix tags the per-PDU census ("<prefix>: p25p2 mac pdu") and
	// alias ("<prefix>: p25p2 talker alias") log lines so an operator can
	// tell the voice composer ("composer") and the signalling follower
	// ("sigfollow") apart while keeping both greppable on the shared
	// "p25p2 mac pdu" / "p25p2 talker alias" substrings (issue #376).
	LogPrefix string
	// System / Serial identify the call this dispatcher serves. System
	// stamps the published TalkerAlias; Serial labels diagnostics.
	System string
	Serial string
	// OnCallSource / OnCallEncryption handle the in-call, call-bound MAC
	// PDUs. Optional; nil skips them (the follower's case).
	OnCallSource     func(p25p2.GroupVoiceChannelUser)
	OnCallEncryption func(p25p2.EncryptionSync)
}

// NewMACDispatcher returns a ready dispatcher.
func NewMACDispatcher(opts MACDispatcherOptions) *MACDispatcher {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	prefix := opts.LogPrefix
	if prefix == "" {
		prefix = "sigfollow"
	}
	return &MACDispatcher{
		bus:              opts.Bus,
		log:              log,
		logPrefix:        prefix,
		system:           opts.System,
		serial:           opts.Serial,
		aliasAsm:         p25p2.NewTalkerAliasAssembler(nil),
		motorolaAliasAsm: p25p2.NewMotorolaAliasAssembler(nil),
		macSeen:          make(map[uint16]struct{}),
		onCallSource:     opts.OnCallSource,
		onCallEncryption: opts.OnCallEncryption,
	}
}

// Dispatch decodes every MAC PDU in sf under macCfg, publishing any
// completed talker alias and invoking the source / encryption hooks. It
// returns the number of MAC PDUs decoded so a caller can drive an
// activity watchdog off signalling presence (0 means the superframe
// carried only voice / idle).
func (d *MACDispatcher) Dispatch(sf p25p2.Superframe, macCfg p25p2.MACDecodeConfig) int {
	pdus := p25p2.DecodeSuperframeMACPDUs(sf, macCfg)
	for _, pdu := range pdus {
		// Log the first PDU seen per (opcode, MFID) on this channel. If a
		// real on-air system emits an opcode we don't dispatch, the line
		// tells the next field tester exactly what we saw (issue #376).
		key := uint16(pdu.Opcode)<<8 | uint16(pdu.MFID)
		if _, seen := d.macSeen[key]; !seen {
			d.macSeen[key] = struct{}{}
			d.log.Info(d.logPrefix+": p25p2 mac pdu",
				"system", d.system, "serial", d.serial,
				"opcode", pdu.Opcode, "mfid", pdu.MFID,
				"payload_len", len(pdu.Payload))
		}
		if u, ok := pdu.AsGroupVoiceChannelUser(); ok {
			if d.onCallSource != nil {
				d.onCallSource(u)
			}
			continue
		}
		if es, ok := pdu.AsEncryptionSync(); ok {
			if d.onCallEncryption != nil {
				d.onCallEncryption(es)
			}
			continue
		}
		if f, ok := pdu.AsTalkerAliasFragment(); ok {
			// Generic working-model fragment path (plain ASCII, not the
			// validated Motorola cipher) — treat as reliable; the
			// corruption check applies to the Motorola FACCH-S decode
			// below (#711).
			if alias, src, complete := d.aliasAsm.Add(f); complete {
				d.publishTalkerAlias(src, alias, true)
			}
			continue
		}
		// Real Motorola FACCH-S alias: header (0x91) seeds the per-channel
		// reassembler, data blocks (0x95) complete it, and the source RID
		// falls out of the decoded message prefix (#376).
		if h, ok := pdu.AsMotorolaAliasHeader(); ok {
			if alias, src, reliable, complete := d.motorolaAliasAsm.AddHeader(h); complete {
				d.publishTalkerAlias(src, alias, reliable)
			}
			continue
		}
		if dat, ok := pdu.AsMotorolaAliasData(); ok {
			if alias, src, reliable, complete := d.motorolaAliasAsm.AddData(dat); complete {
				d.publishTalkerAlias(src, alias, reliable)
			}
			continue
		}
	}
	return len(pdus)
}

// publishTalkerAlias mirrors phase2.ControlChannel.publishTalkerAlias: a
// completed alias reassembled off the traffic channel surfaces on the
// bus with the same payload shape as one decoded on the CC. The
// affiliation tracker binds it onto the RID by (System, SourceID).
func (d *MACDispatcher) publishTalkerAlias(sourceID uint32, alias string, reliable bool) {
	if d.bus == nil || alias == "" {
		return
	}
	d.bus.Publish(events.Event{
		Kind: events.KindTalkerAlias,
		Payload: trunking.TalkerAlias{
			System:     d.system,
			Protocol:   "p25-phase2",
			SourceID:   sourceID,
			Alias:      alias,
			Unreliable: !reliable,
			At:         time.Now().UTC(),
		},
	})
	d.log.Info(d.logPrefix+": p25p2 talker alias",
		"system", d.system, "src", sourceID, "alias", alias, "unreliable", !reliable)
}
