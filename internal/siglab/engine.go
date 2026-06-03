package siglab

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
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

// RunStream is Run with a live per-event sink: onEvent (when non-nil) is
// called for every captured EventRecord as it is observed, in stream order.
// It backs the JSONL exporter and the TUI's live event feed. The full
// Result is still returned at EOF.
func RunStream(path string, cfg Config, onEvent func(EventRecord)) (*Result, error) {
	if cfg.SampleRateHz <= 0 {
		return nil, fmt.Errorf("siglab: sample rate must be > 0")
	}
	if !ccdecoder.HasFactory(cfg.Protocol) {
		return nil, fmt.Errorf("siglab: no pipeline registered for protocol %s", cfg.Protocol)
	}

	decode, bytesPerSample := cfg.Format.Decoder()

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// Resolve the tuning offset (auto-tune estimates from a prefix then
	// rewinds; manual TuneHz is the override).
	tuneHz := cfg.TuneHz
	if cfg.AutoTune {
		est, terr := estimateCaptureCarrierHz(f, decode, bytesPerSample, cfg.SampleRateHz)
		if terr != nil {
			return nil, fmt.Errorf("auto-tune failed: %w", terr)
		}
		tuneHz = est
	}

	res, err := runReader(f, path, decode, bytesPerSample, tuneHz, cfg, onEvent)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// runReader is the format-agnostic core: it mirrors the daemon's DDC →
// pipeline chain (so a replay lock implies an on-air lock), drains the bus
// into a Result, and runs the optional analyzer. Split out so a future
// in-memory source (synthesized IQ) can share it.
func runReader(r io.Reader, source string, decode SampleDecoder, bytesPerSample int, tuneHz float64, cfg Config, onEvent func(EventRecord)) (*Result, error) {
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
	var an *analyzer
	if cfg.CollectIQDiag {
		an = newAnalyzer()
		an.bufferSymbols = cfg.Protocol == trunking.ProtocolP25
	}
	var iqCorrector *rtlsdr.IQImbalanceCorrector
	if cfg.IQCorrect {
		iqCorrector = rtlsdr.NewIQImbalanceCorrector()
	}

	var symbolCount int64
	pipe, ok, err := ccdecoder.NewPipeline(cfg.Protocol, ccdecoder.PipelineOptions{
		Bus:          bus,
		Log:          logger,
		SystemName:   cfg.SystemName,
		FrequencyHz:  cfg.FrequencyHz,
		SampleRateHz: receiverRate,
		System:       cfg.System,
		SymbolTap: func(symbols []uint8, isBits bool, _ int) {
			symbolCount += int64(len(symbols))
			if an != nil {
				an.observeSymbols(symbols, isBits)
			}
		},
	})
	if err != nil {
		sub.Close()
		bus.Close()
		return nil, fmt.Errorf("construct %s pipeline: %w", cfg.Protocol, err)
	}
	if !ok { // guarded above, but keep the invariant explicit
		sub.Close()
		bus.Close()
		return nil, fmt.Errorf("siglab: no pipeline registered for protocol %s", cfg.Protocol)
	}
	defer pipe.Close()

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
			pipe.Process(feed)
			totalSamples += int64(pairs)
		}
		if errors.Is(rerr, io.EOF) || errors.Is(rerr, io.ErrUnexpectedEOF) {
			break
		}
		if rerr != nil {
			readErr = fmt.Errorf("read %s: %w", source, rerr)
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
		res.Signal = an.result(decodeErrTotal)
		if cfg.Protocol == trunking.ProtocolP25 {
			res.P25P1 = buildP25Detail(an.symBuf)
		}
	}

	if cfg.Acceptance != nil {
		res.Verdict = evaluateAcceptance(res, cfg.Acceptance)
	}
	return res
}
