package siglab

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Run decodes the capture at path through the production pipeline for
// cfg.Protocol and returns the structured Result. It is the batch entry
// point behind the replay/analyze/test subcommands.
func Run(path string, cfg Config) (*Result, error) {
	return RunStream(path, cfg, nil)
}

// RunAutoTuneMulti opens the capture at path and decodes cfg.Protocol against
// each detected carrier candidate, returning the strongest lock (see
// RunReaderAutoTuneMulti). It is the file-path entry the replay/analyze
// subcommands use for multi-candidate -auto-tune.
func RunAutoTuneMulti(path string, cfg Config, maxCandidates int) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return RunReaderAutoTuneMulti(f, path, cfg, maxCandidates)
}

// RunStream is Run with a live per-event sink: onEvent (when non-nil) is
// called for every captured EventRecord as it is observed, in stream order.
// It backs the JSONL exporter and the TUI's live event feed. The full
// Result is still returned at EOF.
func RunStream(path string, cfg Config, onEvent func(EventRecord)) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return RunReaderStream(f, path, cfg, onEvent)
}

// RunReader decodes the capture read from r through the production pipeline
// for cfg.Protocol and returns the structured Result. source is recorded as
// the Result's provenance (its base name). It is the in-memory sibling of
// Run: a synthesized signal, an uploaded HTTP body, or any io.Reader can be
// analyzed without a disk round-trip. AutoTune requires r to also satisfy
// io.ReadSeeker (an *os.File or *bytes.Reader does).
func RunReader(r io.Reader, source string, cfg Config) (*Result, error) {
	return RunReaderStream(r, source, cfg, nil)
}

// RunReaderStream is RunReader with a live per-event sink (see RunStream).
// It is the single decode path RunStream and the in-memory callers share.
func RunReaderStream(r io.Reader, source string, cfg Config, onEvent func(EventRecord)) (*Result, error) {
	if cfg.SampleRateHz <= 0 {
		return nil, fmt.Errorf("siglab: sample rate must be > 0")
	}
	if !ccdecoder.HasFactory(cfg.Protocol) {
		return nil, fmt.Errorf("siglab: no pipeline registered for protocol %s", cfg.Protocol)
	}

	decode, bytesPerSample := cfg.Format.Decoder()

	// Resolve the tuning offset (auto-tune estimates from a prefix then
	// rewinds; manual TuneHz is the override). Auto-tune needs to seek the
	// source back to the start after reading its prefix.
	tuneHz := cfg.TuneHz
	if cfg.AutoTune {
		rs, ok := r.(io.ReadSeeker)
		if !ok {
			return nil, fmt.Errorf("siglab: auto-tune requires a seekable source")
		}
		est, terr := estimateCaptureCarrierHz(rs, decode, bytesPerSample, cfg.SampleRateHz)
		if terr != nil {
			return nil, fmt.Errorf("auto-tune failed: %w", terr)
		}
		tuneHz = est
	}

	return runReader(r, source, decode, bytesPerSample, tuneHz, cfg, onEvent, nil)
}

// monitorHook drives converge-and-stop dwell on a streaming source. Every
// Interval of stream time the read loop calls OnTick with the current
// accumulated topology and the elapsed stream seconds; returning true ends the
// run early (the Result assembled so far is returned as usual).
type monitorHook struct {
	Interval time.Duration
	OnTick   func(topo *TopologySnapshot, streamSec float64) (stop bool)
}

// tickSec is the interval in seconds; nil-safe so the read loop can call it
// unconditionally before checking whether monitoring is active.
func (m *monitorHook) tickSec() float64 {
	if m == nil || m.Interval <= 0 {
		return 0
	}
	return m.Interval.Seconds()
}

