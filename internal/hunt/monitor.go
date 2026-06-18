package hunt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Streaming long-dwell control-channel monitor.
//
// The buffered per-candidate path (RunLiveHunt's default) captures the entire
// dwell into memory before decoding — fine for a 3 s grab, but a 5–10 min CC
// monitor would buffer gigabytes of IQ. monitorCandidate instead streams live
// IQ straight into the siglab decoder (bounded memory) and stops early once
// the accumulated topology (identity + neighbors + band plan) settles, capped
// at MonitorSeconds. It is the streaming sibling of decodeAndAccumulate.

const (
	// monitorTick is how often the converge-and-stop check runs (stream time).
	monitorTick = 2 * time.Second
	// monitorStability is how long the topology must stop changing before the
	// dwell is considered settled.
	monitorStability = 30 * time.Second
	// monitorMinDwell is the minimum dwell before convergence can trigger, so a
	// site's slower-cycling broadcasts (adjacent sites, full band plan) get a
	// chance to arrive before we declare the picture complete.
	monitorMinDwell = 20 * time.Second
)

// streamReader adapts an IQSource into an io.Reader of encoded samples for the
// siglab decoder. Each Read pulls a chunk from the source and serves it in the
// requested sample format, buffering the remainder between calls. It returns
// io.EOF when ctx is done (the dwell cap) or the source ends, so the decoder
// assembles and returns the topology accumulated so far.
type streamReader struct {
	ctx    context.Context
	src    IQSource
	chunk  int
	format siglab.SampleFormat
	buf    []byte
	done   bool
}

func (s *streamReader) Read(p []byte) (int, error) {
	for len(s.buf) == 0 {
		if s.done {
			return 0, io.EOF
		}
		if s.ctx.Err() != nil {
			s.done = true
			return 0, io.EOF
		}
		iq, err := s.src.Capture(s.ctx, s.chunk)
		if err != nil || len(iq) == 0 {
			// ctx cancellation, deadline, or end-of-stream — end the dwell
			// gracefully; the decoder still returns what it accumulated.
			s.done = true
			return 0, io.EOF
		}
		s.buf = siglab.EncodeCapture(iq, s.format)
	}
	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	return n, nil
}

