package conventional

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// A P25 conventional channel (issue #1239) opens on in-channel power like an
// FM channel, grants as protocol "p25" so the composer runs its P25 Phase 1
// voice chain, marks the grant Conventional so that chain does not gate on
// the synthetic GroupID, and ends on hangtime when the carrier drops.
func TestP25ChannelGrantsP25AndEndsOnCarrierDrop(t *testing.T) {
	const freq = 155_752_500 // VCALL10, the VHF P25 calling channel
	r := rand.New(rand.NewSource(7))
	dibits := make([]uint8, 720) // 0.15 s of 4800-baud C4FM
	for i := range dibits {
		dibits[i] = uint8(r.Intn(4))
	}
	sig := demod.ModulateP25C4FM(dibits, amTestRate, 1800)
	for i := range sig {
		sig[i] *= 0.1 // ≈ -20 dBFS, well over the -50 squelch
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
		DeviceSerial: "CONV-P25",
		Channels: []Channel{{
			Label: "VCALL10", FrequencyHz: freq, Mode: ModeP25,
			SquelchDbFS: -50, Hangtime: 200 * time.Millisecond,
		}},
		SampleRateHz: amTestRate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.powerMeterFor(0) == nil {
		t.Fatal("a P25 channel must squelch on in-channel power, like FM")
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
		t.Fatal("no call started on a P25 carrier")
	}
	g := eng.starts[0]
	if g.Protocol != "p25" {
		t.Errorf("grant protocol %q, want p25", g.Protocol)
	}
	if !g.Conventional {
		t.Error("grant not marked Conventional: the P25 voice chain would gate the call on its synthetic GroupID")
	}
	if g.GroupLabel != "VCALL10" {
		t.Errorf("grant label %q, want the channel label", g.GroupLabel)
	}
	normal := 0
	for _, reason := range eng.endReasons {
		if reason == trunking.EndReasonNormal {
			normal++
		}
	}
	if normal == 0 {
		t.Errorf("call did not end on hangtime after the carrier dropped (end reasons %v)", eng.endReasons)
	}
}

// Analog-only features have nothing to work on in a digital channel, so the
// scanner refuses them rather than silently never opening.
func TestP25ChannelRejectsAnalogOnlyOptions(t *testing.T) {
	for name, ch := range map[string]Channel{
		"ctcss":   {FrequencyHz: 1, Mode: ModeP25, Tone: ToneConfig{Mode: "ctcss", CTCSSHz: 100}},
		"dcs":     {FrequencyHz: 1, Mode: ModeP25, Tone: ToneConfig{Mode: "dcs", DCSCode: "023"}},
		"decoder": {FrequencyHz: 1, Mode: ModeP25, Decoders: []string{DecoderMDC1200}},
	} {
		if _, err := New(Options{
			Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
			DeviceSerial: "X", Channels: []Channel{ch}, SampleRateHz: amTestRate,
		}); err == nil {
			t.Errorf("%s on a p25 channel accepted", name)
		}
	}
	if _, err := New(Options{
		Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "X", Channels: []Channel{{FrequencyHz: 1, Mode: ModeP25, Tone: ToneConfig{Mode: "none"}}},
	}); err != nil {
		t.Errorf("plain p25 channel rejected: %v", err)
	}
}
