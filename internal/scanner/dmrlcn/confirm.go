package dmrlcn

import (
	"context"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/tuner"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// Confirmer decides whether the carrier at freqHz is a live DMR burst —
// the decode-confirmation step that turns a temporal grant/onset
// coincidence into a trusted (LCN, frequency) pair. confirmed reports
// whether DMR was decoded; weight expresses the confidence (a voice
// burst weighs more than a data burst).
type Confirmer interface {
	Confirm(ctx context.Context, freqHz uint32) (confirmed bool, weight float64)
}

// confirm.go's DMR receiver constants mirror the wideband Tier III path
// (internal/scanner/widebandt2) so a probe tap decodes identically to the
// dongle's primary control-channel tap.
const (
	confirmNarrowbandHz = 48_000.0
	confirmDeviationHz  = 1944.0
	confirmClockGain    = 0.025
	confirmSyncTol      = 2 // dibit errors tolerated in a 24-dibit sync match
)

// ddcConfirmer confirms a candidate carrier by briefly DDC-tapping the
// shared wideband IQ stream at the carrier's offset and running the DMR
// receiver until it sees a sync word (or the probe times out). It holds
// no long-lived state; each Confirm call subscribes, taps, and tears
// down.
type ddcConfirmer struct {
	broker       *iqtap.Broker
	centerHz     uint32
	sampleRateHz uint32
	guardFrac    float64
	timeout      time.Duration
	minSyncs     int
}

func newDDCConfirmer(broker *iqtap.Broker, centerHz, sampleRateHz uint32, guardFrac float64) *ddcConfirmer {
	if guardFrac <= 0 {
		guardFrac = 0.05
	}
	return &ddcConfirmer{
		broker:       broker,
		centerHz:     centerHz,
		sampleRateHz: sampleRateHz,
		guardFrac:    guardFrac,
		timeout:      700 * time.Millisecond,
		minSyncs:     1,
	}
}

// Confirm taps the carrier and returns true once it decodes at least
// minSyncs DMR sync words within the probe timeout. weight is 1.0 when a
// voice sync is seen, 0.8 for a data-only burst, 0.7 otherwise.
func (c *ddcConfirmer) Confirm(ctx context.Context, freqHz uint32) (bool, float64) {
	offset := float64(freqHz) - float64(c.centerHz)
	bank := tuner.NewDDCBank(float64(c.sampleRateHz), confirmNarrowbandHz, c.guardFrac)

	var (
		matches           []dmr.Match
		sawVoice, sawData bool
		syncCount         int
		detector          = dmr.NewSyncDetector(nil, confirmSyncTol)
	)
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: confirmNarrowbandHz,
		DeviationHz:  confirmDeviationHz,
		ClockGain:    confirmClockGain,
		DibitSink: dmr.DibitSink(func(d []uint8, base int) {
			matches, _ = detector.Process(matches[:0], d, base)
			for _, m := range matches {
				syncCount++
				name := m.Pattern.Name
				switch {
				case strings.Contains(name, "Voice"):
					sawVoice = true
				case strings.Contains(name, "Data"):
					sawData = true
				}
			}
		}),
	})
	if err := bank.AddTap(offset, func(out []complex64) {
		if len(out) > 0 {
			rx.Process(out)
		}
	}); err != nil {
		// Carrier is outside the dongle's usable IQ window — can't probe it.
		return false, 0
	}

	sub := c.broker.Subscribe()
	defer sub.Close()

	deadline := time.NewTimer(c.timeout)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return false, 0
		case <-deadline.C:
			return false, 0
		case chunk, ok := <-sub.C:
			if !ok {
				return false, 0
			}
			bank.Process(chunk)
			if syncCount >= c.minSyncs {
				return true, syncWeight(sawVoice, sawData)
			}
		}
	}
}

func syncWeight(voice, data bool) float64 {
	switch {
	case voice:
		return 1.0
	case data:
		return 0.8
	default:
		return 0.7
	}
}
