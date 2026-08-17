package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/diag"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// diagProvider implements api.DiagProvider on top of the daemon's
// iqtap broker map. Each OpenIQStream call attaches a fresh
// subscriber to the requested device's broker, runs a per-request
// diag.Decimator at the negotiated target rate, and pipes the
// wire frames out for the API's WS handler to drain. Cleanup
// tears the subscription and decimator goroutine down.
//
// One decimator per WS subscriber — same trade-off the spectrum
// provider makes. CPU scales linearly with #clients; for the
// single-operator GopherTrunk audience this is fine. A future
// shared-decimator optimization would be mechanical (the
// spectrum_provider.go pattern). Defer until profiling shows the
// wins.
type diagProvider struct {
	pool       *sdr.Pool
	brokers    map[string]*iqtap.Broker
	sampleRate uint32
	log        *slog.Logger
}

func newDiagProvider(pool *sdr.Pool, brokers map[string]*iqtap.Broker, sampleRate uint32, log *slog.Logger) *diagProvider {
	if log == nil {
		log = slog.Default()
	}
	return &diagProvider{pool: pool, brokers: brokers, sampleRate: sampleRate, log: log}
}

// OpenIQStream is api.DiagProvider's only method. See its docstring
// for the wire-side contract; this implementation builds a decimator
// over the broker subscription and runs it on a goroutine.
func (p *diagProvider) OpenIQStream(ctx context.Context, serial string, targetRateSPS uint32, offsetHz int32) (<-chan api.IQFrame, func(), error) {
	if p == nil {
		return nil, nil, errors.New("diag: provider not wired")
	}
	br, ok := p.brokers[serial]
	if !ok {
		return nil, nil, fmt.Errorf("diag: serial %q is not a known SDR: %w", serial, api.ErrUnknownDevice)
	}
	inputRate := br.SampleRateHz()
	if inputRate == 0 {
		inputRate = p.sampleRate
	}
	if inputRate == 0 {
		return nil, nil, errors.New("diag: cannot determine input sample rate")
	}
	if targetRateSPS == 0 {
		targetRateSPS = diag.DefaultDecimatedRateSPS
	}
	if targetRateSPS > inputRate {
		targetRateSPS = inputRate
	}
	// Clamp the requested view offset to the device's Nyquist; a mix
	// beyond +/- Fs/2 just aliases back into the band.
	half := int32(inputRate / 2)
	if offsetHz > half {
		offsetHz = half
	} else if offsetHz < -half {
		offsetHz = -half
	}

	dec, err := diag.New(diag.Options{
		InputRateSPS:   inputRate,
		TargetRateSPS:  targetRateSPS,
		OffsetHz:       offsetHz,
		ChunksPerFrame: 4,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("diag: %w", err)
	}

	sub := br.Subscribe()
	internalOut := make(chan diag.IQFrame, 8)
	wireOut := make(chan api.IQFrame, 8)

	streamCtx, cancel := context.WithCancel(ctx)

	centerHz := func() uint32 { return br.CenterHz() }
	tsNs := func() int64 { return time.Now().UnixNano() }

	go func() {
		defer close(internalOut)
		_ = dec.Run(streamCtx, sub.C, internalOut, centerHz, tsNs)
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
				points := make([]api.IQPoint, len(f.Points))
				for i, pt := range f.Points {
					points[i] = api.IQPoint{I: pt.I, Q: pt.Q}
				}
				wire := api.IQFrame{
					TimestampNs:  f.TimestampNs,
					SampleRateHz: f.SampleRate,
					CenterHz:     f.CenterHz,
					Points:       points,
					EnergyDBFS:   f.EnergyDBFS,
				}
				select {
				case wireOut <- wire:
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
