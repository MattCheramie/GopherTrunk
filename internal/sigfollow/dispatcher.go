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
// goroutine (the receiver's DibitSink). It decodes the real TDMA air
// format (p25p2.AirFramer bursts, DecodeAirMAC); the older superframe
// working model is no longer used here (#1273).
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

	motorolaAliasAsm *p25p2.MotorolaAliasAssembler
	macSeen          map[uint32]struct{}

	onCallSource     func(p25p2.GroupVoiceChannelUser)
	onCallEncryption func(p25p2.EncryptionSync)
}

// MACDispatcherOptions configures a MACDispatcher.
type MACDispatcherOptions struct {
	// Bus is where decoded talker-alias events are published. Required
	// for any event to surface; a nil bus makes DispatchAir a decode-only
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
		motorolaAliasAsm: p25p2.NewMotorolaAliasAssembler(nil),
		macSeen:          make(map[uint32]struct{}),
		onCallSource:     opts.OnCallSource,
		onCallEncryption: opts.OnCallEncryption,
	}
}

// DispatchAir decodes the MAC PDUs in a run of air-format timeslot bursts
// (p25p2.AirFramer output) and handles their messages: SACCH, FACCH and
// LCCH bursts go through p25p2.DecodeAirMAC (RS(63,35) + CRC, descrambled
// with masks for the scrambled DUIDs; nil masks decode the clear ones only),
// voice and other bursts are skipped. A MAC_PTT's fixed fields are the
// call's encryption sync; every other PDU's messages are dispatched. It
// returns how many signalling bursts it saw and how many of those decoded
// CRC-valid — the framing-health pair the per-call census reports (#915).
func (d *MACDispatcher) DispatchAir(bursts []p25p2.AirBurst, masks *p25p2.AirMasks) (signalling, decoded int) {
	for _, b := range bursts {
		if !b.DUID.IsSignalling() {
			continue
		}
		signalling++
		m, ok := p25p2.DecodeAirMAC(b, masks)
		if !ok {
			continue
		}
		decoded++
		if p, ok := m.PTT(); ok {
			d.noteFirst(m, p25p2.MACPDU{}, hex.EncodeToString(m.Bytes))
			if d.onCallEncryption != nil {
				d.onCallEncryption(p.Encryption)
			}
			// MAC_PTT names the keying radio too (OP25 handle_mac_ptt;
			// ALGID 0x80 is clear):
			// the in-call source on a compressed grant (#915).
			if p.Source != 0 && d.onCallSource != nil {
				u := p25p2.GroupVoiceChannelUser{GroupAddress: p.Group, SourceID: p.Source}
				if alg := p.Encryption.AlgorithmID; alg != 0 && alg != 0x80 {
					u.ServiceOptions = 0x40 // encrypted
				}
				d.onCallSource(u)
			}
			continue
		}
		for _, pdu := range m.Messages() {
			d.noteFirst(m, pdu, hex.EncodeToString(pdu.Payload))
			d.dispatchMessage(pdu)
		}
	}
	return signalling, decoded
}

// noteFirst logs the first message seen per (channel, PDU type, opcode,
// MFID) on this traffic channel with its bytes: the instrument that tells
// the next field tester exactly what was on the air (issues #376, #813,
// #915).
func (d *MACDispatcher) noteFirst(m p25p2.AirMAC, pdu p25p2.MACPDU, payloadHex string) {
	key := uint32(m.Channel)<<24 | uint32(m.Type)<<16 | uint32(pdu.Opcode)<<8 | uint32(pdu.MFID)
	if _, seen := d.macSeen[key]; seen {
		return
	}
	d.macSeen[key] = struct{}{}
	d.log.Info(d.logPrefix+": p25p2 mac pdu",
		"system", d.system, "serial", d.serial,
		"channel", m.Channel, "pdu_type", m.Type, "opcode", pdu.Opcode, "mfid", pdu.MFID,
		"rs_corrected", m.Corrected, "payload_hex", payloadHex)
}

// dispatchMessage handles one CRC-verified MAC message.
func (d *MACDispatcher) dispatchMessage(pdu p25p2.MACPDU) {
	if u, ok := pdu.AsGroupVoiceChannelUser(); ok {
		// The completed-call webhook's source_rid is backfilled from this
		// in-call message (#915). It arrives CRC-verified, so it is trusted.
		if d.onCallSource != nil {
			d.onCallSource(u)
		}
		return
	}
	if es, ok := pdu.AsEncryptionSync(); ok {
		if d.onCallEncryption != nil {
			d.onCallEncryption(es)
		}
		return
	}
	// Real Motorola FACCH-S alias: header (0x91) seeds the per-channel
	// reassembler, data blocks (0x95) complete it, and the source RID falls
	// out of the decoded message prefix (#376).
	if h, ok := pdu.AsMotorolaAliasHeader(); ok {
		if res := d.motorolaAliasAsm.AddHeader(h); res.Complete {
			d.completeMotorolaAlias(res)
		}
		return
	}
	if dat, ok := pdu.AsMotorolaAliasData(); ok {
		if res := d.motorolaAliasAsm.AddData(dat); res.Complete {
			d.completeMotorolaAlias(res)
		}
	}
}

// completeMotorolaAlias handles a fully reassembled Motorola FACCH-S
// alias. It publishes the decoded alias (when non-empty) and always logs
// the reassembled ciphertext.
//
// The proprietary alias cipher is now verified (motorola.CipherVerified),
// so a clean decode publishes the radio's real alias (e.g. "CRIO 0062")
// and the alias card shows it (#773). The reassembled *ciphertext* — the
// 2n encoded-alias bytes paired with the source RID and talkgroup, plus the
// CRC-OK flag — is still logged for every completed alias as an audit trail:
// multiple identical lines across keyups are expected and confirm the
// ciphertext is stable (deterministic).
func (d *MACDispatcher) completeMotorolaAlias(res p25p2.MotorolaAliasResult) {
	if len(res.Encoded) > 0 {
		d.log.Info(d.logPrefix+": p25p2 alias ciphertext",
			"system", d.system, "serial", d.serial,
			"rid", res.SourceID, "talkgroup", res.TalkgroupID,
			"encoded_hex", hex.EncodeToString(res.Encoded),
			"crc_ok", res.CRCOK, "reliable", res.Reliable)
	}
	d.publishTalkerAlias(res.SourceID, res.Alias, res.Reliable)
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
