package composer

import (
	"context"
	"sync/atomic"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
)

// runTETRADMOVoiceChain decodes one TETRA DMO (Direct Mode) transmission on the
// same-carrier tap the DMO pipeline granted. DMO has no separate traffic channel and
// no TDMA per-call slots: a single MS keys up and transmits a DSB + a train of DNBs
// on the decoded carrier (EN 300 396-2), so unlike the TMO same-carrier demux (usage-
// marker routing across four slots) this is a self-contained single-call chain,
// closest in shape to the solo-tap runTETRAVoiceChain. It decimates the tap IQ to the
// TETRA symbol rate, recovers the π/4-DQPSK stream with the shared receiver (blind CMA
// equalizer + soft differentials, as the offline DMO path requires), slices each DNB
// with the streaming DMStreamExtractor, TCH/S-decodes it (soft, with a hard fallback)
// and emits the 137-bit speech frames to the recorder — which renders them to PCM with
// the same clean-room ACELP vocoder ("tetra-acelp") the TMO path uses.
//
// colourHint is the DM traffic colour code the pipeline stamped on the grant. Because
// the grant fires on the first DNBs (before the pipeline finishes brute-forcing the
// colour), the hint is often 0, so this chain recovers the colour on its own and
// decodes the buffer retroactively once known — no leading speech is lost. Recovery
// runs, in preference order: (1) liveColour — a poll of the control pipeline's own
// recovery via the same-carrier source (dmoColourSource), verified against a few
// buffered bursts before adoption, at ~1/64 the cost of a brute force; (2) the local
// RecoverDMColourCode brute force, scored ONLY on slot-grid-qualified bursts — the DNB
// correlator false-alarms ~18/s (tetra/dmo_grid.go), and un-gated scoring windows were
// ~half noise, which is why the dominance gate never cleared on air and all six
// attempts burned. This is #1003 work: on-air A/B still gates it (a green synthetic
// decode is not proof — #764/#771).
func (c *Composer) runTETRADMOVoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz float64, colourHint, baseMNI uint32, liveColour func() (uint32, bool), done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-tetra-dmo:"+serial, nil)

	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	fe := newTETRAVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()
	rs, _ := c.sink.(rawFrameSink)

	dec := &dmoVoiceDecoder{
		c:          c,
		serial:     serial,
		bt:         bt,
		rs:         rs,
		colour:     colourHint,
		baseMNI:    baseMNI &^ 0x3F,
		liveColour: liveColour,
		grid:       tetra.NewDMSlotGrid(),
		// A non-zero hint from the grant is the pipeline's already-recovered colour
		// (or the operator's tetra_colour_code override), so it counts as recovered.
		colourKnown:     colourHint != 0,
		colourRecovered: colourHint != 0,
	}
	ext := tetra.NewDMStreamExtractor(dec.onBurst)

	var pendingSoft []complex64
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: symbolHz,
		DibitSink: func(d []uint8, base int) {
			ext.Process(d, pendingSoft, base)
			pendingSoft = nil
		},
		SoftSink:            func(diffs []complex64, base int) { pendingSoft = diffs },
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true, // invert the ISI that garbles DMO TCH/S (required for DMO)
		EnableDCBlock:       true, // strip the front-end DC spur on the voice tap
	})

	c.log.Info("composer: tetra DMO voice follow started — DNB TCH/S decode + ACELP vocoder",
		"serial", serial, "colour_hint", colourHint, "rate_hz", symbolHz)
	defer func() {
		ext.Flush() // emit any tail burst still inside the extractor window
		dec.flush() // decode any DNBs still buffered awaiting colour recovery
		c.log.Info("composer: tetra DMO voice follow ended",
			"serial", serial, "dnb_bursts", dec.dnb.Load(),
			"speech_frames", dec.speech.Load(), "bfi_count", dec.bfi.Load(),
			"colour", dec.colour, "colour_recovered", dec.colourRecovered,
			"colour_attempts", dec.colourTries)
	}()

	// feBuf is the reused front-end output scratch: fe.Process(nil, …) allocated
	// a fresh slice per chunk (~2700/s on SoapyRemote's small datagrams), pure GC
	// pressure on the one goroutine that must keep up with the tap.
	var feBuf []complex64
	process := func(iq []complex64) {
		bt.observe(iq)
		feBuf = fe.Process(feBuf[:0], iq)
		rx.Process(feBuf)
	}
	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Drain buffered IQ so the transmission tail is recorded (mirrors
			// runTETRAVoiceChain's drainTETRAIQ).
			drainTETRAIQ(iqCh, process)
			return
		case <-touchTicker.C:
		case iq, ok := <-iqCh:
			if !ok {
				return
			}
			process(iq)
		}
	}
}

