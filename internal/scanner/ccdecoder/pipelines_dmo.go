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
// A raw DNB detection is NOT evidence of traffic. The DNB correlator is an 11-dibit
// training-sequence match at tolerance 2 under eight matched filters, which fires
// ~18 times a second on an idle channel (the arithmetic is in tetra/dmo_grid.go), and
// a DMO channel is idle most of the time. Taking that at face value is what made the
// daemon grant and open a recording ~230 ms after startup and then latch the grant
// forever, so the operator's real PTT never granted at all. Every DNB is therefore
// qualified against the 255-dibit TETRA slot grid (tetra.DMSlotGrid) before it counts
// towards the grant, the colour recovery or the drought: a real transmission comes
// from one radio on one clock, so all its DNB leads share one residue mod 255, while
// noise hits spread uniformly over all 255.
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
	// dmoGrantMinDNB is how many SLOT-GRID-QUALIFIED DNBs must follow a lock before a
	// grant fires — a short guard on top of the grid gate so a call is not spawned
	// before the burst train is unambiguous.
	dmoGrantMinDNB = 4
	// dmoGrantRearm is the qualified-DNB traffic drought after which the grant re-arms,
	// so the next PTT transmission grants afresh. Longer than the voice chain's hangtime
	// so a brief intra-call gap does not double-grant an already-active call. It is
	// evaluated from Process (per IQ chunk) as well as from onBurst, because once the
	// grid gate is in place a silent channel produces NO qualified bursts at all — the
	// old onBurst-only check could never fire, which is precisely why the grant latched.
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
		// baseMNI is the network's MNI (MCC<<20 | MNC<<6) — the non-colour half of
		// the extended colour code. On a DMO network with a non-zero MNI the TCH/S
		// traffic seed is ExtendedColourCode(MCC, MNC, colour), so colour recovery
		// must search on top of this base or it never reaches the true seed (the
		// reporter's Motorola DMO at MCC 250 / MNC 1 — #1003 follow-up).
		baseMNI:    tetra.ExtendedColourCode(opts.System.TETRAMCC, opts.System.TETRAMNC, 0),
		colourSink: opts.TETRADMOColourSink,
		fnSeen:     map[uint8]struct{}{},
		debug:      log.Enabled(context.Background(), slog.LevelDebug),
	}
	if p.configColour != 0 {
		p.colour = p.configColour
		p.colourKnown = true
		if p.colourSink != nil {
			p.colourSink(p.colour)
		}
	}
	p.ext = tetra.NewDMStreamExtractor(p.onBurst)
	p.grid = tetra.NewDMSlotGrid()
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
	// grid separates real burst trains from the DNB correlator's ~18/s noise
	// false-alarm rate by voting DNB leads onto the 255-dibit slot grid.
	grid *tetra.DMSlotGrid
	bus  *events.Bus
	log  *slog.Logger

	system string
	freqHz uint32
	rateHz float64
	now    func() time.Time
	debug  bool

	// SoftSink → DibitSink hand-off (both fire on the single rx.Process goroutine).
	pendingSoft []complex64

	// Colour recovery: 0 = auto-recover via RecoverDMColourCode; a non-zero
	// tetra_colour_code overrides.
	configColour uint32
	// baseMNI is ExtendedColourCode(MCC, MNC, 0) from tetra_mcc/tetra_mnc — the
	// network MNI folded into every colour-recovery candidate so a non-zero-MNI
	// DMO network decodes (its traffic seed is baseMNI | colour). 0 = MNI 0.
	baseMNI uint32
	colour       uint32
	colourKnown  bool
	colourCand   []tetra.DMBurst
	colourTries  int
	// colourSink, when non-nil, is told the DM colour the moment it is known
	// (config override at construction, or a confident recovery). The decoder
	// exposes it to the same-carrier voice chain, which typically starts
	// before recovery completes (the grant fires at dmoGrantMinDNB=4 bursts,
	// recovery needs dmoColourBatch=20) and would otherwise brute-force the
	// colour all over again — or fail to, on a noisy tap.
	colourSink func(colour uint32)

	// Lock + liveness.
	locked bool
	fnSeen map[uint8]struct{}

	// Grant edge-trigger state.
	grantActive  bool
	dnbSinceLock int
	lastDNB      time.Time

	// Status counters (single-goroutine; no atomics needed). dnbTotal counts RAW
	// correlator detections (mostly noise on an idle channel — kept visible so the
	// false-alarm rate stays diagnosable) and dnbQualified only those that passed
	// the slot-grid gate, which is the number that means "traffic".
	dsbTotal, dsbCRC, dnbTotal, dnbQualified, tchCRC int64
	lastLog                                          time.Time
}

