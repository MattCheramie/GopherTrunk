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

// CaptureStream records seconds worth of raw IQ from the named device's broker,
// handing each chunk to sink as it arrives instead of buffering the whole grab.
// The api handler's sink encodes each chunk straight to disk, so a long or
// high-rate capture no longer holds the whole stream (and a second encoded copy)
// in RAM.
func (p *captureProvider) CaptureStream(ctx context.Context, serial string, seconds int, sink func([]complex64) error) (uint32, uint32, error) {
	if p == nil {
		return 0, 0, errors.New("capture: provider not wired")
	}
	br, ok := p.brokers[serial]
	if !ok {
		return 0, 0, fmt.Errorf("capture: serial %q is not a known SDR: %w", serial, api.ErrUnknownDevice)
	}
	rate := br.SampleRateHz()
	if rate == 0 {
		return 0, 0, errors.New("capture: device has no sample rate yet")
	}
	center := br.CenterHz()
	target := int64(seconds) * int64(rate)

	sub := br.Subscribe()
	defer sub.Close()

	// Safety deadline so an under-delivering device can't hang the request
	// waiting to reach the sample target. The timer is an explicit select arm
	// (not a post-receive check) because the broker pauses fan-out whenever no
	// primary StreamIQ session is running — during a control-channel hunt
	// backoff the device delivers nothing, so a receive-then-check deadline
	// would block forever on sub.C and wedge the HTTP request ("Capturing…"
	// never ends). See internal/sdr/iqtap/broker.go.
	timer := time.NewTimer(time.Duration(seconds)*time.Second + 5*time.Second)
	defer timer.Stop()
	var got int64
loop:
	for got < target {
		select {
		case <-ctx.Done():
			return rate, center, ctx.Err()
		case <-timer.C:
			break loop
		case chunk, ok := <-sub.C:
			if !ok {
				return rate, center, errors.New("capture: IQ stream ended before capture finished")
			}
			if err := sink(chunk); err != nil {
				return rate, center, err
			}
			got += int64(len(chunk))
		}
	}
	if got == 0 {
		return rate, center, fmt.Errorf(
			"capture: device %q delivered no IQ within %ds — the tuner is not currently "+
				"streaming (a scan may be mid-hunt / in control-channel backoff); start or "+
				"resume a scan on this SDR, or retry", serial, seconds)
	}
	return rate, center, nil
}