// monitorCandidate tunes the source at one control-channel frequency, captures
// a short prefix to identify the protocol (unless one is forced), then streams
// the live source into the decoder for up to monitorSeconds, folding the
// accumulated system into sys. It mirrors decodeAndAccumulate's identify gate
// and accumulation, but decodes a continuing stream instead of a fixed buffer.
func monitorCandidate(ctx context.Context, sys *DiscoveredSystem, src IQSource, freqHz uint32, prefixSamples int, monitorSeconds float64, p decodeParams) CaptureReport {
	log := p.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	source := fmt.Sprintf("%.4f MHz", float64(freqHz)/1e6)
	rep := CaptureReport{Path: source}

	if err := src.Tune(freqHz); err != nil {
		rep.Error = fmt.Sprintf("tune: %v", err)
		return rep
	}

	// Short prefix for identify + carrier offset (same gate as the buffered path).
	prefixIQ, err := captureN(ctx, src, prefixSamples)
	if err != nil {
		rep.Error = fmt.Sprintf("prefix capture: %v", err)
		return rep
	}
	prefixBytes := siglab.EncodeCapture(prefixIQ, p.Format)

	proto := p.Protocol
	conf := 1.0
	tuneHz := 0.0
	if proto == trunking.ProtocolUnknown {
		minConf := p.MinConfidence
		if minConf <= 0 {
			minConf = 0.40
		}
		idr, err := siglab.IdentifyReader(bytes.NewReader(prefixBytes), source, siglab.IdentifyConfig{
			SampleRateHz: p.SampleRateHz,
			Format:       p.Format,
			AutoTune:     p.AutoTune,
			Conjugate:    p.Conjugate,
			IQCorrect:    p.IQCorrect,
			MaxSamples:   p.IdentifyMaxSamples,
			Log:          log,
		})
		if err != nil {
			rep.Error = fmt.Sprintf("identify: %v", err)
			return rep
		}
		conf = idr.Confidence
		if idr.Inconclusive || idr.Confidence < minConf {
			rep.Protocol = idr.Winner
			rep.Confidence = idr.Confidence
			rep.Skipped = true
			rep.SkipReason = fmt.Sprintf("no trunked control channel above confidence %.2f (best: %s @ %.2f)",
				minConf, idr.Winner, idr.Confidence)
			return rep
		}
		wp, err := idr.WinnerProtocol()
		if err != nil {
			rep.Error = fmt.Sprintf("winner protocol: %v", err)
			return rep
		}
		proto = wp
		tuneHz = idr.TuneHz
	}
	rep.Protocol = proto.String()
	rep.Confidence = conf

	// Stream the prefix followed by the continuing live source. The dwell is
	// capped by monitorSeconds; convergence may stop it sooner.
	dwellCtx, cancel := context.WithTimeout(ctx, time.Duration(monitorSeconds*float64(time.Second)))
	defer cancel()
	live := &streamReader{ctx: dwellCtx, src: src, chunk: prefixSamples, format: p.Format}
	r := io.MultiReader(bytes.NewReader(prefixBytes), live)

	cfg := siglab.Config{
		Protocol:     proto,
		FrequencyHz:  freqHz,
		SampleRateHz: p.SampleRateHz,
		Format:       p.Format,
		AutoTune:     false,
		TuneHz:       tuneHz,
		Conjugate:    p.Conjugate,
		IQCorrect:    p.IQCorrect,
		// CollectIQDiag engages the P25 Phase 1 deep path that snapshots the
		// system topology (WACN/SYSID/RFSS/Site + neighbors + band plan); the
		// generic pipeline would only yield a NAC. Matches decodeAndAccumulate.
		CollectIQDiag: true,
		Log:           log,
	}

	res, err := siglab.RunReaderMonitor(r, source, cfg, monitorTick, convergeAndStop())
	if err != nil {
		rep.Error = fmt.Sprintf("monitor decode: %v", err)
		return rep
	}

	before := len(sys.Talkgroups)
	Accumulate(sys, Observation{
		Protocol:       proto.String(),
		Confidence:     conf,
		Result:         res,
		FallbackFreqHz: freqHz,
		At:             time.Now(),
	})
	rep.Locked = res.Locked
	if res.Lock != nil {
		rep.ControlHz = res.Lock.FrequencyHz
	}
	if res.Signal != nil {
		rep.ErrorRate = res.Signal.DecodeErrorRate
	}
	rep.Talkgroups = len(sys.Talkgroups) - before
	return rep
}

// convergeAndStop returns a monitor tick callback that stops the dwell once
// identity is present and the topology (neighbors / secondary CCs / band-plan
// slots) has stopped growing for monitorStability, after a monitorMinDwell
// floor. The returned closure carries the per-run change-tracking state.
func convergeAndStop() func(*siglab.TopologySnapshot, float64) bool {
	var (
		lastChangeSec        float64
		prevN, prevS, prevBP int
		haveID               bool
		seenChange           bool
	)
	stability := monitorStability.Seconds()
	minDwell := monitorMinDwell.Seconds()
	return func(topo *siglab.TopologySnapshot, sec float64) bool {
		if topo == nil {
			return false
		}
		n, s, bp := len(topo.Neighbors), len(topo.Secondary), len(topo.BandPlan)
		id := topo.WACN != 0 && topo.SystemID != 0
		if !seenChange || n != prevN || s != prevS || bp != prevBP || id != haveID {
			prevN, prevS, prevBP, haveID, seenChange = n, s, bp, id, true
			lastChangeSec = sec
		}
		if !id || sec < minDwell {
			return false
		}
		return sec-lastChangeSec >= stability
	}
}
