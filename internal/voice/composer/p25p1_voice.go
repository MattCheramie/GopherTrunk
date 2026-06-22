package composer

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25p1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25p1VoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the P25 Phase 1 receiver runs. 48 kHz gives the 4800-baud
// C4FM symbol stream 10 samples per symbol — ample for the receiver's
// RRC matched filter and Mueller-Müller clock recovery.
const p25p1VoiceIntermediateHz = 48_000

// p25p1DeviationHz is the C4FM peak frequency deviation at symbol ±3
// per TIA-102.BAAA. It calibrates the receiver's 4-level slicer.
const p25p1DeviationHz = 1800.0

// resolveP25Phase1DemodMode parses the system-level
// p25_phase1_demod_mode string carried on the grant into a receiver
// mode. Unknown values warn-log and fall back to C4FM so a typo doesn't
// silently kill a previously-working system; empty is the canonical
// default. Factored out so the wiring is unit-testable independently
// of the full IQ → LDU pipeline (issue #356 follow-up).
func (c *Composer) resolveP25Phase1DemodMode(serial, mode string) p25p1rx.DemodMode {
	parsed, ok := p25p1rx.ParseDemodMode(mode)
	if !ok {
		c.log.Warn("composer: unrecognised p25_phase1_demod_mode; voice chain falling back to c4fm",
			"serial", serial, "value", mode)
	}
	return parsed
}

