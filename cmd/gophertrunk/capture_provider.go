package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// captureProvider implements api.CaptureProvider over the daemon's iqtap
// broker map. A capture subscribes a fresh observer to the requested device's
// broker, collects the requested number of seconds of complex64 IQ, and
// returns it with the broker's current sample rate + centre frequency.
//
// It taps the same live stream the control decoder runs on (the broker fans
// out to every observer), so a capture never disturbs decoding — a slow drain
// is dropped by the broker, not backpressured onto the primary consumer. The
// Devices picker delegates to the spectrum provider so the capture and
// spectrum device lists stay identical.
type captureProvider struct {
	sp      *spectrumProvider
	brokers map[string]*iqtap.Broker
}

func newCaptureProvider(pool *sdr.Pool, brokers map[string]*iqtap.Broker, log *slog.Logger) *captureProvider {
	// The capture picker doesn't use per-device modulation, so it omits
	// the systems list (P25Modulation stays empty on its device list).
	return &captureProvider{sp: newSpectrumProvider(pool, brokers, nil, log), brokers: brokers}
}

// Devices reuses the spectrum provider's broker walk.
func (p *captureProvider) Devices() []api.SpectrumDevice { return p.sp.Devices() }

// Capture records seconds worth of raw IQ from the named device's broker.
func (p *captureProvider) Capture(ctx context.Context, serial string, seconds int) ([]complex64, uint32, uint32, error) {
	if p == nil {
		return nil, 0, 0, errors.New("capture: provider not wired")
	}
	br, ok := p.brokers[serial]
	if !ok {
		return nil, 0, 0, fmt.Errorf("capture: serial %q is not a known SDR", serial)
	}
	rate := br.SampleRateHz()
	if rate == 0 {
		return nil, 0, 0, errors.New("capture: device has no sample rate yet")
	}
	target := int64(seconds) * int64(rate)

	sub := br.Subscribe()
	defer sub.Close()

	// Safety deadline so an under-delivering device can't hang the request
	// waiting to reach the sample target.
	deadline := time.Now().Add(time.Duration(seconds)*time.Second + 5*time.Second)
	out := make([]complex64, 0, target)
	for int64(len(out)) < target {
		select {
		case <-ctx.Done():
			return out, rate, br.CenterHz(), ctx.Err()
		case chunk, ok := <-sub.C:
			if !ok {
				return out, rate, br.CenterHz(), errors.New("capture: IQ stream ended before capture finished")
			}
			out = append(out, chunk...)
		}
		if time.Now().After(deadline) {
			break
		}
	}
	return out, rate, br.CenterHz(), nil
}
