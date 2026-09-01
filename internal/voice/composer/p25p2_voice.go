package composer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	p25p2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sigfollow"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25p2VoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the P25 Phase 2 receiver runs. 48 kHz gives the 6000-baud
// H-DQPSK symbol stream 8 samples per symbol — ample for the matched
// filter and Gardner clock recovery without running them at the SDR's
// native multi-MS/s rate.
const p25p2VoiceIntermediateHz = 48_000

// p25p2ChannelSelectHz caps the voice front-end channel filter at half the
// 12.5 kHz P25 channel spacing. A wideband DDC voice tap hands the chain a
// stream already at the intermediate rate but band-limited only to the tap's
// output Nyquist (~±24 kHz), NOT to a single 12.5 kHz channel — so the old
// pass-through front end fed that whole ±24 kHz span into the receiver. Unlike
// Phase 1 (C4FM/FM discriminator) there is no FM capture effect here: Phase 2
// is linear H-DQPSK, and adjacent-channel energy instead pumps the receiver's
// AGC down (shrinking the wanted constellation's decision margin) and raises the
// noise floor the coarse-carrier / Costas loops estimate against — degrading
// carrier recovery and raising EVM. Filtering to ±6.25 kHz drops an adjacent
// channel centred at ±12.5 kHz deep into the 81-tap front end's stopband while
// the wanted H-DQPSK (occupied ≈ ±3.6 kHz at 6000 baud, α=0.2) passes flat.
const p25p2ChannelSelectHz = 6250.0

// newP25P2VoiceFrontEnd builds the channel-select + decimation front end for
// the P25 Phase 2 voice chain. Mirrors newP25P1VoiceFrontEnd: on the
// dedicated-tuner path (iqHz well above the intermediate rate) the decimating
// FIR already band-limits as it decimates; on the pass-through path (a wideband
// DDC tap already at the intermediate rate, decim==1) the old front end applied
// NO filter at all, so this channel-selects to ±min(bw, p25p2ChannelSelectHz)
// before the receiver. Factored out so the selectivity is unit-testable.
func newP25P2VoiceFrontEnd(iqHz float64, bw uint32) *decimatingFIR {
	return newVoiceFrontEnd(iqHz, bw, p25p2VoiceIntermediateHz, p25p2ChannelSelectHz)
}

// p25p2VoiceGardnerGain is the Gardner loop step for the H-DQPSK symbol
// clock. This was 0.005 — a sixth of the receiver default — on the reasoning
// that "H-DQPSK slips" at higher gains. That slip was the loop's inverted
// feedback sign running away from the eye, and a smaller gain only ran away
// more slowly (internal/dsp/sync/gardner.go). With the sign corrected the
// loop is stable at every gain, and slower is simply worse: on an
// eight-capture corpus scored against SDRtrunk, confirmed MAC PDUs go
// 228 → 234 (92% → 94% of the reference) from 0.005 to 0.03, with a flat
// plateau from 0.03 to 0.10. 0.03 is the receiver default; the override is
// kept explicit so the value is visible where the receiver is built.
const p25p2VoiceGardnerGain = 0.03

// burstHistLen sizes the per-call burst-type histogram in the chain census.
// The DUID is a 4-bit field, so all sixteen values need a bucket — an array
// sized to the old SlotType range silently dropped FACCH-U (15), the very
// burst type an unscrambled fixture produces.
const burstHistLen = 16

// slotHistLen sizes the per-call SlotType histogram in the chain census.
// It spans SlotTypeUnknown (0x0) .. SlotTypeMACEndCont (0x9); an
// out-of-range slot value (a future enum addition) is dropped by the
// bounds guard at the increment site rather than panicking.
const slotHistLen = 0xA

