package composer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/events"
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
func (c *Composer) runP25Phase1VoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz uint32, demodMode string, done chan<- struct{}) {
	defer close(done)

	decim := int(iqHz) / p25p1VoiceIntermediateHz
	if decim < 1 {
		decim = 1
	}
	symbolHz := float64(iqHz) / float64(decim)

	mode := c.resolveP25Phase1DemodMode(serial, demodMode)

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
	cutoff := float64(c.bw) / float64(iqHz)
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	lpf := filter.NewFIR(filter.LowpassKaiser(81, cutoff, 8.6))

	rs, _ := c.sink.(rawFrameSink)
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
	rx := p25p1rx.New(p25p1rx.Options{
		SampleRateHz: symbolHz,
		DeviationHz:  p25p1DeviationHz,
		DemodMode:    mode,
		Sink: func(ldu []byte) {
			// Bump first so the watchdog gate accounts for LDU
			// delivery even when there's no raw-frame sink.
			frames.Add(1)
			if rs == nil {
				return
			}
			fs, errBits, err := phase1.ExtractVoiceFrames(ldu)
			if errBits > 0 {
				corrErrBits.Add(uint64(errBits))
			}
			if err != nil {
				uncorrectableLDUs.Add(1)
				c.log.Debug("composer: p25p1 voice extract uncorrectable subframe",
					"serial", serial, "err", err)
			}
			for _, f := range fs {
				if f == nil {
					continue
				}
				if werr := rs.WriteRawFrame(serial, f); werr != nil {
					c.log.Warn("composer: p25p1 raw-frame write failed",
						"serial", serial, "err", werr)
				}
			}
			// LDU1 carries a Link Control word, LDU2 an Encryption
			// Sync — surface the call metadata each identifies.
			duid, derr := phase1.LDUDuid(ldu)
			if derr != nil {
				return
			}
			blocks, berr := phase1.ExtractLCESBlocks(ldu)
			if berr != nil {
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
					if alias, ok := aliasBuf.AddFragment(lcf, content); ok && lastSourceID != 0 {
						c.log.Info("composer: p25p1 motorola talker alias",
							"serial", serial, "src", lastSourceID, "alias", alias)
						c.bus.Publish(events.Event{
							Kind: events.KindTalkerAlias,
							Payload: trunking.TalkerAlias{
								Protocol: "p25-phase1",
								SourceID: lastSourceID,
								Alias:    alias,
								At:       time.Now(),
							},
						})
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
				if es, _, lerr := phase1.ParseEncryptionSync(blocks); lerr == nil && es.Encrypted() {
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

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	var lastFrames uint64
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
			"corrected_bit_errs", corrErrBits.Load())
	}

	for {
		select {
		case <-ctx.Done():
			logDecodeQuality(true)
			return
		case <-touchTicker.C:
			n := frames.Load()
			if n != lastFrames && c.engine != nil {
				c.engine.Touch(serial)
				lastFrames = n
			}
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
			rx.Process(samples)
		}
	}
}
