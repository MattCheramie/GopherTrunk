package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// spectrumProvider implements api.SpectrumProvider on top of the
// daemon's iqtap broker map. Each open stream subscribes a fresh
// observer to the requested device's broker and runs a per-request
// spectrum.Producer that reads IQ chunks, runs windowed FFTs at the
// negotiated frame rate, and writes api.SpectrumFrame onto the
// returned channel. A cleanup func tears down the subscription and
// the producer's goroutine.
//
// Per-request producers (one FFT plan per WS client) keep the design
// simple and let each client pick its own FFT size / frame rate. CPU
// scales linearly with #clients, which is fine for the single- or
// few-operator deployments GopherTrunk targets — a typical browser
// session sums to one open stream at any time. A future optimization
// could share a producer across clients with the same (device, size,
// fps) tuple, but that adds enough bookkeeping to defer until profile
// data shows the wins.
type spectrumProvider struct {
	pool    *sdr.Pool
	brokers map[string]*iqtap.Broker
	// systems is the configured trunking systems, consulted to label
	// each device with the P25 Phase 1 modulation it is decoding so the
	// web panels can auto-select C4FM vs CQPSK.
	systems []trunking.System
	log     *slog.Logger
}

func newSpectrumProvider(pool *sdr.Pool, brokers map[string]*iqtap.Broker, systems []trunking.System, log *slog.Logger) *spectrumProvider {
	if log == nil {
		log = slog.Default()
	}
	return &spectrumProvider{pool: pool, brokers: brokers, systems: systems, log: log}
}

// Devices walks the broker map and reports each device's current
// tuning. Brokers know the most-recent SetCenterFreq / SetSampleRate
// because every setter funnels through them; the pool entry supplies
// driver / role / product metadata.
func (p *spectrumProvider) Devices() []api.SpectrumDevice {
	if p == nil || p.pool == nil || len(p.brokers) == 0 {
		return nil
	}
	entries := p.pool.Entries()
	out := make([]api.SpectrumDevice, 0, len(entries))
	for _, e := range entries {
		br, ok := p.brokers[e.Info.Serial]
		if !ok {
			continue
		}
		centerHz := br.CenterHz()
		sampleRateHz := br.SampleRateHz()
		out = append(out, api.SpectrumDevice{
			Serial:           e.Info.Serial,
			Driver:           e.Info.Driver,
			Product:          e.Info.Product,
			Role:             e.Role.String(),
			CenterHz:         centerHz,
			SampleRateHz:     sampleRateHz,
			P25Modulation:    p25ModulationFor(p.systems, centerHz, sampleRateHz),
			SymbolProto:      symbolProtoFor(p.systems, centerHz, sampleRateHz),
			ControlChannelHz: controlChannelFor(p.systems, centerHz, sampleRateHz),
		})
	}
	return out
}

// p25ModulationFor reports the P25 Phase 1 demod mode ("c4fm"/"cqpsk")
// of the system the device at (centerHz, sampleRateHz) is decoding, or
// "" when no P25 Phase 1 system applies. A system matches when one of
// its control channels falls inside the device passband
// (|cc - centre| <= sample_rate/2) — the control SDR's centre sits on a
// CC exactly, and a voice SDR's wideband front end usually still spans
// it. For P25 Phase 1 the voice channels share the control channel's
// modulation, so one per-device value is correct for both the CC and any
// followed voice call. When no CC matches but exactly one P25 Phase 1
// system is configured (the common single-system deployment), fall back
// to it.
func p25ModulationFor(systems []trunking.System, centerHz, sampleRateHz uint32) string {
	var only *trunking.System
	count := 0
	for i := range systems {
		if systems[i].Protocol != trunking.ProtocolP25 {
			continue
		}
		count++
		only = &systems[i]
		half := int64(sampleRateHz) / 2
		for _, cc := range systems[i].ControlChannels {
			delta := int64(cc) - int64(centerHz)
			if delta < 0 {
				delta = -delta
			}
			if delta <= half {
				return demodModeName(systems[i].P25Phase1DemodMode)
			}
		}
	}
	if count == 1 {
		return demodModeName(only.P25Phase1DemodMode)
	}
	return ""
}

