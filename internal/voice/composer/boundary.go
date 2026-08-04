package composer

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/metrics"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// boundaryTracker is the protocol-agnostic recording-boundary controller
// shared by every voice chain (FM, DMR, P25 Phase 1 / 2, TETRA). It
// centralises the universal call-boundary behaviour so hangtime +
// split/conversation grouping work identically across protocols:
//
//   - Hangtime end-of-call: once voice has been decoding, the call is
//     ended VoiceHangtime after the last decoded voice frame, instead of
//     waiting out the engine's much longer call-timeout watchdog. This
//     keeps recordings tightly bounded to the actual transmission. It
//     controls teardown timing only — for digital protocols the audio is
//     exactly the decoded voice frames, so hangtime never extends a
//     recording's length.
//   - Talkgroup gating (digital protocols that decode an in-band TG):
//     audio whose talkgroup differs from the granted talkgroup is not
//     written, and a sustained foreign talkgroup ends the call — fixing
//     the shared-frequency case where two virtual tuners decode the same
//     IQ and one call would otherwise append the other talkgroup's audio.
//   - Per-transmission splitting: at an end-of-transmission boundary
//     (a P25 terminator, a DMR voice terminator, an FM squelch gap, …)
//     the recorder rolls to a fresh file when split mode is on.
//
// A chain feeds the tracker two facts — onVoice(tg) per decoded voice
// frame and onTransmissionEnd() at an over boundary — and runs run(ctx)
// in a goroutine. Everything else (Touch throttling, EndCall, segment
// publishing) is handled here so the chains stay thin and uniform.
//
// Concurrency: onVoice / onTransmissionEnd are called from the single
// chain goroutine; run executes in its own goroutine and only reads the
// atomics. The mutable match/segment bookkeeping is touched only by the
// chain goroutine. (For the shared TETRA same-carrier demux the "chain
// goroutine" for onVoice is the owner's single vocoder worker, while the
// demux goroutine calls observe(); the two touch disjoint non-atomic field
// sets — onVoice: lastMatch/foreignRun/voiceSinceRoll; observe: sumSq/nSamp —
// so they do not race. See tetraSlotOwner.)
type boundaryTracker struct {
	c              *Composer
	serial         string
	grantTG        uint32
	patches        map[uint32]bool
	hangtime       time.Duration
	startNano      int64         // unixnano the tracker (≈ the call) started
	noVoiceTimeout time.Duration // teardown window when NO voice ever decodes

	lastVoiceNano atomic.Int64 // unixnano of the last MATCHING voice frame
	sawVoice      atomic.Bool

	// Chain-goroutine-only state.
	lastMatch      bool   // most recent talkgroup-match decision (LDU2 etc. inherit it)
	foreignRun     int    // consecutive frames carrying the SAME foreign talkgroup
	lastForeignTG  uint32 // the foreign talkgroup foreignRun is counting
	voiceSinceRoll bool   // wrote audio since the last segment roll / start
	lastTouchNano  int64
	endOnce        sync.Once

	// Signal-level meter. sumSq / nSamp accumulate the call's received
	// channel power and are touched only by the single chain goroutine
	// (observe). meanDbFSBits / haveSignal are the cross-goroutine
	// snapshot end() reads, so they are atomics. See observe / signalDbFS.
	sumSq        float64
	nSamp        uint64
	meanDbFSBits atomic.Uint64
	haveSignal   atomic.Bool

	// Demod-quality accumulator (per-call SNR/EVM). Populated ONLY when the
	// chain installs the receiver's soft/symbol taps — currently just P25
	// Phase 1 — via observeSoft / observeSymbols; other protocols never call
	// them, so their metrics stay unmeasured (nil in the call record). The
	// taps write on the chain goroutine while end() reads on the bt.run
	// goroutine, so the bounded buffers are guarded by demodMu. See
	// demodMetrics for the estimator choice.
	demodMu sync.Mutex
	softBuf []float32   // C4FM 4-level soft symbols (FM-discriminator path)
	symBuf  []complex64 // CQPSK / linear constellation points
}