const (
	// dmoVoiceColourBatch is how many DNBs the voice chain buffers before attempting
	// colour-code recovery (matches the pipeline's dmoColourBatch cadence).
	dmoVoiceColourBatch = 20
	// dmoVoiceColourMax caps the buffer: past this, if no colour has cleared the
	// confidence gate, decode the buffer at colour 0 (a clear radio-to-radio call) and
	// stop buffering — an encrypted/unrecoverable call then simply yields no speech.
	dmoVoiceColourMax = 120
	// dmoVoiceColourMaxAttempts caps how many recovery passes to run, mirroring the
	// control pipeline's dmoColourMaxAttempts.
	//
	// RecoverDMColourCode is a 64-colour brute force and each candidate is a full
	// soft-Viterbi TCH/S decode over every buffered burst, plus a hard decode on the
	// (usually failing) fallback path. Retrying it on EVERY arriving burst from buffer
	// size 20 up to 120 is 64·Σ(20..120) ≈ 450 000 Viterbi decodes per call, crammed
	// into the few seconds it takes to accumulate them — enough to starve the
	// same-carrier IQ tap feeding this very chain ("voice tap dropped IQ to a lagging
	// voice consumer"). Attempting only at batch boundaries, over a decimated buffer,
	// brings it to the ~10k the control pipeline already budgets for.
	dmoVoiceColourMaxAttempts = 6
	// dmoVoiceHintMinBursts / dmoVoiceHintMinValid gate adopting the control
	// pipeline's recovered colour (liveColour): at least MinBursts slot-grid-
	// qualified bursts must be buffered, and decoding them at the hinted colour
	// must yield at least MinValid CRC-valid bursts. Verification is N single-
	// colour decodes — ~1/64 of one brute-force pass — so a correct hint makes
	// the local brute force unnecessary, and a wrong/stale one cannot blindly
	// latch.
	dmoVoiceHintMinBursts = 4
	dmoVoiceHintMinValid  = 2
)

// dmoVoiceDecoder holds the streaming DMO voice decode state for one call. All methods
// run on the single receiver goroutine (the voice chain's process loop), so the fields
// need no locking; the atomic counters are read from the deferred ended-log on the
// same goroutine after the loop exits.
type dmoVoiceDecoder struct {
	c      *Composer
	serial string
	bt     *boundaryTracker
	rs     rawFrameSink

	colour uint32
	// baseMNI is the DMO network MNI (ExtendedColourCode(MCC, MNC, 0)) from the
	// grant's tetra_mcc/tetra_mnc, folded into colour recovery so a non-zero-MNI
	// network decodes, and used as the clear-fallback seed (base | colour 0)
	// instead of a bare 0. Zero on an MNI-0 radio-to-radio DMO.
	baseMNI     uint32
	colourKnown bool
	// colourRecovered distinguishes "a colour cleared verification (local brute
	// force or an adopted pipeline hint)" from "we gave up and fell back to
	// colour 0", which colourKnown alone conflates — the end-of-call log used to
	// claim colour_known=true colour=0 on a call where nothing was ever recovered.
	colourRecovered bool
	buffer          []tetra.DMBurst // ALL DNBs awaiting colour recovery (retroactive decode)
	// scored holds only the slot-grid-QUALIFIED DNBs (freshest
	// dmoVoiceColourBatch), the set colour recovery and hint verification score
	// against. The DNB correlator false-alarms ~18/s, so an un-gated scoring
	// window is ~half noise on a live tap and RecoverDMColourCode's dominance
	// gate can never clear — the on-air "colour_attempts=6, colour_recovered=
	// false, all BFI" failure. Buffering/emission stay un-gated (the CRC gates
	// output; a pre-latch real burst must not be dropped).
	scored []tetra.DMBurst
	// grid votes DNB leads onto the 255-dibit slot grid to qualify them (the
	// same tetra.DMSlotGrid the control pipeline uses).
	grid *tetra.DMSlotGrid
	// liveColour, when non-nil, polls the control pipeline's own colour
	// recovery (via the same-carrier source). lastHint/lastHintValid remember a
	// hinted value that FAILED verification so it is not re-verified every
	// burst — only a changed hint is retried.
	liveColour    func() (uint32, bool)
	lastHint      uint32
	lastHintValid bool
	// colourTries counts recovery passes run, capped at dmoVoiceColourMaxAttempts.
	// sinceTry is how many QUALIFIED bursts arrived since the last pass, so the
	// brute force runs once per batch instead of once per burst.
	colourTries int
	sinceTry    int

	dnb, speech, bfi atomic.Uint64
}