// runP25Phase2VoiceChain consumes IQ for one P25 Phase 2 voice call. It
// decimates the wideband IQ to an H-DQPSK-friendly rate, recovers the
// dibit stream with the Phase 2 receiver, assembles 360 ms superframes,
// and for every voice-bearing sub-frame FEC-decodes its AMBE+2 frames
// and appends them to the recorder's .raw sidecar.
//
// In parallel it dispatches the MAC-typed sub-frames that ride the same
// superframe (Phase 2 voice traffic channels interleave MAC PDUs —
// talker alias, encryption sync, in-call signalling — with voice).
// macCfg pins the FEC pipeline DecodeSuperframeMACPDUs runs on those
// sub-frames; on a completed talker-alias the chain publishes a
// trunking.TalkerAlias event itself, mirroring the CC's
// publishTalkerAlias path. This is the only way display names surface
// on Phase 2 systems whose CCs never emit alias fragments (e.g. MMR).
//
// The recorder maps protocol "p25-phase2" to the pure-Go AMBE+2
// vocoder (voice.DefaultVocoderForProtocol), so WriteRawFrame here
// decodes each 7-byte frame to PCM and into the call's WAV — unlike the
// DMR chain, whose pre-FEC frames the vocoder cannot consume.
func (c *Composer) runP25Phase2VoiceChain(ctx context.Context, serial string, system string, macCfg p25p2.MACDecodeConfig, iqCh <-chan []complex64, iqHz float64, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-p25p2:"+serial, nil)

	// Shared boundary controller: universal hangtime end-of-call + Touch
	// heartbeat. Talkgroup gating is left disabled (grantTG 0) until the
	// Phase 2 chain surfaces a per-voice-frame MAC talkgroup.
	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	// Issue #376 field-diagnostic: the existing "composer: p25p2 mac
	// pdu" log fires only after a successful FEC decode, so every
	// failure mode (zero macCfg, ScramblerOff against scrambled
	// on-air traffic, ISCH FEC corruption) is silent. A single
	// once-per-call line at chain entry confirms the chain actually
	// ran and surfaces the exact FEC config the grant carried.
	c.log.Info("composer: p25p2 voice chain started",
		"serial", serial, "system", system,
		"trellis", macCfg.Trellis, "rs", macCfg.RS,
		"interleave", macCfg.Interleave, "scrambler", macCfg.Scrambler,
		"seed", macCfg.Seed)
	// Surface the specific reason live MAC PDU decode would fail so the
	// operator knows which knob to turn, rather than one opaque catch-all
	// (issue #451). With the live-decode defaults (ScramblerOn + an
	// identity-derived seed) none of these fire.
	switch {
	case macCfg.Trellis == p25p2.TrellisOff:
		c.log.Warn("composer: p25p2 trellis is off; live P25 Phase 2 MAC PDUs are trellis-encoded (TIA-102.AABF), so voice/MAC decode will fail — TrellisOff is only for pre-stripped fixtures (set p25_phase2_trellis_mode=on)",
			"serial", serial, "system", system,
			"trellis", macCfg.Trellis, "rs", macCfg.RS,
			"interleave", macCfg.Interleave, "scrambler", macCfg.Scrambler)
	case macCfg.Seed == 0:
		c.log.Warn("composer: p25p2 macCfg has no scrambler seed; live MAC PDU decode will fail until the system identity is known (set WACN/SystemID/site, or wait for a Network Status Broadcast on Phase 2 control channels)",
			"serial", serial, "system", system,
			"scrambler", macCfg.Scrambler, "seed", macCfg.Seed)
	case macCfg.Scrambler == p25p2.ScramblerOff:
		c.log.Warn("composer: p25p2 scrambler is off; live P25 Phase 2 MAC PDUs are always PN44-scrambled, so MAC decode will fail (set p25_phase2_scrambler_mode=on)",
			"serial", serial, "system", system,
			"scrambler", macCfg.Scrambler, "seed", macCfg.Seed)
	case macCfg.Scrambler == p25p2.ScramblerProbe && macCfg.RS != p25p2.RSOn:
		c.log.Warn("composer: p25p2 scrambler=probe requires p25_phase2_rs_mode=on to verify the slot offset; it will degrade to offset 0 and likely fail",
			"serial", serial, "system", system,
			"scrambler", macCfg.Scrambler, "rs", macCfg.RS, "seed", macCfg.Seed)
	}

	// Front-end decimator: an 81-tap anti-alias FIR that convolves ONLY at
	// the output positions, replacing the old full-rate FIR + every-Nth-
	// sample decimation that filtered every input sample and discarded ~98%
	// of the result. At 2.4 MS/s that wasted ~194M MACs/sec per voice call
	// and starved the live IQ consumer until the SDR dropped chunks. Same
	// coefficients and same kept samples as before, so the decode is
	// byte-for-byte unchanged — only the wasted work is removed. decim==1 (a
	// source already at the intermediate rate — a wideband DDC voice tap) is NOT
	// a pass-through: it still channel-selects to a single P25 channel before the
	// receiver (see newP25P2VoiceFrontEnd / p25p2ChannelSelectHz) so an adjacent
	// ±12.5 kHz channel can't pump the AGC / degrade carrier recovery.
	fe := newP25P2VoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()

	rs, _ := c.sink.(rawFrameSink)
	sfDec := p25p2.NewSuperframeDecoder()
	// The receiver takes its coarse carrier estimate once and never retakes
	// it, so a channel that drifts past the Costas loop's ±750 Hz pull-in
	// stays lost for the rest of the call. carrierWD notices that no
	// superframe has locked for a superframe's worth of dibits and asks for a
	// re-seed; the reset itself happens between IQ chunks below, never from
	// inside the receiver's own sink. Measured on a real Phase 2 traffic
	// channel this is worth 7 → 15 distinct MAC PDUs over a 6.5 s call
	// (issue #915).
	carrierWD := p25p2.NewCarrierWatchdog(0)
	reacquire := false
	var carrierReseeds atomic.Uint64
	// dispatcher decodes the MAC PDUs interleaved with voice on the
	// traffic channel (talker alias, in-call source, encryption sync).
	// It is the same shared dispatch the signalling follower runs off the
	// traffic-channel signalling stream (internal/sigfollow), so the
	// voice and follower paths never diverge on what they decode (#376).
	// The voice chain wires the call-bound source / encryption PDUs to
	// its engine-backfill publishers; the alias the dispatcher publishes
	// itself (identical in both paths).
	dispatcher := sigfollow.NewMACDispatcher(sigfollow.MACDispatcherOptions{
		Bus:              c.bus,
		Log:              c.log,
		LogPrefix:        "composer",
		System:           system,
		Serial:           serial,
		OnCallSource:     func(u p25p2.GroupVoiceChannelUser) { c.publishP25Phase2CallSource(serial, u) },
		OnCallEncryption: func(es p25p2.EncryptionSync) { c.publishP25Phase2CallEncryption(serial, es) },
	})
	// voiceSubframes counts P25 Phase 2 voice-bearing subframes the
	// receiver delivered — i.e. real voice activity. The touch ticker
	// (below) only refreshes the engine's LastHeardAt when this counter
	// has advanced since the previous tick. Without this gate a stalled
	// decoder still kept the call alive forever via an unconditional
	// 1 s heartbeat (issue #356).
	var voiceSubframes atomic.Uint64
	// Decode-quality telemetry — see runP25Phase1VoiceChain for the
	// rationale. A high uncorrectable rate is the measurable signature of
	// weak signal / wrong gain behind garbled audio (issue #356 follow-up).
	var (
		uncorrectableSubframes atomic.Uint64
		corrErrBits            atomic.Uint64
	)
	// MAC-pipeline census (issue #813). A field test on real Phase 2
	// traffic saw the per-PDU "composer: p25p2 mac pdu" line — which fires
	// only on a *successful* MAC decode — never appear at all, which is
	// ambiguous: it can't tell superframe-sync failure from ISCH
	// mis-classification from MAC-FEC failure. These counters feed the
	// unconditional end-of-call census (logCallCensus) that disambiguates
	// the three. They are read from the for-select goroutine below, so they
	// are atomic like the decode-quality counters above.
	var (
		superframesSeen  atomic.Uint64
		macSubframesSeen atomic.Uint64
		macPDUsDecoded   atomic.Uint64
		macRSValid       atomic.Uint64
		slotHist         [burstHistLen]atomic.Uint64
	)
	// drainSuperframes runs the shared per-superframe voice + MAC census
	// body. It is driven by the hard DibitSink or, when the grant requests
	// soft-decision demod (issue #915), the soft SoftSink whose superframes
	// carry per-symbol soft differentials in Subframe.Soft that the MAC
	// decode path picks up automatically. The census body is identical
	// either way, so both paths stay in lock-step.
	// One superframe of PN44 for the channel identity, built once: both the
	// MAC and the voice path descramble against it.
	scrambleSeq := p25p2.ScrambleSequence(macCfg.Seed)
	drainSuperframes := func(sfs []p25p2.Superframe) {
		for _, sf := range sfs {
			superframesSeen.Add(1)
			// Which S-ISCH slot this superframe anchored on, resolved by vote
			// over its ACCH bursts. Voice bursts carry no signalling of their
			// own to resolve it from, so they inherit it.
			slotOffset, slotScore := p25p2.ResolveSuperframeSlotOffset(sf, scrambleSeq)
			for _, sub := range sf.Subframes {
				// Census every sub-frame by the burst type its DUID names,
				// before the voice gate, so the end-of-call census shows
				// what the channel actually carried.
				//
				// This used to census the ISCH SlotType. That field comes
				// from this package's own ISCH model, which does not match
				// the air (issue #915): the histogram read as
				// slot_Unknown for every sub-frame of a channel that was
				// decoding MAC PDUs perfectly well. The DUID is the field
				// both reference decoders dispatch on, and it is protected
				// by its own (8,4) code.
				burst := p25p2.BurstTypeOf(sub.Dibits)
				if int(burst) < burstHistLen {
					slotHist[burst].Add(1)
				}
				if burst.IsACCH() {
					macSubframesSeen.Add(1)
				}
				if !burst.IsVoice() {
					continue
				}
				voiceSubframes.Add(1)
				bt.onVoice(0)
				if rs == nil || slotScore <= 0 {
					continue
				}
				// A voice burst's payload is PN44-scrambled at the same slot
				// phase as an ACCH burst's, so it needs the offset the MAC
				// bursts resolved for this superframe.
				frames, errBits, unc, err := p25p2.ExtractBurstVoiceFrames(
					sub.Dibits, (sub.Index+slotOffset)%p25p2.SubframesPerSuperframe, scrambleSeq)
				if errBits > 0 {
					corrErrBits.Add(uint64(errBits))
				}
				if unc > 0 {
					uncorrectableSubframes.Add(uint64(unc))
				}
				if err != nil {
					uncorrectableSubframes.Add(1)
					c.log.Debug("composer: p25p2 voice extract uncorrectable subframe",
						"serial", serial, "err", err)
				}
				for _, f := range frames {
					if f == nil {
						continue
					}
					if werr := rs.WriteRawFrame(serial, f); werr != nil {
						c.log.Warn("composer: p25p2 raw-frame write failed",
							"serial", serial, "err", werr)
					}
				}
			}
			// The talker alias, in-call source, and encryption-sync
			// MAC PDUs that interleave with voice are decoded by the
			// shared dispatcher (same path the signalling follower
			// runs off the traffic channel, #376). rsValid counts how
			// many carried a clean outer RS(24,16,9) parity — the
			// framing-health signal the end-of-call census reports
			// (issue #915).
			nDec, nRS := dispatcher.Dispatch(sf, macCfg)
			macPDUsDecoded.Add(uint64(nDec))
			macRSValid.Add(uint64(nRS))
		}
	}

	// Build the receiver on the hard slicer by default; when the grant
	// carries the soft-decision flag (issue #915) wire the soft sink so the
	// receiver emits per-symbol soft differentials and the SuperframeDecoder
	// carries them into Subframe.Soft via ProcessSoft.
	rxOpts := p25p2rx.Options{
		SampleRateHz:  symbolHz,
		ClockMode:     p25p2rx.ClockGardner,
		GardnerGain:   p25p2VoiceGardnerGain,
		Equalizer:     macCfg.Equalizer,
		EnableDCBlock: macCfg.DCBlock,
	}
	if macCfg.SoftDecision {
		rxOpts.SoftDecision = true
		rxOpts.SoftSink = func(dibits []uint8, soft []complex64, baseIdx int) {
			sfs := sfDec.ProcessSoft(dibits, soft, baseIdx)
			if carrierWD.Observe(len(dibits), len(sfs)) {
				reacquire = true
			}
			drainSuperframes(sfs)
		}
	} else {
		rxOpts.DibitSink = func(dibits []uint8, baseIdx int) {
			sfs := sfDec.Process(dibits, baseIdx)
			if carrierWD.Observe(len(dibits), len(sfs)) {
				reacquire = true
			}
			drainSuperframes(sfs)
		}
	}
	rx := p25p2rx.New(rxOpts)

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	// logDecodeQuality emits a rolling decode-quality summary, gated to a
	// burst of voice subframes so it does not spam the log every touch
	// tick (issue #356 follow-up). See runP25Phase1VoiceChain.
	var lastQualityLogSubframes uint64
	const qualityLogEverySubframes = 50
	logDecodeQuality := func(final bool) {
		n := voiceSubframes.Load()
		if n == 0 || (!final && n-lastQualityLogSubframes < qualityLogEverySubframes) {
			return
		}
		lastQualityLogSubframes = n
		c.log.Info("composer: p25p2 decode quality",
			"serial", serial, "system", system,
			"voice_subframes", n, "uncorrectable_subframes", uncorrectableSubframes.Load(),
			"corrected_bit_errs", corrErrBits.Load())
	}
	// logCallCensus emits exactly one line at chain end, unconditionally —
	// even when zero superframes ever locked. It is the field diagnostic
	// that disambiguates the three #813 failure modes when no "composer:
	// p25p2 mac pdu" line ever appears:
	//   - superframes=0                  → superframe sync never locked
	//   - superframes>0, mac_subframes=0 → ISCH never classified a MAC slot
	//                                       (inspect the slot_* histogram)
	//   - mac_subframes>0, mac_pdus=0    → MAC FEC chain failed every slot
	//   - mac_pdus>0, mac_rs_valid=0     → the MAC window parses but never
	//                                       satisfies the outer RS(24,16,9)
	//                                       parity: a mis-framed / mis-
	//                                       descrambled superframe decoding
	//                                       random bytes, not real signalling
	//                                       (issue #915 — this is why the
	//                                       clear-MAC source RID never lands)
	// The slot_<type> buckets reuse SlotType.String() so the line is
	// self-describing.
	logCallCensus := func() {
		attrs := []any{
			"serial", serial, "system", system,
			"superframes", superframesSeen.Load(),
			"voice_subframes", voiceSubframes.Load(),
			"mac_subframes", macSubframesSeen.Load(),
			"mac_pdus", macPDUsDecoded.Load(),
			"mac_rs_valid", macRSValid.Load(),
			// carrier_reseeds > 0 means the receiver lost its carrier
			// mid-call and the watchdog restarted acquisition. A steady
			// count on a channel that still decodes is normal; a high count
			// with superframes near zero says acquisition never took.
			"carrier_reseeds", carrierReseeds.Load(),
		}
		for bt := 0; bt < burstHistLen; bt++ {
			if cnt := slotHist[bt].Load(); cnt > 0 {
				attrs = append(attrs, "burst_"+p25p2.BurstType(bt).String(), cnt)
			}
		}
		c.log.Info("composer: p25p2 call census", attrs...)
	}

	for {
		select {
		case <-ctx.Done():
			logDecodeQuality(true)
			logCallCensus()
			return
		case <-touchTicker.C:
			// Touch + hangtime end-of-call handled by the shared boundary
			// tracker; this ticker only drives the decode-quality summary.
			logDecodeQuality(false)
		case iq, ok := <-iqCh:
			if !ok {
				logDecodeQuality(true)
				logCallCensus()
				return
			}
			bt.observe(iq)
			rx.Process(fe.Process(nil, iq))
			if reacquire {
				// Between chunks: Reset clears the receiver's one-shot
				// carrier seed so the next samples re-estimate it, and the
				// superframe decoder drops anchors that belong to the old
				// carrier. Doing this inside the sink would re-enter a
				// receiver that is mid-Process.
				rx.Reset()
				sfDec.Reset()
				carrierWD.Reset()
				reacquire = false
				carrierReseeds.Add(1)
			}
		}
	}
}

