package receiver

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
)

func TestReceiverConstructsAndProcessesSilence(t *testing.T) {
	r := New(Options{
		SampleRateHz: 18_000,
		BitSink:      func(bits []byte, baseIdx int) {},
	})
	silence := make([]complex64, 1800)
	for range 4 {
		r.Process(silence)
	}
}

func TestReceiverConstructorPanicsOnBadParams(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{"missing sample rate", Options{BitSink: func([]byte, int) {}}},
		{"missing sink", Options{SampleRateHz: 18_000}},
		{"sample rate below 2x symbol rate", Options{SampleRateHz: 6000, BitSink: func([]byte, int) {}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic, got nil")
				}
			}()
			_ = New(tc.opts)
		})
	}
}

// controlStream builds a real-air-format SmartNet bit stream: warmup
// + repeated [system-ID pair + idle] frames + trailing sync.
func controlStream(repeats int) []byte {
	seq := []motorola.OSW{
		{Address: 0x4567, Command: motorola.CmdFirstNormal},
		{Address: 0x1F00, Command: 0x8E},
		{Address: 0x02F8, Command: motorola.CmdIdle},
	}
	var out []byte
	for i := 0; i < 200; i++ {
		out = append(out, byte(i&1))
	}
	for r := 0; r < repeats; r++ {
		for _, o := range seq {
			out = append(out, motorola.EncodeOSWFrame(o)...)
		}
	}
	return append(out, motorola.OutboundSyncBits()...)
}

// TestReceiverDecodesModulatedControlChannel is the IQ-level chain
// test at the production channel rate (18 kHz, 5 samples/symbol)
// with the real ±1.2 kHz deviation.
func TestReceiverDecodesModulatedControlChannel(t *testing.T) {
	bits := controlStream(10)
	iq := demod.ModulateGFSK(bits, 5, 4, 0.5, 18_000, DeviationHz)

	got := runChainCountOSWs(t, iq, 18_000, 0)
	if got < 15 {
		t.Fatalf("decoded %d OSWs from a clean modulated stream, want >= 15", got)
	}
}

// TestReceiverToleratesCarrierOffset pins the DC tracker: at
// ±1.2 kHz deviation a 250 Hz residual carrier offset is a ~21%
// slicer bias, which the tracker must remove. (The pre-#1143 chain
// had no offset handling at all.)
func TestReceiverToleratesCarrierOffset(t *testing.T) {
	bits := controlStream(10)
	iq := demod.ModulateGFSK(bits, 5, 4, 0.5, 18_000, DeviationHz)

	got := runChainCountOSWs(t, iq, 18_000, 250)
	if got < 15 {
		t.Fatalf("decoded %d OSWs with a 250 Hz carrier offset, want >= 15", got)
	}
}

// runChainCountOSWs shifts iq by offsetHz, runs the receiver, frames
// the emitted bits and counts CRC-clean OSWs.
func runChainCountOSWs(t *testing.T, iq []complex64, rate, offsetHz float64) int {
	t.Helper()
	if offsetHz != 0 {
		shifted := make([]complex64, len(iq))
		w := 2 * math.Pi * offsetHz / rate
		for i, s := range iq {
			c := complex64(complex(math.Cos(w*float64(i)), math.Sin(w*float64(i))))
			shifted[i] = s * c
		}
		iq = shifted
	}
	count := 0
	framer := motorola.New(motorola.Options{FrequencyHz: 854_562_500, OnOSW: func(motorola.OSW) { count++ }})
	r := New(Options{
		SampleRateHz: rate,
		BitSink:      func(bits []byte, baseIdx int) { framer.Process(bits, baseIdx) },
	})
	r.Process(iq)
	return count
}
