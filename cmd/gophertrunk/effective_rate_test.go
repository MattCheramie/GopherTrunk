package main

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// rateOnlyDevice is a minimal sdr.Device. quantizeTo, when non-zero, makes it
// advertise the optional ActualSampleRate() extension and report that exact
// rate (modelling the RTL2832U resampler quantization). When zero, the device
// does NOT implement ActualSampleRate at all — modelling file replay / rtl_tcp
// backends that deliver exactly the requested rate.
type rateOnlyDevice struct{}

func (rateOnlyDevice) Info() sdr.Info             { return sdr.Info{} }
func (rateOnlyDevice) SetCenterFreq(uint32) error { return nil }
func (rateOnlyDevice) SetSampleRate(uint32) error { return nil }
func (rateOnlyDevice) SetGain(int) error          { return nil }
func (rateOnlyDevice) SetPPM(int) error           { return nil }
func (rateOnlyDevice) SetBiasTee(bool) error      { return nil }
func (rateOnlyDevice) Close() error               { return nil }
func (rateOnlyDevice) StreamIQ(context.Context) (<-chan []complex64, error) {
	return nil, nil
}

// quantizingDevice embeds rateOnlyDevice and additionally implements the
// optional ActualSampleRate() extension.
type quantizingDevice struct {
	rateOnlyDevice
	actual uint32
	err    error
}

func (q quantizingDevice) ActualSampleRate() (uint32, error) { return q.actual, q.err }

func TestEffectiveControlRate(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	const requested = 2_048_000

	cases := []struct {
		name string
		dev  sdr.Device
		want uint32
	}{
		{
			// File replay / rtl_tcp: no ActualSampleRate extension, so the
			// configured rate is used verbatim.
			name: "no extension -> requested",
			dev:  rateOnlyDevice{},
			want: requested,
		},
		{
			// RTL2832U quantizes 2.048 MS/s to a nearby divisor: the decoder
			// must be built from the delivered rate, not the requested one.
			name: "quantized -> actual",
			dev:  quantizingDevice{actual: 2_048_001},
			want: 2_048_001,
		},
		{
			// Exact-divisor rate (2.4 / 0.96 MS/s land here): actual ==
			// requested, no divergence, returned unchanged (and no warn).
			name: "exact -> requested",
			dev:  quantizingDevice{actual: requested},
			want: requested,
		},
		{
			// A read-back error or zero must fall back to the configured
			// rate rather than zeroing the down-converter input rate.
			name: "error -> requested",
			dev:  quantizingDevice{actual: 0, err: io.EOF},
			want: requested,
		},
		{
			name: "zero -> requested",
			dev:  quantizingDevice{actual: 0},
			want: requested,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := effectiveControlRate(log, c.dev, "TESTSERIAL", requested); got != c.want {
				t.Errorf("effectiveControlRate = %d, want %d", got, c.want)
			}
		})
	}
}