// RunReaderMonitor decodes a (typically live, effectively unbounded) stream,
// invoking onTick every tick of stream time with the current accumulated
// topology so the caller can converge-and-stop a long control-channel dwell.
// It is the streaming sibling of RunReader: memory stays bounded (chunked
// reads, nothing is buffered whole), and the run ends when onTick returns
// true, the reader hits EOF, or a read errors. The carrier must already be
// tuned via cfg.TuneHz — auto-tune (which needs to seek) is not supported.
func RunReaderMonitor(r io.Reader, source string, cfg Config, tick time.Duration, onTick func(*TopologySnapshot, float64) bool) (*Result, error) {
	if cfg.SampleRateHz <= 0 {
		return nil, fmt.Errorf("siglab: sample rate must be > 0")
	}
	if !ccdecoder.HasFactory(cfg.Protocol) {
		return nil, fmt.Errorf("siglab: no pipeline registered for protocol %s", cfg.Protocol)
	}
	if cfg.AutoTune {
		return nil, fmt.Errorf("siglab: RunReaderMonitor does not support auto-tune (tune via cfg.TuneHz)")
	}
	if tick <= 0 {
		tick = time.Second
	}
	decode, bytesPerSample := cfg.Format.Decoder()
	mon := &monitorHook{Interval: tick, OnTick: onTick}
	return runReader(r, source, decode, bytesPerSample, cfg.TuneHz, cfg, nil, mon)
}

// RunReaderAutoTuneMulti decodes cfg.Protocol against each detected carrier
// candidate (each tuned to a fixed offset, resolved here rather than
// re-estimated) and returns the Result with the strongest lock. It is the
// multi-candidate counterpart to a single -auto-tune run: when the control
// channel is off-centre AND not the loudest carrier in a wideband capture, the
// single dominant-carrier estimate misses it; this tries the ranked candidates
// strongest-first and stops at the first that locks (or returns the most
// promising failure). r must be an io.ReadSeeker; cfg.AutoTune is implied and
// cfg.TuneHz is ignored. maxCandidates ≤ 0 uses the package default.
func RunReaderAutoTuneMulti(r io.ReadSeeker, source string, cfg Config, maxCandidates int) (*Result, error) {
	if cfg.SampleRateHz <= 0 {
		return nil, fmt.Errorf("siglab: sample rate must be > 0")
	}
	if !ccdecoder.HasFactory(cfg.Protocol) {
		return nil, fmt.Errorf("siglab: no pipeline registered for protocol %s", cfg.Protocol)
	}
	decode, bytesPerSample := cfg.Format.Decoder()
	cands, err := estimateCaptureCarrierCandidates(r, decode, bytesPerSample, cfg.SampleRateHz, maxCandidates)
	if err != nil {
		return nil, fmt.Errorf("auto-tune failed: %w", err)
	}
	offsets := make([]float64, 0, len(cands))
	for _, c := range cands {
		offsets = append(offsets, c.OffsetHz)
	}
	if len(offsets) == 0 {
		offsets = []float64{0}
	}

	runCfg := cfg
	runCfg.AutoTune = false
	var best *Result
	for _, off := range offsets {
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("siglab: auto-tune rewind: %w", err)
		}
		runCfg.TuneHz = off
		res, err := runReader(r, source, decode, bytesPerSample, off, runCfg, nil, nil)
		if err != nil {
			continue
		}
		if best == nil || betterLock(res, best) {
			best = res
		}
		if best.Locked {
			break // strongest-first; the first carrier that locks is the answer
		}
	}
	if best == nil {
		return nil, fmt.Errorf("siglab: auto-tune produced no decodable candidate")
	}
	return best, nil
}

// betterLock reports whether a is a stronger decode than b, for choosing the
// best carrier among auto-tune candidates: a real lock beats no lock; ties
// break on control-channel yield (P25 NID-trusted + TSBK), then grants, then
// symbols processed, then faster lock.
func betterLock(a, b *Result) bool {
	if a.Locked != b.Locked {
		return a.Locked
	}
	if ya, yb := ccYield(a), ccYield(b); ya != yb {
		return ya > yb
	}
	if len(a.Grants) != len(b.Grants) {
		return len(a.Grants) > len(b.Grants)
	}
	if a.Symbols != b.Symbols {
		return a.Symbols > b.Symbols
	}
	if a.Locked && b.Locked {
		return a.LockLatencySec < b.LockLatencySec
	}
	return false
}

