package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/symbolscope"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// mixerProvider implements api.MixerProvider on top of the daemon's iqtap
// broker map. Each OpenMixerStream call attaches a fresh subscriber to
// the requested device's broker and runs a per-request symbolscope.Engine
// with the Mixer FFT taps enabled (DDC → receiver → raw/tuned spectra),
// piping the spectrum frames out for the API's WS handler to drain.
//
// One engine per WS subscriber — the same trade-off symbolProvider makes.
// The Mixer FFTs add a bounded, rate-limited cost on top of the receiver
// the engine already runs; for the single-operator GopherTrunk audience
// this is acceptable, and the broker fan-out never back-pressures the
// production control-channel decode.
type mixerProvider struct {
	pool       *sdr.Pool
	brokers    map[string]*iqtap.Broker
	sampleRate uint32
	log        *slog.Logger
}

func newMixerProvider(pool *sdr.Pool, brokers map[string]*iqtap.Broker, sampleRate uint32, log *slog.Logger) *mixerProvider {
	if log == nil {
		log = slog.Default()
	}
	return &mixerProvider{pool: pool, brokers: brokers, sampleRate: sampleRate, log: log}
}

// OpenMixerStream is api.MixerProvider's only method. It builds a
// symbolscope.Engine with MixerEmit set over the broker subscription and
// runs it on a goroutine.
func (p *mixerProvider) OpenMixerStream(ctx context.Context, serial, proto string, offsetHz int32) (<-chan api.MixerFrame, func(), error) {
	if p == nil {
		return nil, nil, errors.New("mixer: provider not wired")
	}
	br, ok := p.brokers[serial]
	if !ok {
		return nil, nil, fmt.Errorf("mixer: serial %q is not a known SDR", serial)
	}
	protocol, demod, err := symbolProtoToReceiver(proto)
	if err != nil {
		return nil, nil, err
	}
	inputRate := br.SampleRateHz()
	if inputRate == 0 {
		inputRate = p.sampleRate
	}
	if inputRate == 0 {
		return nil, nil, errors.New("mixer: cannot determine input sample rate")
	}
	// Clamp the requested view offset to the device's Nyquist; a mix
	// beyond +/- Fs/2 just aliases back into the band.
	half := int32(inputRate / 2)
	if offsetHz > half {
		offsetHz = half
	} else if offsetHz < -half {
		offsetHz = -half
	}

	wireOut := make(chan api.MixerFrame, 8)
	streamCtx, cancel := context.WithCancel(ctx)

	eng, err := symbolscope.New(symbolscope.Options{
		Protocol:  protocol,
		DemodMode: demod,
		InRateHz:  float64(inputRate),
		OffsetHz:  offsetHz,
		CenterHz:  br.CenterHz(),
		// Symbol Emit is required by the engine; it batches dibits but we
		// don't forward them on this stream — the Mixer plot only needs the
		// FFT taps below.
		Emit: func(symbolscope.Frame) {},
		MixerEmit: func(f symbolscope.MixerFrame) {
			wire := api.MixerFrame{
				TimestampNs:     f.TimestampNs,
				CenterHz:        f.CenterHz,
				SampleRateHz:    f.SampleRateHz,
				OffsetHz:        f.OffsetHz,
				CarrierOffsetHz: f.CarrierOffsetHz,
				RawBins:         f.RawBins,
				TunedBins:       f.TunedBins,
			}
			// Non-blocking: drop on a wedged WS client rather than stalling
			// the broker drain goroutine.
			select {
			case wireOut <- wire:
			case <-streamCtx.Done():
			default:
			}
		},
	})
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("mixer: %w", err)
	}

	sub := br.Subscribe()

	go func() {
		defer close(wireOut)
		defer eng.Close()
		for {
			select {
			case <-streamCtx.Done():
				return
			case chunk, ok := <-sub.C:
				if !ok {
					return
				}
				eng.Process(chunk)
			}
		}
	}()

	cleanup := func() {
		cancel()
		sub.Close()
	}
	return wireOut, cleanup, nil
}
