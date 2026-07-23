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
	"encoding/hex"
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
	macSeen          map[uint32]struct{}

	// scramblerPin lets ScramblerProbe self-align to this channel's PN44
	// phase once and reuse it, instead of sweeping on every MAC burst (issue
	// #915). One per dispatcher, driven single-threaded from Dispatch.
	scramblerPin *p25p2.ScramblerPin

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
		macSeen:          make(map[uint32]struct{}),
		scramblerPin:     &p25p2.ScramblerPin{},
		onCallSource:     opts.OnCallSource,
		onCallEncryption: opts.OnCallEncryption,
	}
}

// Dispatch decodes every MAC PDU in sf under macCfg, publishing any
// completed talker alias and invoking the source / encryption hooks. It
// returns the number of MAC PDUs decoded (0 means the superframe carried
// only voice / idle) and, of those, how many carried a valid outer
// RS(24, 16, 9) parity — the framing-health signal a caller feeds the
// per-call census (issue #915): on a correctly framed + descrambled
// channel nearly every PDU is RS-valid, whereas a mis-framed traffic
// channel decodes a stream of random bytes that (almost) never verify.
func (d *MACDispatcher) Dispatch(sf p25p2.Superframe, macCfg p25p2.MACDecodeConfig) (decodedCount, rsValidCount int) {
	decoded := p25p2.DecodeSuperframeMACPDUsWithSlotPinned(sf, macCfg, d.scramblerPin)
	for _, dec := range decoded {
		pdu := dec.PDU
		if dec.RSValid {
			rsValidCount++
		}
		// Log the first PDU seen per (slot, opcode, MFID) on this channel,
		// with the payload bytes and its RS-integrity flag. If a real
		// on-air system rides the encryption sync on a slot/opcode we don't
		// dispatch, the hex tells the next field tester exactly what we saw
		// — the bytes needed to confirm or correct the MAC_PTT layout; and
		// rs_valid=false across the board is the fingerprint of a mis-framed
		// superframe rather than an unhandled-but-real opcode (issues #376,
		// #813, #915).
		key := uint32(dec.SlotType)<<16 | uint32(pdu.Opcode)<<8 | uint32(pdu.MFID)
		if _, seen := d.macSeen[key]; !seen {
			d.macSeen[key] = struct{}{}
			d.log.Info(d.logPrefix+": p25p2 mac pdu",
				"system", d.system, "serial", d.serial,
				"slot", dec.SlotType, "opcode", pdu.Opcode, "mfid", pdu.MFID,
				"rs_valid", dec.RSValid,
				"payload_len", len(pdu.Payload),
				"payload_hex", hex.EncodeToString(pdu.Payload))
		}
		// MAC_PTT slot: the key-up message that carries the encryption sync
		// (ALGID/KID/MI) on real Phase 2 systems. Routed by slot type, not
		// opcode, because the PTT message has no normal MAC opcode (#813).
		if dec.SlotType == p25p2.SlotTypeMACPTT {
			if es, ok := pdu.AsPushToTalk(); ok {
				if d.onCallEncryption != nil {
					d.onCallEncryption(es)
				}
				continue
			}
		}
		if u, ok := pdu.AsGroupVoiceChannelUser(); ok {
			// Source-RID integrity gate (issue #915). The completed-call
			// webhook's source_rid is backfilled from this in-call PDU, so a
			// mis-decoded MAC window whose opcode byte happens to land on
			// 0x01/0x21 would inject a plausible-but-wrong RID that is
			// indistinguishable downstream from a real one — the source-side
			// analogue of the #924 algid smear. The outer RS parity is the
			// only signal that separates a genuine GROUP_VOICE_CHANNEL_USER
			// from garbage (the opcode alone can't), so only an RS-verified
			// PDU is trusted to set the call's source. A wrong RID is worse
			// than an absent one; the real recovery of the ~64% missing on
			// MMR is a Phase 2 superframe-framing fix, tracked in #915.
			if dec.RSValid && d.onCallSource != nil {
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
	return len(decoded), rsValidCount
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