// Demod-metric accumulation bounds, mirroring internal/siglab so a per-call
// figure agrees with the `replay -diag` line an operator would compare against.
const (
	// demodWarmupSkip drops the receiver's sync/AGC/timing warm-up symbols,
	// whose EVM is transient and unrepresentative of the settled decode.
	demodWarmupSkip = 256
	// demodMinSymbols is the minimum settled (post-warmup) symbol count before
	// a stable EVM/SNR is reported; a shorter blip reports NULL rather than a
	// noisy figure (mirrors how signal_dbfs is nil when unmeasured).
	demodMinSymbols = 64
	// demodMaxSymbols bounds the per-call buffer past the warm-up prefix —
	// enough for a stable RMS EVM without holding a whole 30 s call in memory.
	demodMaxSymbols = 8192
	// demodSNRCapDB is the finite ceiling for a (near) noise-free residual,
	// mirroring siglab's demodSNRCapDB so a capped value is JSON-marshalable.
	demodSNRCapDB = 99.0
)

// foreignRunToEnd is how many consecutive frames carrying the SAME known
// foreign talkgroup end the call. A couple of frames debounces a single
// mis-decoded Link Control word without letting a real foreign
// transmission append more than ~one LDU of audio (which is gated out
// anyway). Requiring the same value across the run guards against a
// one-off RS-aliased mis-decode (a garbage-but-valid LC) ending a call.
const foreignRunToEnd = 2

// noVoiceStartupFactor sets the no-initial-voice teardown window as a
// multiple of the hangtime gap tolerance: a call that has NEVER decoded a
// matching voice frame is ended this-many hangtimes after it started,
// rather than holding the voice tap for the engine's much longer
// call-timeout watchdog (the "hanging silent call").
//
// A live P25 voice channel delivers LDUs continuously the instant the
// transmitter keys — even a keyed-but-silent PTT sends silence-coded IMBE
// frames, which still drive onVoice — so producing nothing at all for two
// full hangtimes means the carrier was already gone (a stale / late CC
// grant), or we are mis-tuned / wrong-demod / under-gained on it. Either
// way the right move is to free the tap promptly, not sit idle for 30 s;
// if voice does resume the CC re-announces the grant and a fresh call
// starts. Two hangtimes leaves ample margin for normal sync acquisition
// (the first LDU on a healthy call lands well inside one hangtime).
const noVoiceStartupFactor = 2

func (c *Composer) newBoundaryTracker(serial string, grantTG uint32, patched []uint32) *boundaryTracker {
	var patches map[uint32]bool
	if len(patched) > 0 {
		patches = make(map[uint32]bool, len(patched))
		for _, p := range patched {
			patches[p] = true
		}
	}
	return &boundaryTracker{
		c:              c,
		serial:         serial,
		grantTG:        grantTG,
		patches:        patches,
		hangtime:       c.hangtime,
		startNano:      time.Now().UnixNano(),
		noVoiceTimeout: c.hangtime * noVoiceStartupFactor,
		lastMatch:      true, // write optimistically until a talkgroup proves foreign
	}
}

// onVoice records one decoded voice frame and returns whether its audio
// should be written. tg is the in-band talkgroup the frame belongs to,
// or 0 when this frame carries no talkgroup (P25 LDU2, FM, protocols
// without per-frame TG) — in which case the previous match decision is
// inherited. When the grant has no talkgroup (grantTG == 0) gating is
// disabled and everything matches.
func (bt *boundaryTracker) onVoice(tg uint32) bool {
	if bt.grantTG == 0 {
		bt.lastMatch = true
	} else if tg != 0 {
		bt.lastMatch = tg == bt.grantTG || bt.patches[tg]
		if bt.lastMatch {
			bt.foreignRun = 0
		} else {
			// Only count a sustained run of the SAME foreign talkgroup;
			// a different value restarts the debounce so a lone mis-decode
			// can't tip the count over the edge.
			if tg != bt.lastForeignTG {
				bt.lastForeignTG = tg
				bt.foreignRun = 0
			}
			bt.foreignRun++
			if bt.foreignRun >= foreignRunToEnd {
				// A different talkgroup has taken this (shared) frequency;
				// our call is over. The engine will start the other
				// talkgroup's call on its own tuner.
				bt.end(trunking.EndReasonNormal)
			}
		}
	}
	if bt.lastMatch {
		bt.lastVoiceNano.Store(time.Now().UnixNano())
		bt.sawVoice.Store(true)
		bt.voiceSinceRoll = true
	}
	return bt.lastMatch
}

