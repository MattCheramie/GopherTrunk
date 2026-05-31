package composer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	p25p2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25p2VoiceIntermediateHz is the rate the wideband IQ is decimated to
// before the P25 Phase 2 receiver runs. 48 kHz gives the 6000-baud
// H-DQPSK symbol stream 8 samples per symbol — ample for the matched
// filter and Gardner clock recovery without running them at the SDR's
// native multi-MS/s rate.
const p25p2VoiceIntermediateHz = 48_000

// p25p2VoiceGardnerGain matches the value newP25Phase2Pipeline settled
// on for the H-DQPSK symbol clock (smaller than the receiver default
// because H-DQPSK slips differently than C4FM).
const p25p2VoiceGardnerGain = 0.005

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
func (c *Composer) runP25Phase2VoiceChain(ctx context.Context, serial string, system string, macCfg p25p2.MACDecodeConfig, iqCh <-chan []complex64, iqHz uint32, done chan<- struct{}) {
	defer close(done)

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
	if macCfg.Scrambler == p25p2.ScramblerOff || macCfg.Seed == 0 {
		c.log.Warn("composer: p25p2 macCfg suggests live MAC PDU decode will fail",
			"serial", serial, "system", system,
			"scrambler", macCfg.Scrambler, "seed", macCfg.Seed)
	}

	decim := int(iqHz) / p25p2VoiceIntermediateHz
	if decim < 1 {
		decim = 1
	}
	symbolHz := float64(iqHz) / float64(decim)

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
	sfDec := p25p2.NewSuperframeDecoder()
	aliasAsm := p25p2.NewTalkerAliasAssembler(nil)
	// macSeen rate-limits the diagnostic per-PDU log to one line per
	// opcode (+ MFID for vendor opcodes) per call — enough to confirm
	// which transports MMR-style systems actually emit, without
	// drowning the log when an opcode repeats many times per call.
	macSeen := make(map[uint16]struct{})
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
	rx := p25p2rx.New(p25p2rx.Options{
		SampleRateHz: symbolHz,
		ClockMode:    p25p2rx.ClockGardner,
		GardnerGain:  p25p2VoiceGardnerGain,
		DibitSink: func(dibits []uint8, baseIdx int) {
			for _, sf := range sfDec.Process(dibits, baseIdx) {
				for _, sub := range sf.Subframes {
					if !sub.SlotType.IsVoice() {
						continue
					}
					voiceSubframes.Add(1)
					if rs == nil {
						continue
					}
					frames, errBits, err := p25p2.ExtractVoiceFrames(sub)
					if errBits > 0 {
						corrErrBits.Add(uint64(errBits))
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
				for _, pdu := range p25p2.DecodeSuperframeMACPDUs(sf, macCfg) {
					// Prong C diagnostic: log the first PDU seen per
					// (opcode, MFID) on this call. If a real on-air
					// system emits an opcode we don't dispatch
					// (vendor talker-alias variants, an unexpected
					// User PDU encoding, …) the log line tells the
					// next field tester exactly what we saw.
					key := uint16(pdu.Opcode)<<8 | uint16(pdu.MFID)
					if _, seen := macSeen[key]; !seen {
						macSeen[key] = struct{}{}
						c.log.Info("composer: p25p2 mac pdu",
							"system", system, "serial", serial,
							"opcode", pdu.Opcode, "mfid", pdu.MFID,
							"payload_len", len(pdu.Payload))
					}
					if u, ok := pdu.AsGroupVoiceChannelUser(); ok {
						c.publishP25Phase2CallSource(serial, u)
						continue
					}
					if es, ok := pdu.AsEncryptionSync(); ok {
						c.publishP25Phase2CallEncryption(serial, es)
						continue
					}
					if f, ok := pdu.AsTalkerAliasFragment(); ok {
						alias, src, complete := aliasAsm.Add(f)
						if complete {
							c.publishP25Phase2TalkerAlias(system, src, alias)
						}
						continue
					}
				}
			}
		},
	})

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	var lastSubframes uint64
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

	for {
		select {
		case <-ctx.Done():
			logDecodeQuality(true)
			return
		case <-touchTicker.C:
			n := voiceSubframes.Load()
			if n != lastSubframes && c.engine != nil {
				c.engine.Touch(serial)
				lastSubframes = n
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

// publishP25Phase2TalkerAlias mirrors phase2.ControlChannel.publishTalkerAlias
// for the voice-channel MAC dispatch path: a completed alias the
// composer reassembled off the traffic channel surfaces on the bus
// with the same payload shape as one decoded on the CC.
func (c *Composer) publishP25Phase2TalkerAlias(system string, sourceID uint32, alias string) {
	if c.bus == nil {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindTalkerAlias,
		Payload: trunking.TalkerAlias{
			System:   system,
			Protocol: "p25-phase2",
			SourceID: sourceID,
			Alias:    alias,
			At:       time.Now().UTC(),
		},
	})
	c.log.Info("composer: p25p2 talker alias",
		"system", system, "src", sourceID, "alias", alias)
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
