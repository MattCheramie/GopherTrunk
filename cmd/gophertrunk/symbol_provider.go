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
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// symbolProvider implements api.SymbolProvider on top of the daemon's
// iqtap broker map. Each OpenSymbolStream call attaches a fresh
// subscriber to the requested device's broker and runs a per-request
// symbolscope.Engine (DDC → protocol receiver → soft/dibit taps),
// piping the recovered-symbol frames out for the API's WS handler to
// drain. Cleanup tears the subscription and engine goroutine down.
//
// One engine per WS subscriber — same trade-off diagProvider makes. The
// engine runs a full down-converter + receiver, so it is heavier than
// the diag decimator; for the single-operator GopherTrunk audience this
// is acceptable, and the broker fan-out never back-pressures the
// production control-channel decode.
type symbolProvider struct {
	pool       *sdr.Pool
	brokers    map[string]*iqtap.Broker
	sampleRate uint32
	log        *slog.Logger
}

func newSymbolProvider(pool *sdr.Pool, brokers map[string]*iqtap.Broker, sampleRate uint32, log *slog.Logger) *symbolProvider {
	if log == nil {
		log = slog.Default()
	}
	return &symbolProvider{pool: pool, brokers: brokers, sampleRate: sampleRate, log: log}
}

// symbolProtoToReceiver maps the WS proto selector onto a protocol +
// P25 demod mode. Phase 1 ships P25 Phase 1 only.
func symbolProtoToReceiver(proto string) (trunking.Protocol, string, error) {
	switch proto {
	case "", "p25-c4fm":
		return trunking.ProtocolP25, "c4fm", nil
	case "p25-cqpsk":
		return trunking.ProtocolP25, "cqpsk", nil
	default:
		return trunking.ProtocolUnknown, "", fmt.Errorf("symbol: unsupported proto %q", proto)
	}
}

// OpenSymbolStream is api.SymbolProvider's only method. See its
// docstring for the wire-side contract; this implementation builds a
// symbolscope.Engine over the broker subscription and runs it on a
// goroutine.
func (p *symbolProvider) OpenSymbolStream(ctx context.Context, serial, proto string, offsetHz int32) (<-chan api.SymbolFrame, func(), error) {
	if p == nil {
		return nil, nil, errors.New("symbol: provider not wired")
	}
	br, ok := p.brokers[serial]
	if !ok {
		return nil, nil, fmt.Errorf("symbol: serial %q is not a known SDR", serial)
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
		return nil, nil, errors.New("symbol: cannot determine input sample rate")
	}
	// Clamp the requested view offset to the device's Nyquist; a mix
	// beyond +/- Fs/2 just aliases back into the band.
	half := int32(inputRate / 2)
	if offsetHz > half {
		offsetHz = half
	} else if offsetHz < -half {
		offsetHz = -half
	}

	wireOut := make(chan api.SymbolFrame, 8)
	streamCtx, cancel := context.WithCancel(ctx)

	eng, err := symbolscope.New(symbolscope.Options{
		Protocol:  protocol,
		DemodMode: demod,
		InRateHz:  float64(inputRate),
		OffsetHz:  offsetHz,
		CenterHz:  br.CenterHz(),
		Emit: func(f symbolscope.Frame) {
			wire := api.SymbolFrame{
				TimestampNs:  f.TimestampNs,
				SymbolRateHz: f.SymbolRateHz,
				CenterHz:     f.CenterHz,
				OffsetHz:     f.OffsetHz,
				Soft:         f.Soft,
				SymI:         f.SymI,
				SymQ:         f.SymQ,
				Dibits:       f.Dibits,
				IsBits:       f.IsBits,
				BaseIdx:      f.BaseIdx,
			}
			// Non-blocking: drop on a wedged WS client rather than
			// stalling the broker drain goroutine.
			select {
			case wireOut <- wire:
			case <-streamCtx.Done():
			default:
			}
		},
	})
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("symbol: %w", err)
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