// onTransmissionEnd marks an end-of-transmission boundary. In split
// (per-transmission) mode it rolls the recording to a fresh file via a
// KindCallSegment event — but only when audio was written since the last
// roll, so a run of terminators / idle frames doesn't spawn empty files.
func (bt *boundaryTracker) onTransmissionEnd() {
	if !bt.c.splitTx || !bt.voiceSinceRoll {
		return
	}
	bt.voiceSinceRoll = false
	bt.c.bus.Publish(events.Event{
		Kind: events.KindCallSegment,
		Payload: trunking.CallSegment{
			DeviceSerial: bt.serial,
			At:           time.Now(),
		},
	})
}

// run drives the hangtime timer and throttled Touch heartbeat until ctx
// is cancelled (which the composer does when the call ends). It ends the
// call once matching voice stops arriving for hangtime — or, when no
// matching voice EVER arrives, once the no-voice startup window elapses,
// so a phantom / mis-tuned grant frees its voice tap instead of hanging
// for the engine's much longer call-timeout watchdog.
func (bt *boundaryTracker) run(ctx context.Context) {
	// Poll fast enough that the hangtime end + Touch heartbeat are
	// responsive, but coarser than a frame interval so we don't spin.
	tick := 200 * time.Millisecond
	if bt.c.touchEvery > 0 && bt.c.touchEvery < tick {
		tick = bt.c.touchEvery
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !bt.sawVoice.Load() {
				// No matching voice frame has EVER decoded. End the call
				// once the no-voice startup window elapses instead of
				// holding the tap for the full engine watchdog — this is
				// the "hanging silent call" (carrier already dropped, stale
				// CC grant, or a mis-tuned / wrong-demod channel). Reason
				// timeout: not a single frame was delivered, matching the
				// engine's silent-from-start classification.
				if time.Since(time.Unix(0, bt.startNano)) > bt.noVoiceTimeout {
					bt.end(trunking.EndReasonTimeout)
					return
				}
				continue
			}
			last := bt.lastVoiceNano.Load()
			// Keep the engine's LastHeardAt fresh: Touch once per new
			// activity (gated on progress — when voice stops, last stops
			// advancing and we stop touching, so the engine watchdog
			// backstop still sees the call go idle).
			if bt.c.engine != nil && last != bt.lastTouchNano {
				bt.c.engine.Touch(bt.serial)
				bt.lastTouchNano = last
			}
			if time.Since(time.Unix(0, last)) > bt.hangtime {
				bt.end(trunking.EndReasonNormal)
				return
			}
		}
	}
}

// observe accumulates the received channel power of one IQ chunk into the
// call's running signal-level meter. Called from the single chain goroutine
// on every IQ chunk (before decode), so it must stay cheap. The running sum
// stays chain-goroutine-local; only the per-chunk dBFS snapshot is published
// to the atomics end() reads.
//
// SignalDbFS = 10*log10(mean(|iq|²) + eps) is a channel-power / RSSI-style
// figure in dBFS (0 = digital full scale; real signals negative). It is NOT
// calibrated absolute RSSI (no front-end gain / antenna correction) and NOT
// SNR/EVM (no noise-floor reference). Being a mean of |iq|² it is
// sample-rate-invariant, so it is comparable across the ~48 kHz wideband
// virtual voice tap and a physical-SDR voice tuner alike.
func (bt *boundaryTracker) observe(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	for _, s := range iq {
		re := float64(real(s))
		im := float64(imag(s))
		bt.sumSq += re*re + im*im
	}
	bt.nSamp += uint64(len(iq))
	if bt.nSamp == 0 {
		return
	}
	dbfs := 10 * math.Log10(bt.sumSq/float64(bt.nSamp)+1e-12)
	bt.meanDbFSBits.Store(math.Float64bits(dbfs))
	bt.haveSignal.Store(true)
}

// signalDbFS returns the call's mean received channel power in dBFS and
// whether any IQ was measured. Safe to call from any goroutine.
func (bt *boundaryTracker) signalDbFS() (float64, bool) {
	if !bt.haveSignal.Load() {
		return 0, false
	}
	return math.Float64frombits(bt.meanDbFSBits.Load()), true
}