// controlChannelFor reports the configured P25 control channel that the
// device at (centerHz, sampleRateHz) can see — the in-band CC closest to
// centre when several fall inside the passband (|cc - centre| <=
// sample_rate/2) — or 0 when none does. The web Plots panels rest their
// view on it (#557): a control SDR parked on its CC yields that CC at
// offset ~0, and a voice SDR that still spans the CC rests there too.
//
// Unlike p25ModulationFor there is deliberately no single-system fallback:
// an out-of-band CC would mix beyond Nyquist (the panel clamps it to the
// band edge), resting the view on a frequency the SDR cannot see. Absent
// (0) instead keeps the old centre default for devices that can't see a CC.
// Closest-to-centre (rather than first-match) is used because a wideband
// voice SDR can span several CCs of one system; the nearest one is the
// deterministic, operator-intuitive choice.
func controlChannelFor(systems []trunking.System, centerHz, sampleRateHz uint32) uint32 {
	var best uint32
	var bestDelta int64 = -1
	half := int64(sampleRateHz) / 2
	for i := range systems {
		if systems[i].Protocol != trunking.ProtocolP25 {
			continue
		}
		for _, cc := range systems[i].ControlChannels {
			delta := int64(cc) - int64(centerHz)
			if delta < 0 {
				delta = -delta
			}
			if delta <= half && (bestDelta < 0 || delta < bestDelta) {
				best = cc
				bestDelta = delta
			}
		}
	}
	return best
}

// demodModeName canonicalises a configured demod-mode string to the
// receiver's lowercase name ("c4fm"/"cqpsk"), reusing the same parser
// the decode pipeline applies so the panel sees exactly what the
// receiver runs. Unknown values fall back to C4FM (ParseDemodMode's
// default), matching the receiver's behaviour.
func demodModeName(cfg string) string {
	mode, _ := p25phase1rx.ParseDemodMode(cfg)
	return mode.String()
}

// symbolProtoFor reports which symbol-scope receiver the device at
// (centerHz, sampleRateHz) should be demodulated with — one of the
// selectors WS /api/v1/diag/symbols accepts ("p25-c4fm", "p25-cqpsk",
// "tetra", "dmr") — or "" when no configured system matches, in which
// case the panels keep their existing default.
//
// This exists because p25ModulationFor only inspects P25 Phase 1
// systems, so a TETRA-, DMO- or DMR-only rig reported "" and the web
// panels' "Auto" mode fell back to p25-c4fm. That opened a full P25
// C4FM receiver on a π/4-DQPSK carrier: the 4-level soft track it
// produces there is meaningless, but it is non-empty, so the symbol
// panels computed an MER-like SNR from it (~9 dB on a healthy TETRA
// control channel) and latched "symbol: poor" permanently while the
// decode-health chip correctly read "decode: clean". It also burned a
// whole extra receiver per open panel.
//
// Matching follows p25ModulationFor: a system matches when one of its
// control channels falls inside the device passband, with a fallback to
// the sole configured system when exactly one exists.
func symbolProtoFor(systems []trunking.System, centerHz, sampleRateHz uint32) string {
	proto := func(s trunking.System) string {
		switch s.Protocol {
		case trunking.ProtocolP25:
			// P25 Phase 1 voice shares the CC's modulation, so one value
			// covers both the control channel and any followed call.
			if demodModeName(s.P25Phase1DemodMode) == "cqpsk" {
				return "p25-cqpsk"
			}
			return "p25-c4fm"
		case trunking.ProtocolTETRA, trunking.ProtocolTETRADMO:
			// DMO reuses the TMO physical layer (π/4-DQPSK at 18 ksym/s),
			// so the same receiver applies.
			return "tetra"
		case trunking.ProtocolDMR, trunking.ProtocolDMRTier2, trunking.ProtocolDMRTier1:
			return "dmr"
		default:
			// NXDN, MPT1327, EDACS, … have no symbol-scope receiver yet;
			// "" leaves the panel on its default rather than asserting a
			// wrong one.
			return ""
		}
	}

	var only *trunking.System
	count := 0
	for i := range systems {
		if proto(systems[i]) == "" {
			continue
		}
		count++
		only = &systems[i]
		half := int64(sampleRateHz) / 2
		for _, cc := range systems[i].ControlChannels {
			delta := int64(cc) - int64(centerHz)
			if delta < 0 {
				delta = -delta
			}
			if delta <= half {
				return proto(systems[i])
			}
		}
	}
	if count == 1 {
		return proto(*only)
	}
	return ""
}

