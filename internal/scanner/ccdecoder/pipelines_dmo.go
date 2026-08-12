package ccdecoder

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TETRA DMO (Direct Mode Operation) control-side pipeline. Unlike TMO — which has a
// continuous control channel the daemon locks and follows — DMO is infrastructure-
// less peer-to-peer: a transmitting MS keys up, sends a Direct Mode Synchronisation
// Burst (DSB) then a train of Direct Mode Normal Bursts (DNB) carrying the call, and
// the channel is silent between transmissions (EN 300 396-2). So newTETRADMOPipeline
// cannot reuse tetra.ControlChannel (a TMO CC state machine that slices the TMO NDB
// geometry and expects a persistent carrier); it drives the standalone, offline-
// validated DMO burst decoders (dmo.go / dmo_decode.go) over the receiver's dibit +
// soft-differential stream via DMStreamExtractor instead.
//
// Its job on the daemon control path is:
//   - LOCK: publish events.KindCCLocked the first time a DSB's SCH/S decodes CRC-valid
//     with a parseable SYNC PDU, so the cchunt supervisor stops hunting and camps the
//     DM channel. The lock is sticky (no cc.lost on inter-transmission silence) —
//     re-hunting a camped DMO frequency every quiet gap is exactly the churn the TMO
//     pipeline produced on a DMO capture.
//   - COLOUR: recover the DM traffic colour code once (RecoverDMColourCode), so the
//     voice chain can descramble TCH/S. A non-zero tetra_colour_code overrides.
//   - GRANT: publish events.KindGrant (Protocol "tetra-dmo") on the rising edge of a
//     DNB traffic burst train, so the engine starts a same-carrier voice chain that
//     decodes the actual speech (composer.runTETRADMOVoiceChain). Edge-triggered and
//     re-armed only after a traffic drought, because a DMO grant carries no talkgroup
//     (GroupID 0) and the engine does not de-duplicate zero-group grants.
//
// The actual TCH/S → ACELP audio is decoded in the voice chain, not here; this
// pipeline decodes SCH/S (lock) and brute-forces the colour, and only needs to detect
// DNB *presence* to grant. This is #1003, design-first and on-air A/B gated: a green
// synthetic decode is not proof of on-air correctness (the #764/#771 rule).

const (
	// dmoStatusInterval throttles the DMO decode-status debug line (mirrors the TMO
	// tetraStatusInterval).
	dmoStatusInterval = 5 * time.Second
	// dmoColourBatch is how many DNBs to buffer before brute-forcing the DM colour
	// code (RecoverDMColourCode). ~one second of a full-rate call, enough for the
	// confidence gate (dmColourMinCRC/dmColourDominance) to separate the true colour
	// from the chance floor.
	dmoColourBatch = 20
	// dmoColourMaxAttempts caps how many recovery passes to try before giving up and
	// falling back to colour 0 (a channel that never clears the confidence gate is
	// encrypted or unreceivable — chasing it forever would burn CPU on every DNB).
	dmoColourMaxAttempts = 6
	// dmoGrantMinDNB is how many DNBs must follow a lock before a grant fires — a
	// short guard so a lone stray DNB detection does not spawn a call.
	dmoGrantMinDNB = 4
	// dmoGrantRearm is the DNB-traffic drought after which the grant re-arms, so the
	// next PTT transmission grants afresh. Longer than the voice chain's hangtime so a
	// brief intra-call gap does not double-grant an already-active call.
	dmoGrantRearm = 3 * time.Second
)

