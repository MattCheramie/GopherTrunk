package ccdecoder

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// dpmrBeaconIQ is a Mode 3 beacon in the air format of
// internal/radio/dpmr (clause 4.2.3.7), modulated as the standard
// specifies (4FSK, 2400 symbols a second, +-1050 Hz outer deviation, root
// raised cosine 0.2) at 48 kHz, offHz off frequency, with white noise
// snrDB below it. Every fourth frameset is a Goto Channel sending
// talkgroup 012345* to 446.00625 MHz, with the calling party 1234567 in
// its SYScasts; the rest are Alohas.
func dpmrBeaconIQ(seed int64, framesets int, offHz, snrDB float64) []complex64 {
	r := rand.New(rand.NewSource(seed))
	var d []uint8
	for i := 0; i < 600; i++ {
		d = append(d, uint8(r.Intn(4)))
	}
	d = append(d, dpmr.AirFS1Dibits()...)
	const sys = 0x0C55
	sy1 := dpmr.EncodeSYScast1(dpmr.SYScast1{SystemCode: sys})
	band, sep, field, _ := dpmr.FrequencyFields(446_006_250)
	for i := 0; i < framesets; i++ {
		if i%4 == 3 {
			g := dpmr.EncodeGotoChannel(dpmr.GotoChannel{Called: 0x02C00A, RxField: field, TxField: field,
				Band: band, Sep5k: sep, Mode: dpmr.AirModeVoice})
			a, b := dpmr.EncodeCallingParty(0x1B91FD)
			d = append(d, dpmr.EncodeBeaconFrameset(g, dpmr.BeaconChannelCode0, [3]uint32{sy1, a, b})...)
			continue
		}
		cfc := dpmr.EncodeSYScast2(dpmr.SYTypeFrameCounter, uint16(i))
		d = append(d, dpmr.EncodeBeaconFrameset(dpmr.EncodeAloha(dpmr.Aloha{MSID: 0xFFFFFF, SystemCode: sys}),
			dpmr.BeaconChannelCode0, [3]uint32{sy1, cfc, cfc})...)
	}
	const fs = 48_000.0
	iq := demod.ModulateC4FM(d, 20, 8, 0.20, fs, 1050)
	sigma := math.Pow(10, -snrDB/20) / math.Sqrt2
	out := make([]complex64, len(iq))
	for i := range iq {
		p := 2 * math.Pi * offHz * float64(i) / fs
		v := complex128(iq[i]) * complex(math.Cos(p), math.Sin(p))
		v += complex(r.NormFloat64()*sigma, r.NormFloat64()*sigma)
		out[i] = complex64(v)
	}
	return out
}

// runDPMRPipeline feeds iq through the production pipeline and returns the
// lock and grant events.
func runDPMRPipeline(t *testing.T, iq []complex64) (locks []dpmr.LockState, grants []trunking.Grant) {
	t.Helper()
	bus := events.NewBus(4096)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	p, err := newDPMRPipeline(PipelineOptions{Bus: bus, SystemName: "Beacon", FrequencyHz: 446_100_000, SampleRateHz: 48_000})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(iq); i += 4096 {
		p.Process(iq[i:min(i+4096, len(iq))])
	}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				locks = append(locks, ev.Payload.(dpmr.LockState))
			case events.KindGrant:
				grants = append(grants, ev.Payload.(trunking.Grant))
			}
		case <-deadline:
			return locks, grants
		default:
			if len(sub.C) == 0 {
				return locks, grants
			}
		}
	}
}

// A modulated beacon locks the production pipeline with its System Identity
// Code and turns its Goto Channels into grants for the right call, on
// frequency and a dongle's typical error off it.
func TestDPMRPipelineLocksAndGrantsOnABeacon(t *testing.T) {
	for _, offHz := range []float64{0, 450, -900} {
		locks, grants := runDPMRPipeline(t, dpmrBeaconIQ(1, 40, offHz, 25))
		if len(locks) != 1 || locks[0].SystemID != 0x0C55 {
			t.Errorf("%+.0f Hz: locks %+v", offHz, locks)
		}
		// Ten Goto Channels, 440 ms apart: the same call, so the first is
		// the grant and the rest repeat it within the 2 s window... except
		// that the window restarts on every repeat, so exactly one.
		if len(grants) != 1 {
			t.Errorf("%+.0f Hz: %d grants, want 1", offHz, len(grants))
			continue
		}
		g := grants[0]
		if g.Protocol != "dpmr" || g.GroupID != 0x02C00A || g.SourceID != 0x1B91FD || g.FrequencyHz != 446_006_250 {
			t.Errorf("%+.0f Hz: grant %+v", offHz, g)
		}
	}
}

// From every sub-symbol start phase the pipeline locks within 400 ms of the
// beacon starting. Measured: 120 ms at all twenty phases of one symbol with
// the timing acquisition; without it 0.56-3.2 s from ten of them, and never
// within the 4.4 s beacon from one.
func TestDPMRPipelineLocksFromEveryTimingPhase(t *testing.T) {
	const lead = 600 * 20 // the beacon's 600 leading noise dibits, in samples
	base := dpmrBeaconIQ(2, 40, 300, 25)
	for delay := 0; delay < 20; delay++ {
		iq := append(make([]complex64, delay), base...)
		bus := events.NewBus(4096)
		p, err := newDPMRPipeline(PipelineOptions{Bus: bus, SystemName: "Beacon", FrequencyHz: 446_100_000, SampleRateHz: 48_000})
		if err != nil {
			t.Fatal(err)
		}
		pp := p.(*dpmrPipeline)
		lockAt := -1
		for i := 0; i < len(iq) && lockAt < 0; i += 480 {
			pp.Process(iq[i:min(i+480, len(iq))])
			if pp.cc.Locked() {
				lockAt = i
			}
		}
		bus.Close()
		if ms := (lockAt - lead) * 1000 / 48_000; lockAt < 0 || ms > 400 {
			t.Errorf("start %d samples later: locked %d ms after the beacon began (never if negative)", delay, ms)
		}
	}
}

// Noise and an analog FM carrier never lock the pipeline.
func TestDPMRPipelineIgnoresOtherCarriers(t *testing.T) {
	const fs = 48_000.0
	r := rand.New(rand.NewSource(3))
	noise := make([]complex64, int(30*fs))
	tone := make([]complex64, int(10*fs))
	phase := 0.0
	for i := range noise {
		noise[i] = complex(float32(r.NormFloat64()), float32(r.NormFloat64()))
	}
	for i := range tone {
		phase += 2 * math.Pi * 2500 * math.Sin(2*math.Pi*1000*float64(i)/fs) / fs
		tone[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	for name, iq := range map[string][]complex64{"noise": noise, "FM tone": tone} {
		if locks, grants := runDPMRPipeline(t, iq); len(locks) != 0 || len(grants) != 0 {
			t.Errorf("%s: %d locks, %d grants", name, len(locks), len(grants))
		}
	}
}