// ccYield is a coarse control-channel decode-yield metric (P25 NID-trusted +
// decoded TSBK count); 0 for protocols without a P25 Phase 1 detail.
func ccYield(r *Result) int64 {
	if d, ok := r.Detail.(*P25P1Detail); ok && d.CCStats != nil {
		return d.CCStats.NIDTrusted + d.CCStats.TSBKDecoded
	}
	return 0
}

// runReader is the format-agnostic core: it mirrors the daemon's DDC →
// pipeline chain (so a replay lock implies an on-air lock), drains the bus
// into a Result, and runs the optional analyzer. Split out so a future
// in-memory source (synthesized IQ) can share it.
func runReader(r io.Reader, source string, decode SampleDecoder, bytesPerSample int, tuneHz float64, cfg Config, onEvent func(EventRecord), mon *monitorHook) (*Result, error) {
	logger := cfg.logger()

	// Mirror the production ccdecoder DDC: decimate to the per-protocol
	// channel rate (TETRA → 144 kHz, else 48 kHz) whenever the input is
	// wider, and apply the tuning shift. An already-channelized capture
	// (rate ≤ target) with no tuning passes straight through.
	target := ccdecoder.DDCTargetForProtocol(cfg.Protocol)
	ddcTarget := cfg.SampleRateHz
	if cfg.SampleRateHz > target {
		ddcTarget = target
	}
	var ddc *ccdecoder.Downconverter
	receiverRate := cfg.SampleRateHz
	if tuneHz != 0 || ddcTarget < cfg.SampleRateHz {
		ddc = ccdecoder.NewDownconverterWithOffset(cfg.SampleRateHz, ddcTarget, tuneHz)
		receiverRate = ddc.OutRateHz()
	}

	bus := events.NewBus(1024)
	sub := bus.Subscribe()

	// Optional protocol-agnostic analyzer + raw-IQ imbalance corrector.
	deep := cfg.wantP25Deep()
	var an *analyzer
	if cfg.CollectIQDiag {
		an = newAnalyzer()
		// Buffer the recovered symbols when a protocol-specific detail builder
		// needs the full stream: P25 (its deep path) or any protocol with a
		// registered symbol-domain detail spec.
		an.bufferSymbols = cfg.Protocol == trunking.ProtocolP25 || hasDetailSpec(cfg.Protocol)
	}
	var iqCorrector *rtlsdr.IQImbalanceCorrector
	if cfg.IQCorrect {
		iqCorrector = rtlsdr.NewIQImbalanceCorrector()
	}

	// Optional decimated-IQ + soft-sample capture for the visualization
	// consumers (constellation / spectrogram / PSD / eye). Strided off the
	// channelized stream and hard-capped, so it is bounded and never
	// full-rate.
	var iqTap *iqTapBuffer
	if cfg.CaptureIQ {
		iqTap = newIQTapBuffer(receiverRate, cfg.captureIQMaxPoints())
	}

	var symbolCount int64
	symbolTap := func(symbols []uint8, isBits bool, _ int) {
		symbolCount += int64(len(symbols))
		if an != nil {
			an.observeSymbols(symbols, isBits)
		}
	}
	softTap := func(soft []float32) {
		if an != nil {
			an.observeSoft(soft)
		}
		if iqTap != nil {
			iqTap.observeSoft(soft)
		}
	}
	// constTap carries the complex symbol-decision points (CQPSK / linear
	// path) into the analyzer so the demod-quality metrics can use the
	// constellation estimators. Only the P25 deep path's CQPSK mode fires it.
	constTap := func(pts []complex64) {
		if an != nil {
			an.observeConstellation(pts)
		}
	}

	// Optional per-signaling-block (TSBK) dissection for the data-PDU inspector
	// — P25 deep path only.
	var pduColl *pduCollector
	var pduTap func(p25phase1.SignalingBlock)
	if cfg.CollectPDUs {
		pduColl = newPDUCollector(expectedSymbolRate(cfg.Protocol))
		pduTap = pduColl.observeP25
	}

	// Build the run bundle: either a generic factory pipeline (any protocol)
	// or the P25 Phase 1 deep path (direct receiver + control channel with
	// soft/state capture and the experimental bisect knobs).
	bundle, err := buildBundle(cfg, bus, logger, receiverRate, symbolTap, softTap, constTap, pduTap)
	if err != nil {
		sub.Close()
		bus.Close()
		return nil, err
	}
	if bundle.closeFn != nil {
		defer bundle.closeFn()
	}

	// Drain the bus in the background so events are collected as they fire.
	start := time.Now()
	coll := newCollector(start, onEvent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range sub.C {
			coll.observe(ev)
		}
	}()

	// Per-(stream-)second receiver-state capture (deep P25 path only).
	const stateLogIntervalSec = 1.0
	var nextStateLogAt = stateLogIntervalSec
	var states []ReceiverState
	captureState := bundle.stateAt != nil && cfg.CollectReceiverState

	// Converge-and-stop monitoring (streaming long-dwell): fire the tick on
	// stream time so the caller sees the same cadence however fast replay runs.
	monitoring := mon != nil && mon.OnTick != nil && bundle.topology != nil
	nextMonitorAt := mon.tickSec()

	chunk := cfg.chunkSamples()
	buf := make([]byte, chunk*bytesPerSample)
	samples := make([]complex64, chunk)
	var ddcOut []complex64
	var totalSamples int64
	var readErr error
	for {
		n, rerr := io.ReadFull(r, buf)
		if n > 0 {
			pairs := n / bytesPerSample
			if pairs > len(samples) {
				samples = make([]complex64, pairs)
			}
			decode(buf[:pairs*bytesPerSample], samples[:pairs])
			feed := samples[:pairs]
			if cfg.Conjugate {
				for i, s := range feed {
					feed[i] = complex(real(s), -imag(s))
				}
			}
			if an != nil {
				an.observeIQ(feed)
			}
			if iqCorrector != nil {
				iqCorrector.Process(feed)
			}
			if ddc != nil {
				ddcOut = ddc.Process(ddcOut, feed)
				feed = ddcOut
			}
			if iqTap != nil {
				iqTap.observeIQ(feed)
			}
			bundle.proc.Process(feed)
			totalSamples += int64(pairs)

			// State snapshots are throttled on IQ-stream time (totalSamples /
			// input rate), so a given capture yields the same series however
			// fast replay chews through it.
			if captureState {
				if t := float64(totalSamples) / cfg.SampleRateHz; t >= nextStateLogAt {
					states = append(states, bundle.stateAt(t))
					nextStateLogAt = t + stateLogIntervalSec
				}
			}

			// Converge-and-stop: hand the caller the live topology on the tick
			// cadence; a true return ends the dwell early.
			if monitoring {
				if t := float64(totalSamples) / cfg.SampleRateHz; t >= nextMonitorAt {
					nextMonitorAt = t + mon.tickSec()
					if mon.OnTick(bundle.topology(), t) {
						break
					}
				}
			}
		}
		if errors.Is(rerr, io.EOF) || errors.Is(rerr, io.ErrUnexpectedEOF) {
			break
		}
		if rerr != nil {
			readErr = fmt.Errorf("read %s: %w", source, rerr)
			break
		}
		// Prefix cap (identifier fast scan): stop once enough input samples
		// have been processed. Checked after a whole chunk, so it may
		// overshoot by up to one chunk — fine for a bounded scan.
		if cfg.MaxSamples > 0 && totalSamples >= cfg.MaxSamples {
			break
		}
	}

	// Release the drainer and wait for it so totals are complete.
	bus.Close()
	<-done
	sub.Close()
	if readErr != nil {
		return nil, readErr
	}

	res := assembleResult(source, cfg, totalSamples, symbolCount, receiverRate, tuneHz, coll, an)

	// Attach the protocol-specific deep dive. P25 Phase 1 uses its dedicated
	// deep path (soft eye + receiver-state + native CCStats); every other
	// protocol with a registered builder gets a symbol-domain detail (sync
	// landscape + FEC tally) computed from the buffered recovered symbols.
	switch {
	case deep:
		// CCStats and the receiver-state series are captured whenever the deep
		// path is active (so `replay -protocol p25p1` shows them); the
		// dibit/soft landscape is added only when the analyzer buffered
		// symbols (CollectIQDiag).
		var d *P25P1Detail
		if an != nil {
			d = buildP25Detail(an.symBuf, an.softBuf)
		}
		if d == nil {
			d = &P25P1Detail{}
		}
		if bundle.ccStats != nil {
			d.CCStats = bundle.ccStats()
		}
		d.ReceiverStates = states
		res.Detail = d
	case an != nil && hasDetailSpec(cfg.Protocol):
		if d := buildProtocolDetail(cfg.Protocol, an.symBuf, an.cardinality == 2, cfg.System); d != nil {
			res.Detail = d
		}
	}

	// Attach accumulated system topology (identity / neighbors / band plan).
	// This is the only path that surfaces WACN/SYSID/RFSS/Site etc., which
	// never ride on event payloads.
	if bundle.topology != nil {
		if t := bundle.topology(); !t.Empty() {
			res.Topology = t
		}
	}

	if iqTap != nil {
		res.IQTaps = iqTap.result(receiverRate)
		// Attach the aligned symbol series (dibits + pre-slicer soft) for
		// the Symbol-scope viz, decimated to the same cap as the IQ. The
		// analyzer already buffers them in lockstep on the deep P25 path.
		if an != nil && len(an.symBuf) > 0 {
			d, s := decimateSymbolSeries(an.symBuf, an.softBuf, cfg.captureIQMaxPoints())
			res.IQTaps.SymbolDibits = d
			res.IQTaps.SymbolSoft = s
			res.IQTaps.SymbolCardinality = an.cardinality
		}
		// Per-symbol differential phase (the π/4-DQPSK rotation signal) for the
		// rotation-tracker viz. Populated only on the CQPSK path, which is the
		// only one that buffers complex constellation points.
		if an != nil && len(an.constBuf) > 1 {
			res.IQTaps.DiffPhase = diffPhaseSeries(an.constBuf, cfg.captureIQMaxPoints())
		}
	}

	// Surface the receiver's settled carrier-frequency error onto the VSA demod
	// metrics (deep path only; the generic factory path exposes no AFC loop).
	if bundle.carrierErrHz != nil && res.Signal != nil && res.Signal.Demod != nil {
		res.Signal.Demod.CarrierFreqErrorHz = bundle.carrierErrHz()
	}

	// Attach the per-signaling-block dissection (data-PDU inspector feed).
	if pduColl != nil {
		res.PDUs = pduColl.recs
		res.PDUsTruncated = pduColl.truncated
	}

	// Name *why* the control channel did not lock (SNR-limited / misaligned /
	// not-control / no-signal), from the demod + frame-decode metrics now
	// attached — so a capture-quality failure is not misread as a tuning or AGC
	// bug (issues #764 / #771). No-op when locked.
	res.NoLockReason = classifyNoLockReason(res)
	return res, nil
}