func newTETRADMOPipeline(opts PipelineOptions) (ProtocolPipeline, error) {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	tetraClockMode, ok := tetrarx.ParseClockMode(opts.System.TETRAClockMode)
	if !ok {
		log.Warn("ccdecoder: unrecognised tetra_clock_mode; falling back to gardner",
			"system", opts.SystemName, "value", opts.System.TETRAClockMode)
	}
	p := &tetraDMOPipeline{
		bus:          opts.Bus,
		log:          log,
		system:       opts.SystemName,
		freqHz:       opts.FrequencyHz,
		rateHz:       opts.SampleRateHz,
		now:          time.Now,
		configColour: opts.System.TETRAColourCode,
		fnSeen:       map[uint8]struct{}{},
		debug:        log.Enabled(context.Background(), slog.LevelDebug),
	}
	if p.configColour != 0 {
		p.colour = p.configColour
		p.colourKnown = true
	}
	p.ext = tetra.NewDMStreamExtractor(p.onBurst)
	p.rx = tetrarx.New(tetrarx.Options{
		SampleRateHz: opts.SampleRateHz,
		DibitSink: func(dibits []uint8, baseIdx int) {
			opts.tapDibits(dibits, baseIdx)
			p.ext.Process(dibits, p.pendingSoft, baseIdx)
			p.pendingSoft = nil
		},
		// The SoftSink fires just before the matching DibitSink with the same base
		// (issue #553), so stash the differentials and hand them to the extractor
		// alongside the dibits, keeping the two strictly parallel.
		SoftSink: func(diffs []complex64, baseIdx int) {
			p.pendingSoft = diffs
		},
		ClockMode:           tetraClockMode,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		// The blind SnapshotCMA equalizer is REQUIRED for DMO, not optional: on the
		// reporter's 438.9 MHz capture it lifts CRC-valid SCH/S from ~6 to ~64 by
		// inverting the ISI that smears the π/4-DQPSK constellation (same lever the
		// TMO CC path and the offline TestTETRADMOReplay run by default).
		EnableEqualizer: true,
	})
	log.Info("ccdecoder: tetra DMO pipeline configured",
		"system", opts.SystemName, "freq_hz", opts.FrequencyHz,
		"colour_override", opts.System.TETRAColourCode)
	return p, nil
}

type tetraDMOPipeline struct {
	rx  *tetrarx.Receiver
	ext *tetra.DMStreamExtractor
	bus *events.Bus
	log *slog.Logger

	system string
	freqHz uint32
	rateHz float64
	now    func() time.Time
	debug  bool

	// SoftSink → DibitSink hand-off (both fire on the single rx.Process goroutine).
	pendingSoft []complex64

	// Colour recovery: 0 = auto-recover via RecoverDMColourCode; a non-zero
	// tetra_colour_code overrides.
	configColour  uint32
	colour        uint32
	colourKnown   bool
	colourCand    []tetra.DMBurst
	colourTries   int

	// Lock + liveness.
	locked bool
	fnSeen map[uint8]struct{}

	// Grant edge-trigger state.
	grantActive  bool
	dnbSinceLock int
	lastDNB      time.Time

	// Status counters (single-goroutine; no atomics needed).
	dsbTotal, dsbCRC, dnbTotal, tchCRC int64
	lastLog                            time.Time
}

func (p *tetraDMOPipeline) Process(iq []complex64) {
	p.rx.Process(iq)
	p.maybeLogStatus()
}

// onBurst is the DMStreamExtractor callback, invoked (in time order) for each
// newly-completed DSB/DNB during rx.Process. Runs on the single receiver goroutine,
// so all pipeline state below is touched by exactly one writer.
func (p *tetraDMOPipeline) onBurst(b tetra.DMBurst) {
	switch b.Kind {
	case tetra.DMBurstSync:
		p.dsbTotal++
		type1, ok := tetra.DecodeDMSCHS(b)
		if !ok {
			return
		}
		p.dsbCRC++
		if pdu, pok := tetra.ParseSyncPDU(type1); pok {
			p.fnSeen[pdu.FN] = struct{}{}
			p.maybeLock()
		}
	case tetra.DMBurstNormal:
		p.dnbTotal++
		now := p.now()
		// Re-arm the grant after a traffic drought so the next PTT grants afresh.
		if !p.lastDNB.IsZero() && now.Sub(p.lastDNB) > dmoGrantRearm {
			p.grantActive = false
			p.dnbSinceLock = 0
		}
		p.lastDNB = now
		p.dnbSinceLock++
		p.recoverColour(b)
		// Telemetry only: count CRC-valid TCH/S when the colour is known (the voice
		// chain does the real decode). Cheap — a single decode at the known colour.
		if p.colourKnown {
			if len(tetra.DMBurstTCHSpeechSoft(b, p.colour)) == 2 ||
				len(tetra.DMBurstTCHSpeech(b, p.colour)) == 2 {
				p.tchCRC++
			}
		}
		p.maybeGrant()
	}
}