// tryRecoverColour runs one capped colour-recovery pass and reports whether the
// confidence gate was cleared.
//
// It scores only the qualified window (d.scored — the freshest
// dmoVoiceColourBatch slot-grid-qualified bursts), not the whole buffer: the
// gate needs a couple of dozen REAL bursts to separate the true colour from the
// ~1/256 chance floor, and correlator noise in the window dilutes the dominance
// ratio it needs. The full buffer is deliberately left intact — unlike the
// control pipeline, which only wants the colour and can decimate, this chain
// must still decode every buffered burst retroactively once the colour lands,
// or the start of the transmission is lost from the recording. A call whose
// grid never latched (very short PTT) falls back to the buffer tail — worse
// odds, but better than scoring nothing at flush.
func (d *dmoVoiceDecoder) tryRecoverColour() bool {
	d.colourTries++
	d.sinceTry = 0
	scored := d.scored
	if len(scored) == 0 {
		scored = d.buffer
		if len(scored) > dmoVoiceColourBatch {
			scored = scored[len(scored)-dmoVoiceColourBatch:]
		}
	}
	c, _, ok := tetra.RecoverDMColourCode(scored, d.baseMNI)
	if !ok {
		return false
	}
	d.colour, d.colourKnown, d.colourRecovered = c, true, true
	d.c.log.Info("composer: tetra DMO colour code recovered",
		"serial", d.serial, "colour", c, "attempt", d.colourTries)
	return true
}

// tryAdoptLiveColour polls the control pipeline's own colour recovery and, when
// it has an answer this chain hasn't already rejected, verifies it against the
// buffered qualified bursts before adopting: at least dmoVoiceHintMinValid of
// them must TCH/S-decode CRC-valid at the hinted colour. Verification is a
// handful of single-colour decodes (~1/64 of one brute-force pass), so a
// correct hint replaces the local brute force almost for free, while a wrong
// or stale hint cannot blindly latch. A hint that fails is remembered
// (lastHint) and only re-verified if the pipeline's answer changes. Returns
// true when a colour was adopted.
func (d *dmoVoiceDecoder) tryAdoptLiveColour() bool {
	if d.liveColour == nil || d.colourRecovered {
		return false
	}
	c, known := d.liveColour()
	if !known || (d.colourKnown && c == d.colour) {
		return false
	}
	if d.lastHintValid && c == d.lastHint {
		return false // this exact value already failed verification here
	}
	if len(d.scored) < dmoVoiceHintMinBursts {
		return false // too few real bursts to verify; retry as more arrive
	}
	valid := 0
	for i := len(d.scored) - 1; i >= 0 && valid < dmoVoiceHintMinValid; i-- {
		bb := d.scored[i]
		if len(tetra.DMBurstTCHSpeechSoft(bb, c)) == 2 || len(tetra.DMBurstTCHSpeech(bb, c)) == 2 {
			valid++
		}
	}
	if valid < dmoVoiceHintMinValid {
		d.lastHint, d.lastHintValid = c, true
		return false
	}
	d.colour, d.colourKnown, d.colourRecovered = c, true, true
	d.c.log.Info("composer: tetra DMO colour adopted from control pipeline",
		"serial", d.serial, "colour", c)
	return true
}