// assembleResult builds the final Result from the read-loop totals and the
// drained collector.
func assembleResult(source string, cfg Config, totalSamples, symbols int64, receiverRate, tuneHz float64, coll *collector, an *analyzer) *Result {
	duration := 0.0
	if cfg.SampleRateHz > 0 {
		duration = float64(totalSamples) / cfg.SampleRateHz
	}
	res := &Result{
		Source:         filepath.Base(source),
		Protocol:       cfg.Protocol.String(),
		SampleRateHz:   cfg.SampleRateHz,
		PipelineRateHz: receiverRate,
		TuneHz:         tuneHz,
		DurationSec:    duration,
		TotalSamples:   totalSamples,
		Symbols:        symbols,
		ExpectedBaud:   expectedSymbolRate(cfg.Protocol),
		Locked:         coll.locked,
		Lock:           coll.lock,
		Grants:         coll.grants,
		Events:         coll.events,
		EventCounts:    coll.eventCounts,
		DecodeErrors:   coll.decodeErr,
	}
	if coll.locked {
		res.LockLatencySec = coll.lockLatency
	}
	if res.Grants == nil {
		res.Grants = []GrantRecord{}
	}
	if res.Events == nil {
		res.Events = []EventRecord{}
	}
	if duration > 0 && symbols > 0 {
		res.EffectiveBaud = float64(symbols) / duration
		if res.ExpectedBaud > 0 {
			res.BaudDeviationPct = (res.EffectiveBaud - res.ExpectedBaud) / res.ExpectedBaud * 100
		}
	}

	// Total decode errors across stages → analyzer error rate.
	var decodeErrTotal int64
	for _, n := range coll.decodeErr {
		decodeErrTotal += int64(n)
	}
	if an != nil {
		res.Signal = an.result(decodeErrTotal, cfg.captureIQMaxPoints())
	}

	if cfg.Acceptance != nil {
		res.Verdict = evaluateAcceptance(res, cfg.Acceptance)
	}
	return res
}

