package receiver

import (
	"io"
	"log/slog"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// TestReceiverAFCLocksWithLargeCarrierOffset drives the production 144 kHz /
// Gardner / AFC path with a synthesized TETRA control channel shifted by a
// carrier-frequency offset well beyond the differential AFC's ±f_sym/8 =
// ±2250 Hz acquisition window, and asserts a real lock at each offset.
//
// This is the regression guard for issue #940. The original AFC estimated the
// offset from the 4×Δφ differential mean, which wraps into (−π/4, π/4] rad/
// symbol, so any carrier more than 2250 Hz off-centre aliased into that window
// and left a multi-kHz constellation spin that broke Gardner timing — the
// receiver never locked even on a clean, strong signal. Real SDR front-ends
// routinely sit several kHz off-frequency, so this was a real acquisition gap.
// The two-stage AFC (coarse mean-frequency on the pre-matched-filter block +
// the existing fine estimator) extends acquisition to roughly ±6 kHz.
//
// Before the fix this fails at every offset here; after it, all lock.
func TestReceiverAFCLocksWithLargeCarrierOffset(t *testing.T) {
	const (
		controlFreqHz        = 412_062_500
		colourCode           = uint32(0x12345)
		locationArea  uint16 = 0x42
		sps                  = 8
		span                 = 8
		alpha                = 0.35
		gardnerGain          = 0.005
		sampleRateHz         = 144_000.0
	)

	dibits := buildTETRASCHHDDibits(120, colourCode, locationArea)
	baseIQ := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)

	// Offsets chosen to straddle the old ±2250 Hz cliff, both signs, out to the
	// ~±6 kHz range the two-stage AFC targets.
	for _, cfoHz := range []float64{-5500, -4000, 3000, 4000, 5500} {
		t.Run(offsetName(cfoHz), func(t *testing.T) {
			iq := make([]complex64, len(baseIQ))
			for i, s := range baseIQ {
				ph := 2 * math.Pi * cfoHz * float64(i) / sampleRateHz
				iq[i] = s * complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
			}

			bus := events.NewBus(64)
			defer bus.Close()
			sub := bus.Subscribe()
			defer sub.Close()

			cc := tetra.New(tetra.Options{
				Bus:         bus,
				Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				SystemName:  "TETRASite",
				FrequencyHz: controlFreqHz,
			})
			cc.SetChannelCoding(tetra.ChannelCodingOn)
			cc.SetExpectedChannel(tetra.ChannelSCHHD)
			cc.SetColourCode(colourCode)

			rx := New(Options{
				SampleRateHz:        sampleRateHz,
				ClockMode:           ClockGardner,
				GardnerGain:         gardnerGain,
				EnableAFC:           true,
				EnableChannelFilter: true,
				DibitSink:           func(d []uint8, b int) { cc.Process(d, b) },
				SoftSink:            func(diffs []complex64, b int) { cc.StashSoft(diffs, b) },
			})

			const chunk = 4096
			for i := 0; i < len(iq); i += chunk {
				end := i + chunk
				if end > len(iq) {
					end = len(iq)
				}
				rx.Process(iq[i:end])
			}

			locked := false
		drain:
			for {
				select {
				case ev := <-sub.C:
					if ev.Kind == events.KindCCLocked {
						locked = true
					}
				default:
					break drain
				}
			}
			if !locked {
				t.Fatalf("no cc.locked at CFO=%.0f Hz (afc reported %.0f Hz): the AFC failed "+
					"to acquire a carrier beyond ±2250 Hz (regression for issue #940)",
					cfoHz, rx.afc.OffsetHz(SymbolRate))
			}
		})
	}
}

func offsetName(hz float64) string {
	sign := "+"
	if hz < 0 {
		sign = "-"
		hz = -hz
	}
	// small hand-formatter to avoid importing fmt/strconv for a label
	whole := int(hz + 0.5)
	digits := []byte{}
	if whole == 0 {
		digits = []byte{'0'}
	}
	for whole > 0 {
		digits = append([]byte{byte('0' + whole%10)}, digits...)
		whole /= 10
	}
	return "CFO" + sign + string(digits) + "Hz"
}