// onBurst handles one streamed DMO burst. DSBs carry no speech (signalling only), so
// only DNBs are decoded. Until the colour code is known, DNBs are buffered; once
// recovered/adopted (or the cap forces a colour-0 fallback) the buffer is decoded in
// order and each subsequent DNB decodes immediately.
func (d *dmoVoiceDecoder) onBurst(b tetra.DMBurst) {
	if b.Kind != tetra.DMBurstNormal {
		return
	}
	d.dnb.Add(1)
	qualified := d.grid.Observe(b.Lead)
	if d.colourKnown {
		// A give-up latch (colourKnown without colourRecovered) can still be
		// rescued by the pipeline's recovery landing later: keep the qualified
		// window fresh and adopt a verified hint for the remaining bursts.
		if !d.colourRecovered {
			if qualified {
				d.pushScored(b)
			}
			d.tryAdoptLiveColour()
		}
		d.emit(b)
		return
	}
	d.buffer = append(d.buffer, b)
	if qualified {
		d.pushScored(b)
		d.sinceTry++
	}
	// Attempt only once per batch of fresh QUALIFIED bursts, and only up to the
	// attempt cap — re-running the 64-colour brute force on every arriving burst is
	// what made this chain starve its own IQ tap.
	canBrute := d.colourTries < dmoVoiceColourMaxAttempts &&
		len(d.scored) >= dmoVoiceColourBatch &&
		(d.colourTries == 0 || d.sinceTry >= dmoVoiceColourBatch)
	switch {
	case d.tryAdoptLiveColour():
		// Adopted the pipeline's verified colour; fall through and flush.
	case canBrute && d.tryRecoverColour():
		// Recovered locally; fall through and flush.
	case len(d.buffer) >= dmoVoiceColourMax || d.colourTries >= dmoVoiceColourMaxAttempts:
		// Give up recovering; assume clear colour 0 on top of the known MNI
		// (baseMNI | 0 — plain 0 when MNI is 0). A genuinely encrypted call then
		// yields no CRC-valid speech (bfi), which the hangtime ends normally.
		d.colour, d.colourKnown = d.baseMNI, true
	default:
		return // keep buffering
	}
	buf := d.buffer
	d.buffer = nil
	for _, bb := range buf {
		d.emit(bb)
	}
}

// pushScored appends a qualified burst to the scoring window, keeping only the
// freshest dmoVoiceColourBatch entries.
func (d *dmoVoiceDecoder) pushScored(b tetra.DMBurst) {
	d.scored = append(d.scored, b)
	if len(d.scored) > dmoVoiceColourBatch {
		d.scored = append(d.scored[:0], d.scored[len(d.scored)-dmoVoiceColourBatch:]...)
	}
}

// emit TCH/S-decodes one DNB (soft with a hard fallback) at the known colour and writes
// its speech frames to the recorder, refreshing call liveness on real speech. A DNB
// that yields no CRC-valid speech is a Bad Frame Indication (encrypted/corrupt).
func (d *dmoVoiceDecoder) emit(b tetra.DMBurst) {
	frames := tetra.DMBurstTCHSpeechSoft(b, d.colour)
	if len(frames) != 2 {
		frames = tetra.DMBurstTCHSpeech(b, d.colour)
	}
	if len(frames) != 2 {
		d.bfi.Add(1)
		return
	}
	d.bt.onVoice(0)
	for _, sf := range frames {
		d.speech.Add(1)
		if d.rs != nil {
			if err := d.rs.WriteRawFrame(d.serial, sf); err != nil {
				d.c.log.Warn("composer: TETRA DMO speech-frame write failed", "serial", d.serial, "err", err)
			}
		}
	}
}

// flush decodes any DNBs still buffered awaiting colour recovery at end-of-call. If the
// colour never cleared the confidence gate, it makes a final best-effort attempt, then
// falls back to colour 0 so a short clear call is not dropped.
//
// The fallback sets colourKnown (the buffer is about to be decoded at SOME colour) but
// deliberately NOT colourRecovered, so the end-of-call log distinguishes "recovered
// colour 0" from "never recovered anything". Setting the flag unconditionally is what
// made a call that decoded nothing report `colour=0 colour_known=true`, which reads as
// a successful recovery.
func (d *dmoVoiceDecoder) flush() {
	if len(d.buffer) == 0 {
		return
	}
	if !d.colourKnown {
		if !d.tryAdoptLiveColour() {
			d.tryRecoverColour()
		}
		if !d.colourRecovered {
			// Nothing cleared the gate — fall back to clear colour 0 on top of the
			// known MNI (baseMNI | 0) so a short clear call still decodes.
			d.colour = d.baseMNI
		}
		d.colourKnown = true
	}
	buf := d.buffer
	d.buffer = nil
	for _, bb := range buf {
		d.emit(bb)
	}
}