// observeSoft accumulates a batch of C4FM 4-level soft symbols (the
// FM-discriminator path's post-matched-filter symbol values) for the call's
// EVM/SNR estimate, capped at the warm-up prefix plus demodMaxSymbols. Called
// from the chain goroutine via the receiver's SoftSink.
func (bt *boundaryTracker) observeSoft(soft []float32) {
	if len(soft) == 0 {
		return
	}
	bt.demodMu.Lock()
	if room := demodWarmupSkip + demodMaxSymbols - len(bt.softBuf); room > 0 {
		if len(soft) > room {
			soft = soft[:room]
		}
		bt.softBuf = append(bt.softBuf, soft...)
	}
	bt.demodMu.Unlock()
}

// observeSymbols accumulates a batch of complex constellation points (the
// linear CQPSK/LSM path) for the call's EVM/SNR estimate. Called from the chain
// goroutine via the receiver's SymbolSink.
func (bt *boundaryTracker) observeSymbols(pts []complex64) {
	if len(pts) == 0 {
		return
	}
	bt.demodMu.Lock()
	if room := demodWarmupSkip + demodMaxSymbols - len(bt.symBuf); room > 0 {
		if len(pts) > room {
			pts = pts[:room]
		}
		bt.symBuf = append(bt.symBuf, pts...)
	}
	bt.demodMu.Unlock()
}

// demodMetrics computes the call's RMS EVM (%) and estimated SNR (dB) from the
// buffered demod symbols, mirroring internal/siglab's demodMetrics: the complex
// constellation (CQPSK/linear) when the receiver populated it, otherwise the
// 4-level C4FM soft eye. Returns ok=false until enough settled (post-warmup)
// symbols accumulated for a stable estimate, or when no soft/symbol tap fed the
// tracker at all (every protocol but P25 Phase 1 today). Safe to call from any
// goroutine.
func (bt *boundaryTracker) demodMetrics() (evmPct, snrDB float64, ok bool) {
	bt.demodMu.Lock()
	defer bt.demodMu.Unlock()
	if len(bt.symBuf) >= demodWarmupSkip+demodMinSymbols {
		pts := bt.symBuf[demodWarmupSkip:]
		return metrics.EVMConstellation(pts), capDemodSNR(metrics.SNRM2M4Constellation(pts)), true
	}
	if len(bt.softBuf) >= demodWarmupSkip+demodMinSymbols {
		soft := bt.softBuf[demodWarmupSkip:]
		outer := metrics.EstimateOuterRailC4FM(soft)
		if outer <= 0 {
			return 0, 0, false
		}
		return metrics.EVMC4FM(soft, outer), capDemodSNR(metrics.SNRResidualC4FM(soft, outer)), true
	}
	return 0, 0, false
}

// capDemodSNR clamps an SNR estimate to ±demodSNRCapDB so a noise-free residual
// (SNR → ±Inf) stays finite and JSON-marshalable, mirroring siglab's capSNR.
func capDemodSNR(db float64) float64 {
	switch {
	case math.IsInf(db, 1) || db > demodSNRCapDB:
		return demodSNRCapDB
	case math.IsInf(db, -1) || db < -demodSNRCapDB:
		return -demodSNRCapDB
	}
	return db
}

// end ends the call exactly once via the engine, which publishes
// CallEnd; the composer then cancels this chain's context.
func (bt *boundaryTracker) end(reason trunking.EndReason) {
	bt.endOnce.Do(func() {
		if bt.c.engine == nil {
			return
		}
		// Stamp the measured signal level onto the bound call before the
		// engine publishes CallEnd. Same-goroutine ordering guarantees the
		// stamp lands first; calls ended by non-composer paths (watchdog,
		// preemption, shutdown) carry no measurement and read back NULL.
		if d, ok := bt.signalDbFS(); ok {
			bt.c.engine.UpdateSignal(bt.serial, d)
		}
		// Same for the demod-quality (EVM/SNR) figure — the number an operator
		// compares against SDRTrunk. Only P25 Phase 1 feeds the soft/symbol
		// taps today, so other protocols carry no measurement (NULL).
		if evm, snr, ok := bt.demodMetrics(); ok {
			bt.c.engine.UpdateDemod(bt.serial, evm, snr)
		}
		bt.c.engine.EndCall(bt.serial, reason)
	})
}