// maybeLock publishes cc.locked once, on the first CRC-valid SCH/S with a parseable
// SYNC PDU. The lock is sticky: the pipeline never publishes cc.lost, so a camped DMO
// frequency is not re-hunted during the silence between transmissions.
func (p *tetraDMOPipeline) maybeLock() {
	if p.locked || p.bus == nil {
		return
	}
	p.locked = true
	p.bus.Publish(events.Event{
		Kind:    events.KindCCLocked,
		Payload: tetra.LockState{FrequencyHz: p.freqHz},
	})
	p.log.Info("tetra dmo cc locked", "freq", p.freqHz, "system", p.system)
}

// recoverColour brute-forces the DM traffic colour code once (RecoverDMColourCode),
// buffering DNBs until a batch is available. No-op once the colour is known (config
// override or a prior successful recovery) or after dmoColourMaxAttempts give up.
func (p *tetraDMOPipeline) recoverColour(b tetra.DMBurst) {
	if p.colourKnown || p.colourTries >= dmoColourMaxAttempts {
		return
	}
	p.colourCand = append(p.colourCand, b)
	if len(p.colourCand) < dmoColourBatch {
		return
	}
	p.colourTries++
	if c, n, confident := tetra.RecoverDMColourCode(p.colourCand); confident {
		p.colour = c
		p.colourKnown = true
		p.log.Info("tetra dmo colour code recovered", "colour", c, "crc_valid_tchs", n, "system", p.system)
	}
	// Keep only the freshest half so the next attempt reflects current channel
	// conditions rather than re-scoring stale bursts.
	keep := len(p.colourCand) / 2
	p.colourCand = append(p.colourCand[:0], p.colourCand[keep:]...)
}

// maybeGrant publishes a tetra-dmo grant on the rising edge of a DNB traffic burst
// train (after a short DNB-count guard). DMO carries no talkgroup, so GroupID is 0;
// the engine does not de-duplicate zero-group grants, so this must fire exactly once
// per transmission — grantActive gates that, re-armed only after dmoGrantRearm of DNB
// silence. The recovered DM colour rides in TETRAColourExt so the voice chain can
// descramble TCH/S; when it is not yet recovered the grant still fires (colour 0
// hint) and the voice chain recovers it independently.
func (p *tetraDMOPipeline) maybeGrant() {
	if p.grantActive || p.bus == nil || p.dnbSinceLock < dmoGrantMinDNB {
		return
	}
	p.grantActive = true
	p.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:         p.system,
			Protocol:       "tetra-dmo",
			FrequencyHz:    p.freqHz,
			TETRAColourExt: p.colour,
			At:             p.now(),
		},
	})
	p.log.Info("tetra dmo grant (traffic detected)",
		"freq", p.freqHz, "colour", p.colour, "system", p.system)
}

func (p *tetraDMOPipeline) maybeLogStatus() {
	if !p.debug || p.log == nil {
		return
	}
	now := p.now()
	if p.lastLog.IsZero() {
		p.lastLog = now
		return
	}
	if now.Sub(p.lastLog) < dmoStatusInterval {
		return
	}
	p.lastLog = now
	p.log.Debug("tetra dmo: decode status",
		"system", p.system,
		"locked", p.locked,
		"carrier_off_hz", math.Round(p.rx.CarrierOffsetHz()*10)/10,
		"dsb_total", p.dsbTotal,
		"dsb_schs_crc", p.dsbCRC,
		"dnb_total", p.dnbTotal,
		"tch_crc", p.tchCRC,
		"distinct_fn", len(p.fnSeen),
		"colour", p.colour,
		"colour_known", p.colourKnown,
		"grant_active", p.grantActive,
	)
}

// AFCOffsetHz reports the receiver's measured carrier offset (Hz). Satisfies the
// afcReporter capability the decoder type-asserts for.
func (p *tetraDMOPipeline) AFCOffsetHz() float64 { return p.rx.CarrierOffsetHz() }

// BSCHCounts exposes the DSB SCH/S (ok, fail) decode counts as the DMO analogue of
// the TMO BSCH counts, from which the decoder derives a decode-quality bucket.
func (p *tetraDMOPipeline) BSCHCounts() (ok, fail int64) {
	return p.dsbCRC, p.dsbTotal - p.dsbCRC
}

func (p *tetraDMOPipeline) Reset() {
	p.rx.Reset()
	p.ext.Reset()
	p.pendingSoft = nil
	p.lastLog = time.Time{}
}

func (p *tetraDMOPipeline) Close() error { return nil }