// processor is the minimal surface the read loop drives: push a chunk of
// channelized IQ. Both ccdecoder.ProtocolPipeline and the directly-built P25
// receiver satisfy it.
type processor interface {
	Process(iq []complex64)
}

// runBundle is what the read loop drives plus the optional deep-path hooks.
// stateAt/ccStats are nil for the generic factory path. topology is set when
// the pipeline can report accumulated system topology (P25 deep path always;
// any factory pipeline implementing TopologyProvider).
type runBundle struct {
	proc     processor
	closeFn  func()
	stateAt  func(t float64) ReceiverState
	ccStats  func() *CCStatsBreakdown
	topology func() *TopologySnapshot
	// carrierErrHz reports the receiver's settled residual carrier-frequency
	// error (Hz) for the VSA metrics. Nil on the generic factory path.
	carrierErrHz func() float64
}

// buildBundle constructs the run bundle for cfg: the P25 Phase 1 deep path
// when requested, otherwise the generic factory pipeline (any protocol). Both
// publish lock/grant/decode-error events to bus and feed symbolTap; only the
// deep path feeds softTap and exposes state/ccStats.
func buildBundle(cfg Config, bus *events.Bus, logger *slog.Logger, receiverRate float64, symbolTap func([]uint8, bool, int), softTap func([]float32), constTap func([]complex64), pduTap func(p25phase1.SignalingBlock)) (*runBundle, error) {
	if cfg.wantP25Deep() {
		return buildP25DeepBundle(cfg, bus, logger, receiverRate, symbolTap, softTap, constTap, pduTap)
	}
	pipe, ok, err := ccdecoder.NewPipeline(cfg.Protocol, ccdecoder.PipelineOptions{
		Bus:          bus,
		Log:          logger,
		SystemName:   cfg.SystemName,
		FrequencyHz:  cfg.FrequencyHz,
		SampleRateHz: receiverRate,
		System:       cfg.System,
		SymbolTap:    symbolTap,
	})
	if err != nil {
		return nil, fmt.Errorf("construct %s pipeline: %w", cfg.Protocol, err)
	}
	if !ok {
		return nil, fmt.Errorf("siglab: no pipeline registered for protocol %s", cfg.Protocol)
	}
	rb := &runBundle{proc: pipe, closeFn: func() { _ = pipe.Close() }}
	// Any factory pipeline may expose accumulated topology (identity / neighbor
	// sites). Snapshot it at EOF when the protocol implements the hook.
	if tp, ok := pipe.(TopologyProvider); ok {
		rb.topology = tp.TopologySnapshot
	}
	return rb, nil
}