// Tune programs the named SDR's centre frequency. Routes through
// the iqtap broker so the broker's CenterHz cache stays current
// (the spectrum + constellation frame stamps follow), the change
// survives pool.Reacquire, and external rigctld clients see the
// same view.
func (p *spectrumProvider) Tune(serial string, centerHz uint32) error {
	if p == nil {
		return errors.New("spectrum: provider not wired")
	}
	br, ok := p.brokers[serial]
	if !ok {
		return fmt.Errorf("spectrum: serial %q is not a known SDR: %w", serial, api.ErrUnknownDevice)
	}
	if err := br.SetCenterFreq(centerHz); err != nil {
		return fmt.Errorf("spectrum: tune %s to %d Hz: %w", serial, centerHz, err)
	}
	return nil
}

// OpenStream subscribes to the named device's broker and starts a
// producer goroutine that converts IQ to spectrum frames at the
// negotiated rate. The returned cleanup func MUST be called by the
// HTTP handler on disconnect — it closes the iqtap subscription and
// cancels the producer's context so its goroutine exits.
func (p *spectrumProvider) OpenStream(ctx context.Context, serial string, fftSize int, fps float64) (<-chan api.SpectrumFrame, func(), error) {
	if p == nil {
		return nil, nil, errors.New("spectrum: provider not wired")
	}
	br, ok := p.brokers[serial]
	if !ok {
		return nil, nil, fmt.Errorf("spectrum: serial %q is not a known SDR: %w", serial, api.ErrUnknownDevice)
	}
	prod, err := spectrum.New(spectrum.Options{
		FFTSize:      fftSize,
		FrameRate:    fps,
		CenterFreqHz: br.CenterHz(),
		SampleRateHz: br.SampleRateHz(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("spectrum: %w", err)
	}

	sub := br.Subscribe()
	internalOut := make(chan spectrum.Frame, 8)
	wireOut := make(chan api.SpectrumFrame, 8)

	streamCtx, cancel := context.WithCancel(ctx)

	go func() {
		defer close(internalOut)
		_ = prod.Run(streamCtx, sub.C, internalOut)
	}()

	go func() {
		defer close(wireOut)
		for {
			select {
			case <-streamCtx.Done():
				return
			case f, ok := <-internalOut:
				if !ok {
					return
				}
				// Refresh stamp in case the device retuned mid-stream.
				prod.SetCenter(br.CenterHz())
				prod.SetSampleRate(br.SampleRateHz())
				select {
				case wireOut <- api.SpectrumFrame{
					TimestampNs:  f.Timestamp.UnixNano(),
					CenterHz:     f.CenterHz,
					SampleRateHz: f.SampleRate,
					Bins:         f.Bins,
				}:
				case <-streamCtx.Done():
					return
				}
			}
		}
	}()

	cleanup := func() {
		cancel()
		sub.Close()
	}
	return wireOut, cleanup, nil
}
