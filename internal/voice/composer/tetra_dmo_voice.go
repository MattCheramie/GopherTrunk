package composer

import (
	"context"
	"fmt"
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
// equalizer + soft differentials, as the offline DMO path requires), slices each
// DSB/DNB with the streaming DMStreamExtractor, TCH/S-decodes each DNB (soft, with a
// hard fallback) and emits the 137-bit speech frames to the recorder — which renders
// them to PCM with the same clean-room ACELP vocoder ("tetra-acelp") the TMO path uses.
//
// The TCH/S scramble seed is PER TRANSMISSION on air (tetra/dmo_seed_tracker.go: the
// 12 Sep #1003 capture carried a different 30-bit seed on each PTT — the
// transmitting radio's source address under a fixed prefix), so the chain runs its
// own tetra.DMSeedTracker: the DSB SCH/H it slices announces the seed, the first
// error-free DNB solves it exactly, and until one of those has confirmed a seed the
// DNBs are buffered and decoded retroactively — no leading speech is lost. seedHint
// is the pipeline's answer stamped on the grant (verified there, so adopted as is
// when non-zero); liveColour polls the pipeline for the answer it reaches after the
// grant, which the tracker confirms against this chain's own bursts before use.
func (c *Composer) runTETRADMOVoiceChain(ctx context.Context, serial string, iqCh <-chan []complex64, iqHz float64, seedHint uint32, liveColour func() (uint32, bool), done chan<- struct{}) {
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
		seeds:      tetra.NewDMSeedTracker(),
		liveColour: liveColour,
		grid:       tetra.NewDMSlotGrid(),
	}
	if seedHint != 0 {
		// The grant's seed is the pipeline's verified answer (or the operator's
		// tetra_colour_code override): use it from the first burst.
		dec.seeds.Adopt(seedHint)
		dec.colourFromPipeline = true
	}
	ext := tetra.NewDMStreamExtractor(dec.onBurst)

	// The receiver knobs are shared with the control pipeline through
	// tetrarx.DMOOptions (equalizer on, DC block OFF — the 20 Aug #1003 run showed
	// this chain, with the DC blocker on, could not verify the colour the pipeline
	// recovered on the same carrier). Only the sinks are attached here.
	var pendingSoft []complex64
	rxOpts := tetrarx.DMOOptions(symbolHz)
	rxOpts.DibitSink = func(d []uint8, base int) {
		ext.Process(d, pendingSoft, base)
		pendingSoft = nil
	}
	rxOpts.SoftSink = func(diffs []complex64, base int) { pendingSoft = diffs }
	rx := tetrarx.New(rxOpts)

	c.log.Info("composer: tetra DMO voice follow started — DNB TCH/S decode + ACELP vocoder",
		"serial", serial, "seed_hint", fmt.Sprintf("%#010x", seedHint), "rate_hz", symbolHz)
	defer func() {
		ext.Flush() // emit any tail burst still inside the extractor window
		dec.flush() // decode any DNBs still buffered awaiting the seed
		seed, known := dec.seeds.Seed()
		c.log.Info("composer: tetra DMO voice follow ended",
			"serial", serial, "dnb_bursts", dec.dnb.Load(),
			"speech_frames", dec.speech.Load(), "bfi_count", dec.bfi.Load(),
			"seed", fmt.Sprintf("%#010x", seed), "seed_known", known,
			"seed_verified", dec.seeds.Verified(),
			"seed_from_pipeline", dec.colourFromPipeline,
			"bursts_solved", dec.seeds.Solved,
			"exact_adopts", dec.seeds.ExactAdopts, "hint_adopts", dec.seeds.HintAdopts)
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

// dmoVoiceSeedMax caps the DNBs buffered while no seed is known: past this, if
// neither an exact solve nor a confirmed hint has landed (an encrypted call, or
// one too weak for a single error-free burst), stop buffering and decode at the
// best guess available — the pipeline's live answer if it has one, else seed 0
// (the spec default for a radio-to-radio DMO). An unrecoverable call then simply
// yields no speech. ~7 s of a full-rate call.
const dmoVoiceSeedMax = 120

// dmoVoiceDecoder holds the streaming DMO voice decode state for one call. All
// methods run on the single receiver goroutine (the voice chain's process loop), so
// the fields need no locking; the atomic counters are read from the deferred
// ended-log on the same goroutine after the loop exits.
type dmoVoiceDecoder struct {
	c      *Composer
	serial string
	bt     *boundaryTracker
	rs     rawFrameSink

	// seeds learns the transmission's scramble seed from this chain's own DSBs and
	// DNBs (and from the pipeline's hint, which it confirms before use).
	seeds *tetra.DMSeedTracker
	// colourFromPipeline records that the seed in use came from the pipeline
	// (grant or give-up adoption) rather than this chain's own bursts.
	colourFromPipeline bool
	// buffer holds every DNB seen while no seed was known, in order, so the
	// start of the transmission is decoded retroactively once the seed lands.
	buffer []tetra.DMBurst
	// grid votes DNB leads onto the 255-dibit slot grid to qualify them (the same
	// tetra.DMSlotGrid the control pipeline uses): the DNB correlator false-alarms
	// ~18/s, and only qualified bursts are allowed to teach the tracker a seed.
	// Buffering/emission stay un-gated (the CRC gates output; a pre-latch real
	// burst must not be dropped).
	grid *tetra.DMSlotGrid
	// liveColour, when non-nil, polls the control pipeline's own seed recovery
	// (via the same-carrier source).
	liveColour func() (uint32, bool)

	dnb, speech, bfi atomic.Uint64
}

// onBurst handles one streamed DMO burst. A DSB carries no speech but its SCH/H
// announces the transmission's seed, so it feeds the tracker; a DNB is decoded at
// once when the seed is known, else buffered until it is.
func (d *dmoVoiceDecoder) onBurst(b tetra.DMBurst) {
	switch b.Kind {
	case tetra.DMBurstSync:
		d.seeds.ObserveDSB(b)
		return
	case tetra.DMBurstNormal:
	default:
		return
	}
	d.dnb.Add(1)
	qualified := d.grid.Observe(b.Lead)
	if d.liveColour != nil && !d.seeds.Verified() {
		if s, known := d.liveColour(); known {
			d.seeds.Hint(s)
		}
	}
	if _, known := d.seeds.Seed(); known {
		d.emit(b, qualified)
		return
	}
	d.buffer = append(d.buffer, b)
	if qualified {
		// Let the qualified burst teach the tracker (an exact solve, or a CRC
		// confirmation of the hint). Its own frames are emitted with the buffer.
		_, seed, adopted := d.seeds.ObserveDNB(b, true)
		if adopted {
			d.c.log.Info("composer: tetra DMO scramble seed recovered",
				"serial", d.serial, "seed", fmt.Sprintf("%#010x", seed),
				"exact_adopts", d.seeds.ExactAdopts, "hint_adopts", d.seeds.HintAdopts)
		}
	}
	if _, known := d.seeds.Seed(); !known && len(d.buffer) >= dmoVoiceSeedMax {
		d.giveUp()
	}
	if _, known := d.seeds.Seed(); !known {
		return // keep buffering
	}
	buf := d.buffer
	d.buffer = nil
	for _, bb := range buf {
		d.emit(bb, true)
	}
}

// giveUp installs a fallback seed once the buffer cap is hit with nothing
// confirmed: the pipeline's live answer if it has one, else seed 0. A later exact
// solve still replaces it.
func (d *dmoVoiceDecoder) giveUp() {
	if d.liveColour != nil {
		if s, known := d.liveColour(); known {
			d.seeds.Fallback(s)
			d.colourFromPipeline = true
			d.c.log.Info("composer: tetra DMO seed adopted from control pipeline (not confirmed on this chain's bursts)",
				"serial", d.serial, "seed", fmt.Sprintf("%#010x", s))
			return
		}
	}
	d.seeds.Fallback(0)
}

// emit TCH/S-decodes one DNB through the tracker (which keeps learning: an exact
// solve on a burst of a NEW transmission switches the seed) and writes its speech
// frames to the recorder, refreshing call liveness on real speech. A DNB that
// yields no CRC-valid speech is a Bad Frame Indication (noise/encrypted/corrupt).
func (d *dmoVoiceDecoder) emit(b tetra.DMBurst, qualified bool) {
	// Every burst goes through the tracker: the exact solve is safe on an
	// unqualified (possibly noise) burst — no solution can come from noise —
	// and it is what catches the first bursts of a fast re-key before the slot
	// grid re-latches; only hint confirmation is withheld from unqualified ones.
	frames, seed, adopted := d.seeds.ObserveDNB(b, qualified)
	if adopted {
		d.c.log.Info("composer: tetra DMO scramble seed changed",
			"serial", d.serial, "seed", fmt.Sprintf("%#010x", seed))
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

// flush decodes any DNBs still buffered awaiting the seed at end-of-call. If no
// seed was ever confirmed, it falls back (giveUp) so a short clear call is still
// decoded at the best available guess; the ended-log reports seed_verified=false
// so a fallback is never mistaken for a recovery.
func (d *dmoVoiceDecoder) flush() {
	if len(d.buffer) == 0 {
		return
	}
	if _, known := d.seeds.Seed(); !known {
		d.giveUp()
	}
	buf := d.buffer
	d.buffer = nil
	for _, bb := range buf {
		d.emit(bb, true)
	}
}
