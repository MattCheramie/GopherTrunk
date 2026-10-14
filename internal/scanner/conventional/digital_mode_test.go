package conventional

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// DMR, NXDN and D-STAR scan-list channels (issue #1261) open on in-channel
// power like p25, grant under the protocol the composer and recorder map to
// that mode's voice chain and vocoder, mark the grant Conventional so the
// chain does not gate on the synthetic GroupID, and ask for the
// cadence-detecting DMR decoder on a DMR channel only.
func TestDigitalModeChannelsGrantTheirProtocol(t *testing.T) {
	for _, tc := range []struct {
		mode, protocol string
		interleaved    bool
	}{
		{ModeDMR, "dmr-tier2", true},
		{ModeNXDN, "nxdn", false},
		{ModeDStar, "dstar", false},
		{ModeYSF, "ysf", false},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			const freq = 442_387_500
			r := rand.New(rand.NewSource(11))
			dibits := make([]uint8, 720)
			for i := range dibits {
				dibits[i] = uint8(r.Intn(4))
			}
			sig := demod.ModulateP25C4FM(dibits, amTestRate, 1800)
			for i := range sig {
				sig[i] *= 0.1
			}
			idle := make([]complex64, amTestRate/2)
			for i := range idle {
				idle[i] = complex(float32(r.NormFloat64()*1e-4), float32(r.NormFloat64()*1e-4))
			}
			tuner := &fakeTuner{}
			eng := &fakeEngine{}
			iq := &fakeIQ{chunks: map[uint32][][]complex64{freq: append(chunked(sig, 16384), chunked(idle, 16384)...)}, tuner: tuner}
			s, err := New(Options{
				Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
				DeviceSerial: "CONV-DIG",
				Channels: []Channel{{
					Label: "Rptr", FrequencyHz: freq, Mode: tc.mode,
					SquelchDbFS: -50, Hangtime: 200 * time.Millisecond,
				}},
				SampleRateHz: amTestRate,
			})
			if err != nil {
				t.Fatal(err)
			}
			if s.powerMeterFor(0) == nil {
				t.Fatal("a digital channel must squelch on in-channel power, like FM")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			go func() { _ = s.Run(ctx) }()
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) && eng.normalEndCount() == 0 {
				time.Sleep(10 * time.Millisecond)
			}
			cancel()
			eng.mu.Lock()
			defer eng.mu.Unlock()
			if len(eng.starts) == 0 {
				t.Fatalf("no call started on a %s carrier", tc.mode)
			}
			g := eng.starts[0]
			if g.Protocol != tc.protocol {
				t.Errorf("grant protocol %q, want %q", g.Protocol, tc.protocol)
			}
			if !g.Conventional {
				t.Error("grant not marked Conventional")
			}
			if g.DMRInterleavedVoice != tc.interleaved {
				t.Errorf("DMRInterleavedVoice = %v, want %v", g.DMRInterleavedVoice, tc.interleaved)
			}
			if len(eng.endReasons) == 0 {
				t.Error("call never ended after the carrier dropped")
			}
		})
	}
}

// The analog-only options are refused on the new digital modes, as on p25.
func TestDigitalModeChannelsRejectAnalogOptions(t *testing.T) {
	for _, mode := range []string{ModeDMR, ModeNXDN, ModeDStar, ModeYSF} {
		for _, ch := range []Channel{
			{Label: "t", FrequencyHz: 1, Mode: mode, Tone: ToneConfig{Mode: "ctcss", CTCSSHz: 100}},
			{Label: "d", FrequencyHz: 1, Mode: mode, Decoders: []string{DecoderMDC1200}},
		} {
			_, err := New(Options{
				Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
				DeviceSerial: "X", Channels: []Channel{ch}, SampleRateHz: amTestRate,
			})
			if err == nil {
				t.Errorf("mode %s accepted %+v", mode, ch)
			}
		}
	}
}