// publishP25Phase2CallSource publishes a KindCallSourceUpdate event so
// the trunking engine can backfill the bound ActiveCall's SourceID +
// Encrypted from an in-call GROUP_VOICE_CHANNEL_USER PDU. The engine
// fills in System / Protocol / GroupID from the bound Grant on
// republish — leave them blank here.
func (c *Composer) publishP25Phase2CallSource(serial string, u p25p2.GroupVoiceChannelUser) {
	if c.bus == nil {
		return
	}
	so := p25p2.ServiceOptions(u.ServiceOptions)
	c.bus.Publish(events.Event{
		Kind: events.KindCallSourceUpdate,
		Payload: trunking.CallSourceUpdate{
			DeviceSerial: serial,
			SourceID:     u.SourceID,
			Encrypted:    so.Encrypted(),
			Emergency:    so.Emergency(),
			Priority:     so.Priority(),
			At:           time.Now().UTC(),
		},
	})
}

// publishP25Phase2CallEncryption mirrors the Phase 1 LDU2 in-call
// encryption-sync path for Phase 2 traffic-channel EncryptionSync MAC
// PDUs. The engine backfills ALGID/KID onto the bound ActiveCall's
// Grant and republishes with the call's identity.
func (c *Composer) publishP25Phase2CallEncryption(serial string, es p25p2.EncryptionSync) {
	if c.bus == nil {
		return
	}
	// Validity gate: a mis-decoded MAC Encryption Sync yields an Algorithm
	// ID outside the TIA-102 set (the field-observed smear across 0x00-0xFF,
	// one Key ID per call). Drop it rather than surface a plausible-but-wrong
	// algid+key that downstream can't tell from a real key. Issue #924.
	if !p25.AlgorithmKnown(es.AlgorithmID) {
		c.log.Debug("composer: p25p2 dropping out-of-set encryption sync",
			"serial", serial, "alg", p25.FormatAlgorithm(es.AlgorithmID), "key", es.KeyID)
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindCallEncryption,
		Payload: trunking.CallEncryption{
			DeviceSerial:     serial,
			AlgorithmID:      es.AlgorithmID,
			KeyID:            es.KeyID,
			MessageIndicator: es.MessageIndicator,
			At:               time.Now().UTC(),
		},
	})
}