// runP25Phase1VoiceChain consumes IQ for one P25 Phase 1 voice call. It
// decimates the wideband IQ to a C4FM-friendly rate, recovers the dibit
// stream with the Phase 1 receiver, assembles complete 1728-bit LDUs,
// and for each LDU extracts its 9 IMBE voice frames and appends them to
// the recorder's .raw sidecar.
//
// demodMode is the raw system-level p25_phase1_demod_mode string from
// the grant ("c4fm" / "cqpsk" / ""). An empty or unrecognised value
// preserves the legacy C4FM path; "cqpsk" / "lsm" / "linear" routes
// voice IQ through the linear-CQPSK path required for LSM simulcast
// sites. Without this the voice chain was hardcoded to C4FM regardless
// of the system setting and never decoded LDUs on simulcast sites
// (issue #356 follow-up).
//
// The recorder maps protocol "p25" to the pure-Go IMBE vocoder
// (voice.DefaultVocoderForProtocol), so WriteRawFrame here decodes each
// 11-byte frame to PCM and into the call's WAV.
func (c *Composer) runP25Phase1VoiceChain(ctx context.Context, serial, system string, iqCh <-chan []complex64, iqHz float64, demodMode string, grantTG uint32, callID uint64, patched []uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-p25p1:"+serial, nil)

	// Shared boundary controller: universal hangtime end-of-call,
	// talkgroup gating (so a different talkgroup on a shared voice
	// frequency is never appended), and per-transmission file splitting.
	bt := c.newBoundaryTracker(serial, grantTG, patched)
	go bt.run(ctx)

	decim := int(math.Round(iqHz)) / p25p1VoiceIntermediateHz
	if decim < 1 {
		decim = 1
	}
	symbolHz := iqHz / float64(decim)

	mode := c.resolveP25Phase1DemodMode(serial, demodMode)

	// Autotune: pre-rotate the voice IQ by this dongle's running-average
	// carrier-error correction so the receiver's AFC starts near lock, then
	// fold the residual back into the average at end-of-call. atManager is
	// nil when sdr.autotune is off (or on the CQPSK path the average stays
	// 0), in which case atNCO stays nil and the loop below is byte-exact
	// with the pre-autotune behaviour. The correction is captured once at
	// call start; the +offset convention matches the control decoder's DDC.
	atManager := c.autotune.Get(serial)
	var (
		atNCO     *dsp.NCO
		atApplied int
	)
	if atManager != nil {
		// Correction() is 0 until the running average has warmed up, so the
		// first few calls on a fresh dongle apply nothing rather than chasing
		// a single noisy measurement.
		atApplied = atManager.Correction()
		if atApplied != 0 {
			atNCO = dsp.NewNCO(float64(atApplied), symbolHz)
		}
	}

	// Surface the resolved demod mode + sample rates once per call so an
	// operator can confirm from logs that voice grants are running the
	// same demod mode the CC pipeline already prints at startup
	// (ccdecoder: p25/phase1 pipeline configured demod=…). Issue #356
	// follow-up: a field log showed the CC locked on cqpsk but gave no
	// way to verify the voice chain wasn't silently still on c4fm.
	// Mirrors the Phase 2 chain's "composer: p25p2 voice chain started"
	// line.
	c.log.Info("composer: p25p1 voice chain started",
		"serial", serial, "demod_mode", mode,
		"iq_rate_hz", iqHz, "symbol_rate_hz", symbolHz)

	// Front-end LPF: doubles as the anti-aliasing filter for the
	// decimation, so it is only needed when the IQ is actually
	// decimated (decim == 1 only in tests that feed IQ already at the
	// intermediate rate).
	cutoff := float64(c.bw) / iqHz
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	lpf := filter.NewFIR(filter.LowpassKaiser(81, cutoff, 8.6))

	rs, _ := c.sink.(rawFrameSink)
	// ers, when the sink supports it, carries the per-frame FEC corrected-bit
	// count to the IMBE decoder's adaptive smoothing (P25 Phase 1 only).
	ers, _ := c.sink.(errAwareRawSink)
	// cas, when the sink supports it, additionally carries this call's
	// CallID so the recorder fences stale frames from a previous call on a
	// reused tap serial (voice-tap mix-up). Preferred over ers when present.
	cas, _ := c.sink.(callAwareRawSink)
	// lastES is the last Encryption Sync this chain published on the
	// bus. The voice frame stream carries one ES per LDU2 (every ~180
	// ms), but ALGID/KID rarely change inside a single call — gate the
	// publish so we don't fan out the same event a dozen times.
	var (
		lastES    phase1.EncryptionSync
		hasLastES bool
	)
	// aliasBuf reassembles the per-call Motorola voice-channel
	// talker alias (LCO 0x15 header + N × LCO 0x17 data blocks).
	// Distinct from the Motorola vendor TSBK form on the control
	// channel, which phase1.TalkerAliasAssembler still handles. One
	// buffer per voice chain — each chain handles exactly one call's
	// worth of LDU1s.
	aliasBuf := phase1.NewMotorolaTalkerAliasBuf(nil)
	// lastSourceID is the most recently observed SourceID from an
	// LCO-0 (group voice channel user) LC on this chain. The alias
	// LCs (0x15/0x16/0x17) carry only payload bytes — no SourceID —
	// so the voice channel's own LCO-0 LCs are how we tag a decoded
	// alias with the right radio.
	var lastSourceID uint32
	// publishMotorolaAlias surfaces a completed voice-channel talker
	// alias on the bus (the affiliation tracker binds it onto the RID).
	// Shared by the LDU1 link-control path and the TDULC terminator
	// path — Motorola systems carry these 0x15/0x17 LCs on either
	// (#376), so both feed the same buffer and publish identically.
	publishMotorolaAlias := func(alias string, reliable bool) {
		if lastSourceID == 0 || alias == "" {
			return
		}
		c.log.Info("composer: p25p1 motorola talker alias",
			"system", system, "serial", serial, "src", lastSourceID, "alias", alias,
			"unreliable", !reliable)
		c.bus.Publish(events.Event{
			Kind: events.KindTalkerAlias,
			Payload: trunking.TalkerAlias{
				System:     system,
				Protocol:   "p25-phase1",
				SourceID:   lastSourceID,
				Alias:      alias,
				Unreliable: !reliable,
				At:         time.Now(),
			},
		})
	}
	// frames counts LDUs delivered by the receiver — i.e. real voice
	// activity. The touch ticker (below) only refreshes the engine's
	// LastHeardAt when this counter has advanced since the previous
	// tick. Without this gate, a stalled decoder still kept the call
	// alive forever via an unconditional 1 s heartbeat (issue #356).
	var frames atomic.Uint64
	// Decode-quality telemetry. An LDU whose IMBE FEC could not fully
	// correct a subframe still yields (degraded) audio, so a high
	// uncorrectable rate is the measurable signature of weak signal /
	// wrong gain — exactly the "multiple uncorrectable errors" an
	// operator sees when audio is garbled. Surfacing a rolling summary
	// lets them watch the rate fall as they raise gain (issue #356
	// follow-up). corrErrBits sums the bit errors the FEC *did* correct.
	var (
		uncorrectableLDUs atomic.Uint64
		corrErrBits       atomic.Uint64
	)
	// Outer-RS telemetry for the Link Control (LDU1) and Encryption Sync
	// (LDU2) words. A rising RS-uncorrectable rate is the measurable
	// signature of marginal signal at the FEC layer that drives the
	// talkgroup-gating decision — high values mean talkgroups can't be
	// trusted and the operator should improve gain/antenna.
	var (
		lcRSUncorrectable  atomic.Uint64
		essRSUncorrectable atomic.Uint64
	)
	rx := p25p1rx.New(p25p1rx.Options{
		SampleRateHz: symbolHz,
		DeviationHz:  p25p1DeviationHz,
		DemodMode:    mode,
		Sink: func(ldu []byte) {
			duid, derr := phase1.LDUDuid(ldu)
			// A Terminator Data Unit (with or without LC) marks the end
			// of a transmission (talker released PTT). In split mode this
			// rolls the recording to a fresh file for the next over; it
			// is not voice, so don't write or count it.
			if derr == nil && (duid == phase1.DUIDTerminator || duid == phase1.DUIDTerminatorWithLC) {
				// A TDULC terminator still carries link control, and on
				// Motorola systems the talker alias (LCO 0x15 header /
				// 0x17 data) rides here rather than in LDU1 (#376). Pull
				// its LC before ending the call so the alias isn't lost
				// on the terminator.
				if duid == phase1.DUIDTerminatorWithLC {
					if lcf, content, _, terr := phase1.ExtractTDULCLinkControl(ldu); terr == nil {
						switch {
						case phase1.IsTalkerAliasLCO(lcf):
							if alias, ok, reliable := aliasBuf.AddFragment(lcf, content); ok {
								publishMotorolaAlias(alias, reliable)
							}
						case lcf == phase1.LCOGroupVoiceChannelUser:
							// A terminator LCO-0 can carry the source ID;
							// keep lastSourceID current so a just-completed
							// alias tags the right radio.
							src := uint32(content[6])<<16 | uint32(content[7])<<8 | uint32(content[8])
							if src != 0 {
								lastSourceID = src
							}
						}
					}
				}
				bt.onTransmissionEnd()
				return
			}
			// Count LDU delivery for the decode-quality telemetry.
			frames.Add(1)

			var blocks [phase1.LDULCESBlockCount][]byte
			haveBlocks := false
			if derr == nil {
				if b, berr := phase1.ExtractLCESBlocks(ldu); berr == nil {
					blocks = b
					haveBlocks = true
				}
			}
			// Talkgroup gating: LDU1 carries the talkgroup in its Link
			// Control; LDU2 (and any LDU whose LC didn't decode) passes 0
			// so the tracker inherits the last decision. The tracker
			// returns false when the talkgroup doesn't match the grant —
			// i.e. another talkgroup has taken this shared frequency —
			// and we drop that audio rather than append it.
			tg := uint32(0)
			if duid == phase1.DUIDLogicalLink1 && haveBlocks {
				lc, _, lerr := phase1.ParseLinkControl(blocks)
				switch {
				case errors.Is(lerr, phase1.ErrLinkControlUncorrectable):
					// The outer RS layer could not recover the LC, so its
					// talkgroup is untrustworthy. Leave tg=0 so the boundary
					// tracker INHERITS the last match instead of gating audio
					// (or ending the call) on a garbage talkgroup — the
					// dominant cause of dropped/fragmented recordings in the
					// field capture.
					lcRSUncorrectable.Add(1)
				case lerr == nil && lc.LCFormat == phase1.LCOGroupVoiceChannelUser:
					tg = uint32(lc.TalkgroupID)
				}
			}
			write := bt.onVoice(tg)

			if write && rs != nil {
				fs, frameErrs, errBits, err := phase1.ExtractVoiceFramesDetailed(ldu)
				if errBits > 0 {
					corrErrBits.Add(uint64(errBits))
				}
				if err != nil {
					uncorrectableLDUs.Add(1)
					c.log.Debug("composer: p25p1 voice extract uncorrectable subframe",
						"serial", serial, "err", err)
				}
				for i, f := range fs {
					if f == nil {
						continue
					}
					// Hand the per-frame corrected-bit count to the IMBE
					// decoder (via the recorder) so its adaptive smoothing can
					// track the channel error rate; the call-aware path also
					// fences stale frames on a reused serial. Fall back through
					// errAwareRawSink then the plain write for sinks that don't
					// support the richer shapes.
					var werr error
					switch {
					case cas != nil:
						werr = cas.WriteRawFrameForCall(serial, callID, f, frameErrs[i])
					case ers != nil:
						werr = ers.WriteRawFrameWithErrors(serial, f, frameErrs[i])
					default:
						werr = rs.WriteRawFrame(serial, f)
					}
					if werr != nil {
						c.log.Warn("composer: p25p1 raw-frame write failed",
							"serial", serial, "err", werr)
					}
				}
			}

			// Call metadata (talker alias, source ID, encryption sync) is
			// surfaced regardless of the audio gating decision.
			if !haveBlocks {
				return
			}
			switch duid {
			case phase1.DUIDLogicalLink1:
				// Motorola voice-channel talker-alias LCs reuse the
				// LC octets for the alias payload, so dispatch on
				// the opcode byte from ParseLinkControlContent —
				// ParseLinkControl's TG/SRC view is only meaningful
				// for LCOGroupVoiceChannelUser (0x00).
				content, _, cerr := phase1.ParseLinkControlContent(blocks)
				if cerr != nil {
					return
				}
				lcf := content[0]
				switch {
				case phase1.IsTalkerAliasLCO(lcf):
					if alias, ok, reliable := aliasBuf.AddFragment(lcf, content); ok {
						publishMotorolaAlias(alias, reliable)
					}
				default:
					if lc, _, lerr := phase1.ParseLinkControl(blocks); lerr == nil {
						c.log.Debug("composer: p25p1 link control",
							"serial", serial, "lcf", lc.LCFormat,
							"tg", lc.TalkgroupID, "src", lc.SourceID)
						if lc.LCFormat == phase1.LCOGroupVoiceChannelUser && lc.SourceID != 0 {
							lastSourceID = lc.SourceID
						}
					}
				}
			case phase1.DUIDLogicalLink2:
				es, _, lerr := phase1.ParseEncryptionSync(blocks)
				if errors.Is(lerr, phase1.ErrEncryptionSyncUncorrectable) {
					// Outer RS could not recover the ES; do not surface a
					// garbage algorithm/key as a real encryption change.
					essRSUncorrectable.Add(1)
				}
				if lerr == nil && es.Encrypted() {
					c.log.Debug("composer: p25p1 encryption sync",
						"serial", serial, "alg", es.AlgorithmID, "key", es.KeyID)
					// Publish on the bus so the trunking engine
					// backfills the active call's Grant. Skip when
					// neither ALGID nor KID has changed since the
					// last publish.
					if !hasLastES || es.AlgorithmID != lastES.AlgorithmID || es.KeyID != lastES.KeyID || es.MessageIndicator != lastES.MessageIndicator {
						c.bus.Publish(events.Event{
							Kind: events.KindCallEncryption,
							Payload: trunking.CallEncryption{
								DeviceSerial:     serial,
								AlgorithmID:      es.AlgorithmID,
								KeyID:            es.KeyID,
								MessageIndicator: es.MessageIndicator,
								At:               time.Now(),
							},
						})
						lastES = es
						hasLastES = true
					}
				}
			}
		},
	})

	// At end-of-call fold this call's residual carrier offset into the
	// dongle's running average — but only from a call that decoded *cleanly*.
	// The AFC residual is only a meaningful carrier-error estimate when the
	// loop actually locked and tracked; a short or error-ridden call yields a
	// data-dependent snapshot that, folded in, drags the correction toward the
	// kHz. So require a minimum run of LDUs and a low uncorrectable rate before
	// trusting the measurement. The residual (AFCOffsetHz) plus what we already
	// applied (atApplied) is the dongle's true error; the next call on this
	// serial picks up the updated correction. centerFreq is 0 here (the
	// chain isn't handed the grant frequency), so the PPM sanity warning is
	// the control decoder's job; the average + suggested value still log.
	if atManager != nil {
		defer func() {
			n := frames.Load()
			bad := uncorrectableLDUs.Load()
			// minAutotuneLDUs ~= 1 s of voice (9 frames/LDU, 180 ms/LDU);
			// drop a call with too few LDUs or >10% uncorrectable.
			const minAutotuneLDUs = 5
			if n < minAutotuneLDUs || bad*10 > n {
				c.log.Debug("composer: p25p1 autotune measurement skipped (call too short or noisy)",
					"serial", serial, "ldus", n, "uncorrectable_ldus", bad)
				return
			}
			observed := int(math.Round(rx.AFCOffsetHz()))
			atManager.AddErrorMeasurement(observed, atApplied, 0)
			c.log.Info("composer: p25p1 "+atManager.StatusString(0, 0),
				"serial", serial, "observed_hz", observed, "applied_hz", atApplied)
		}()
	}

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	// logDecodeQuality emits a rolling decode-quality summary. Gated to
	// fire only after a burst of LDUs (~4.5 s of voice at 9 frames /
	// 180 ms per LDU) so it does not spam the log every touch tick.
	var lastQualityLogLDUs uint64
	const qualityLogEveryLDUs = 25
	logDecodeQuality := func(final bool) {
		n := frames.Load()
		if n == 0 || (!final && n-lastQualityLogLDUs < qualityLogEveryLDUs) {
			return
		}
		lastQualityLogLDUs = n
		c.log.Info("composer: p25p1 decode quality",
			"serial", serial, "demod_mode", mode,
			"ldus", n, "uncorrectable_ldus", uncorrectableLDUs.Load(),
			"corrected_bit_errs", corrErrBits.Load(),
			"lc_rs_uncorrectable", lcRSUncorrectable.Load(),
			"ess_rs_uncorrectable", essRSUncorrectable.Load())
	}

	for {
		select {
		case <-ctx.Done():
			logDecodeQuality(true)
			return
		case <-touchTicker.C:
			// Engine Touch + hangtime end-of-call are handled by the
			// shared boundary tracker (bt.run); this ticker only drives
			// the rolling decode-quality summary.
			logDecodeQuality(false)
		case iq, ok := <-iqCh:
			if !ok {
				logDecodeQuality(true)
				return
			}
			samples := iq
			if decim > 1 {
				samples = decimateComplex(lpf.Process(nil, iq), decim)
			}
			if atNCO != nil {
				// Shift the estimated carrier (at +atApplied Hz) down to DC.
				// In-place is safe (NCO.Mix supports dst aliasing src).
				samples = atNCO.Mix(samples, samples)
			}
			rx.Process(samples)
		}
	}
}
