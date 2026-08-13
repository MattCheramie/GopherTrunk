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
// colour), the hint is often 0, so this chain recovers the colour independently from
// its own buffered DNBs (tetra.RecoverDMColourCode) and decodes the buffer
// retroactively once known — no leading speech is lost. This is #1003 work: on-air A/B
// still gates it (a green synthetic decode is not proof — #764/#771).
func (c *Composer) runTETRADMOVoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz float64, colourHint uint32, done chan<- struct{}) {
	defer close(done)
	defer gtlog.Recover(c.log, "voice-chain-tetra-dmo:"+serial, nil)

	bt := c.newBoundaryTracker(serial, 0, nil)
	go bt.run(ctx)

	fe := newTETRAVoiceFrontEnd(iqHz, c.bw)
	symbolHz := fe.OutRateHz()
	rs, _ := c.sink.(rawFrameSink)

	dec := &dmoVoiceDecoder{
		c:      c,
		serial: serial,
		bt:     bt,
		rs:     rs,
		colour: colourHint,
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
		dec.flush() // decode any DNBs still buffered awaiting colour recovery
		c.log.Info("composer: tetra DMO voice follow ended",
			"serial", serial, "dnb_bursts", dec.dnb.Load(),
			"speech_frames", dec.speech.Load(), "bfi_count", dec.bfi.Load(),
			"colour", dec.colour, "colour_recovered", dec.colourRecovered,
			"colour_attempts", dec.colourTries)
	}()

	process := func(iq []complex64) {
		bt.observe(iq)
		rx.Process(fe.Process(nil, iq))
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

	colour      uint32
	colourKnown bool
	// colourRecovered distinguishes "RecoverDMColourCode cleared its confidence
	// gate" from "we gave up and fell back to colour 0", which colourKnown alone
	// conflates — the end-of-call log used to claim colour_known=true colour=0 on a
	// call where nothing was ever recovered.
	colourRecovered bool
	buffer          []tetra.DMBurst // DNBs awaiting colour recovery
	// colourTries counts recovery passes run, capped at dmoVoiceColourMaxAttempts.
	// sinceTry is how many bursts have arrived since the last pass, so the brute
	// force runs once per batch instead of once per burst.
	colourTries int
	sinceTry    int

	dnb, speech, bfi atomic.Uint64
}

// tryRecoverColour runs one capped colour-recovery pass and reports whether the
// confidence gate was cleared.
//
// It scores only the FRESHEST dmoVoiceColourBatch bursts, not the whole buffer: the
// gate needs a couple of dozen bursts to separate the true colour from the ~1/256
// chance floor, so scoring more costs linearly more Viterbi work for no extra
// discrimination. The full buffer is deliberately left intact — unlike the control
// pipeline, which only wants the colour and can decimate, this chain must still
// decode every buffered burst retroactively once the colour lands, or the start of
// the transmission is lost from the recording.
func (d *dmoVoiceDecoder) tryRecoverColour() bool {
	d.colourTries++
	d.sinceTry = 0
	scored := d.buffer
	if len(scored) > dmoVoiceColourBatch {
		scored = scored[len(scored)-dmoVoiceColourBatch:]
	}
	c, _, ok := tetra.RecoverDMColourCode(scored)
	if !ok {
		return false
	}
	d.colour, d.colourKnown, d.colourRecovered = c, true, true
	d.c.log.Info("composer: tetra DMO colour code recovered",
		"serial", d.serial, "colour", c, "attempt", d.colourTries)
	return true
}

// onBurst handles one streamed DMO burst. DSBs carry no speech (signalling only), so
// only DNBs are decoded. Until the colour code is known, DNBs are buffered; once
// recovered (or the cap forces a colour-0 fallback) the buffer is decoded in order and
// each subsequent DNB decodes immediately.
func (d *dmoVoiceDecoder) onBurst(b tetra.DMBurst) {
	if b.Kind != tetra.DMBurstNormal {
		return
	}
	d.dnb.Add(1)
	if d.colourKnown {
		d.emit(b)
		return
	}
	d.buffer = append(d.buffer, b)
	d.sinceTry++
	if len(d.buffer) < dmoVoiceColourBatch {
		return
	}
	// Attempt only once per batch of fresh bursts, and only up to the attempt cap —
	// re-running the 64-colour brute force on every arriving burst is what made this
	// chain starve its own IQ tap.
	canTry := d.colourTries < dmoVoiceColourMaxAttempts &&
		(d.colourTries == 0 || d.sinceTry >= dmoVoiceColourBatch)
	switch {
	case canTry && d.tryRecoverColour():
		// Recovered; fall through and flush the buffer.
	case len(d.buffer) >= dmoVoiceColourMax || d.colourTries >= dmoVoiceColourMaxAttempts:
		// Give up recovering; assume clear colour 0. A genuinely encrypted call then
		// yields no CRC-valid speech (bfi), which the hangtime ends normally.
		d.colour, d.colourKnown = 0, true
	default:
		return // keep buffering
	}
	buf := d.buffer
	d.buffer = nil
	for _, bb := range buf {
		d.emit(bb)
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
		d.tryRecoverColour()
		d.colourKnown = true
	}
	buf := d.buffer
	d.buffer = nil
	for _, bb := range buf {
		d.emit(bb)
	}
}