func (p *tetraDMOPipeline) Process(iq []complex64) {
	p.rx.Process(iq)
	p.maybeRearmGrant(p.now())
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
		// Re-arm first, so a burst arriving after a long drought re-learns the grid
		// (the next PTT is a new transmission with its own slot phase).
		p.maybeRearmGrant(now)
		// Qualify against the slot grid: an unqualified detection is a correlator
		// false alarm and must not count as traffic anywhere below.
		if !p.grid.Observe(b.Lead) {
			return
		}
		p.dnbQualified++
		p.lastDNB = now
		p.dnbSinceLock++
		p.recoverColour(b)
		// Telemetry only: count CRC-valid TCH/S when the colour is known (the voice
		// chain does the real decode). This is a full Viterbi pass per burst (two when
		// the soft decode fails and the hard fallback runs), which was unbounded while
		// noise detections reached here; qualified bursts cap it at the ~17/s a real
		// transmission produces, and only while one is in progress. Debug-gated: the
		// counter only feeds the debug status line, so at INFO level the decode is
		// pure waste on the control decode goroutine.
		if p.debug && p.colourKnown {
			if len(tetra.DMBurstTCHSpeechSoft(b, p.colour)) == 2 ||
				len(tetra.DMBurstTCHSpeech(b, p.colour)) == 2 {
				p.tchCRC++
			}
		}
		p.maybeGrant()
	}
}

// maybeRearmGrant clears the grant edge (and the learned slot grid) once qualified
// DNB traffic has been absent for dmoGrantRearm, so the next PTT transmission grants
// afresh. Driven from Process as well as onBurst: with the grid gate in place an idle
// channel produces no qualified bursts at all, so a burst-driven check alone would
// never fire and the grant would latch for the life of the daemon.
func (p *tetraDMOPipeline) maybeRearmGrant(now time.Time) {
	if p.lastDNB.IsZero() || now.Sub(p.lastDNB) <= dmoGrantRearm {
		return
	}
	p.grantActive = false
	p.dnbSinceLock = 0
	p.lastDNB = time.Time{}
	p.grid.Reset()
	// A drought ends the transmission, so the colour-recovery attempt budget is
	// per-transmission, not per-process: without this, six failed attempts (a
	// weak first PTT, or noise-diluted early batches) disabled recovery for the
	// daemon's lifetime — the operator's run showed colour_known=false forever
	// while hundreds of later, perfectly decodable qualified DNBs arrived.
	// Buffered candidates are stale for the same reason: bursts carried across
	// a transmission boundary dilute the next attempt's dominance gate.
	p.colourTries = 0
	p.colourCand = p.colourCand[:0]
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
	if c, n, confident := tetra.RecoverDMColourCode(p.colourCand, p.baseMNI); confident {
		p.colour = c
		p.colourKnown = true
		p.log.Info("tetra dmo colour code recovered", "colour", c, "crc_valid_tchs", n, "system", p.system)
		if p.colourSink != nil {
			p.colourSink(c)
		}
	}
	// Keep only the freshest half so the next attempt reflects current channel
	// conditions rather than re-scoring stale bursts.
	keep := len(p.colourCand) / 2
	p.colourCand = append(p.colourCand[:0], p.colourCand[keep:]...)
}

// maybeGrant publishes a tetra-dmo grant on the rising edge of a QUALIFIED DNB
// traffic burst train. DMO carries no talkgroup, so GroupID is 0; the engine does not
// de-duplicate zero-group grants, so this must fire exactly once per transmission —
// grantActive gates that, re-armed only after dmoGrantRearm of traffic silence. The
// recovered DM colour rides in TETRAColourExt so the voice chain can descramble
// TCH/S; when it is not yet recovered the grant still fires (colour 0 hint) and the
// voice chain recovers it independently.
//
// The lock check is load-bearing, not belt-and-braces: dnbSinceLock is named for a
// lock it never actually consulted, so before this the pipeline would grant on an
// unlocked, silent channel. A DSB SCH/S lock is CRC- and SYNC-PDU-validated and so is
// real evidence a DMO radio is on this frequency; qualified DNBs then say it is
// transmitting traffic right now. Both are required.
func (p *tetraDMOPipeline) maybeGrant() {
	if p.grantActive || p.bus == nil || !p.locked || p.dnbSinceLock < dmoGrantMinDNB {
		return
	}
	p.grantActive = true
	p.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:          p.system,
			Protocol:        "tetra-dmo",
			FrequencyHz:     p.freqHz,
			TETRAColourExt:  p.colour,
			TETRADMOBaseMNI: p.baseMNI,
			At:              p.now(),
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
		"dnb_qualified", p.dnbQualified,
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
	// The receiver re-anchors its dibit index, so the learned slot residue is stale.
	p.grid.Reset()
	p.grantActive = false
	p.dnbSinceLock = 0
	p.lastDNB = time.Time{}
	p.pendingSoft = nil
	p.lastLog = time.Time{}
	// Buffered colour candidates predate the re-anchor and the attempt budget is
	// per-transmission (see maybeRearmGrant); a recovered/configured colour itself
	// stays — the DM colour is a network constant, not receiver state.
	p.colourTries = 0
	p.colourCand = p.colourCand[:0]
}

func (p *tetraDMOPipeline) Close() error { return nil }
