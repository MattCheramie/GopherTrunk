package conventional

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// p25TransmissionDibits builds n back-to-back LDUs (LDU1/LDU2 alternating)
// carrying nac, with random voice bits: one keyed P25 Phase 1 radio.
func p25TransmissionDibits(t *testing.T, nac uint16, n int, seed int64) []uint8 {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	var out []uint8
	for i := 0; i < n; i++ {
		var voice [phase1.LDUVoiceSubframeCount][]byte
		for j := range voice {
			voice[j] = make([]byte, phase1.LDUVoiceSubframeBits)
			for k := range voice[j] {
				voice[j][k] = byte(r.Intn(2))
			}
		}
		duid := phase1.DUIDLogicalLink1
		if i%2 == 1 {
			duid = phase1.DUIDLogicalLink2
		}
		var lces [phase1.LDULCESBlockCount][]byte
		var lsd [phase1.LDULSDBlockCount][]byte
		bits, err := phase1.AssembleLDU(nac, duid, voice, lces, lsd)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, framing.BitsToDibits(bits)...)
	}
	return out
}

// randomDibits is a P25-shaped carrier with no frame sync: modulated noise.
func randomDibits(n int, seed int64) []uint8 {
	r := rand.New(rand.NewSource(seed))
	d := make([]uint8, n)
	for i := range d {
		d[i] = uint8(r.Intn(4))
	}
	return d
}

// feedDetector runs iq through d in RTL-sized chunks and reports whether the
// gate was open at the end.
func feedDetector(d *NACDetector, iq []complex64) bool {
	open := false
	for _, c := range chunked(iq, 16384) {
		open = d.Process(c)
	}
	return open
}

func TestParseNAC(t *testing.T) {
	for in, want := range map[string]uint16{"293": 0x293, "0x293": 0x293, "F7E": 0xF7E, "f7f": 0xF7F, " 000 ": 0} {
		got, err := ParseNAC(in)
		if err != nil || got != want {
			t.Errorf("ParseNAC(%q) = %#x, %v; want %#x", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "1000", "29G", "-1"} {
		if _, err := ParseNAC(bad); err == nil {
			t.Errorf("ParseNAC(%q) accepted", bad)
		}
	}
}

// The gate opens on the wanted NAC, stays shut on another system's NAC (and
// says so), and shuts at once when a different NAC keys up mid-dwell.
func TestNACDetectorGatesOnNID(t *testing.T) {
	const rate = 48_000
	mod := func(d []uint8) []complex64 { return demod.ModulateP25C4FM(d, rate, 1800) }
	lead := mod(randomDibits(960, 1)) // 0.2 s of modulated noise for the loops to settle

	d := NewNACDetector(rate, 0x293, "")
	if !feedDetector(d, append(append([]complex64{}, lead...), mod(p25TransmissionDibits(t, 0x293, 6, 2))...)) {
		t.Fatal("gate did not open on a transmission carrying the wanted NAC 293")
	}

	d.Reset()
	if feedDetector(d, append(append([]complex64{}, lead...), mod(p25TransmissionDibits(t, 0x4A5, 6, 3))...)) {
		t.Fatal("gate opened on NAC 4A5 with nac 293 configured")
	}
	if nac, foreign := d.TakeForeign(); !foreign || nac != 0x4A5 {
		t.Fatalf("TakeForeign = %#x, %v; want 4A5, true", nac, foreign)
	}

	// Wanted NAC, then another radio on 4A5 keys up: the gate shuts on its
	// first NID instead of riding out the hold.
	d.Reset()
	feedDetector(d, append(append([]complex64{}, lead...), mod(p25TransmissionDibits(t, 0x293, 4, 4))...))
	if !d.Present() {
		t.Fatal("gate not open after the wanted transmission")
	}
	if feedDetector(d, mod(p25TransmissionDibits(t, 0x4A5, 2, 5))) {
		t.Fatal("gate still open after a different NAC keyed up")
	}

	// A carrier with no P25 frame sync never opens it.
	d.Reset()
	if feedDetector(d, mod(randomDibits(4800, 6))) {
		t.Fatal("gate opened on modulated noise with no NID")
	}
}

// NAC F7E ("receive any") opens on every NAC, but only on P25.
func TestNACDetectorReceiveAny(t *testing.T) {
	const rate = 48_000
	mod := func(d []uint8) []complex64 { return demod.ModulateP25C4FM(d, rate, 1800) }
	d := NewNACDetector(rate, NACReceiveAny, "")
	if !feedDetector(d, append(mod(randomDibits(960, 1)), mod(p25TransmissionDibits(t, 0x4A5, 6, 7))...)) {
		t.Fatal("receive-any gate did not open on NAC 4A5")
	}
	d.Reset()
	if feedDetector(d, mod(randomDibits(4800, 8))) {
		t.Fatal("receive-any gate opened on a carrier with no NID")
	}
}

// With p25_phase1_demod_mode cqpsk the gate reads NIDs off a linear (LSM)
// transmitter, which the C4FM path cannot.
func TestNACDetectorCQPSK(t *testing.T) {
	const rate = 48_000
	// LSM sends canonical dibits 2 and 3 swapped (the receiver's
	// lsmDibitRemap; see receiver/cqpsk_test.go dibitsToLSMIQ).
	lsm := func(d []uint8) []complex64 {
		onAir := make([]uint8, len(d))
		for i, v := range d {
			onAir[i] = [4]uint8{0, 1, 3, 2}[v&3]
		}
		return demod.ModulatePiOver4DQPSK(onAir, 10, 8, 0.20, math.Pi/4)
	}
	sig := append(lsm(randomDibits(960, 1)), lsm(p25TransmissionDibits(t, 0x293, 6, 9))...)
	if !feedDetector(NewNACDetector(rate, 0x293, "cqpsk"), sig) {
		t.Fatal("CQPSK gate did not open on an LSM transmission carrying NAC 293")
	}
}

// p25ScanStream is a P25 transmission at the scanner's SDR rate, at
// ≈ -20 dBFS (well over the -50 squelch), then receiver noise.
func p25ScanStream(t *testing.T, nac uint16) [][]complex64 {
	t.Helper()
	// Modulate at 48 kHz and interpolate to the SDR rate: a band-limited
	// transmitter, as on air. (ModulateP25C4FM run directly at 2.4 MS/s,
	// 500 samples per symbol, does not produce a decodable eye.)
	// Linear interpolation, not a polyphase filter: the scanner's own
	// decimating front end removes the images, and the filter took ~40 s
	// under -race.
	const up = amTestRate / 48_000
	tx48 := demod.ModulateP25C4FM(append(randomDibits(960, 11), p25TransmissionDibits(t, nac, 8, 12)...), 48_000, 1800)
	tx := make([]complex64, 0, len(tx48)*up)
	for i := 0; i+1 < len(tx48); i++ {
		a, b := tx48[i], tx48[i+1]
		for k := 0; k < up; k++ {
			f := float32(k) / up
			tx = append(tx, a+complex(f, 0)*(b-a))
		}
	}
	for i := range tx {
		tx[i] *= 0.1
	}
	r := rand.New(rand.NewSource(13))
	idle := make([]complex64, amTestRate/2)
	for i := range idle {
		idle[i] = complex(float32(r.NormFloat64()*1e-4), float32(r.NormFloat64()*1e-4))
	}
	return append(chunked(tx, 16384), chunked(idle, 16384)...)
}

// TestP25ChannelNACFilter is the #1262 regression: a mode p25 channel with
// nac 293 must not open on another system's P25 carrier (the #1239
// reporter's frequency carried their own units, the repeater and a third
// party's encrypted units, all recorded under one channel). Without the gate
// the carrier's power alone opened the channel.
func TestP25ChannelNACFilter(t *testing.T) {
	const freq = 155_752_500
	// The waits key on the data, not the clock: under -race on a CI runner
	// the scanner processes 2.4 MS/s IQ slower than real time, so a
	// wall-clock wait (or the default scan window) could expire before the
	// gate had seen a frame. MinDwellPerChannel is set long for the same
	// reason; on hardware the scanner keeps up with real time.
	run := func(nac uint16, wantStart bool) *fakeEngine {
		tuner := &fakeTuner{}
		eng := &fakeEngine{}
		iq := &fakeIQ{chunks: map[uint32][][]complex64{freq: p25ScanStream(t, nac)}, tuner: tuner}
		s, err := New(Options{
			Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
			DeviceSerial: "CONV-P25",
			Channels: []Channel{{
				Label: "VCALL10", FrequencyHz: freq, Mode: ModeP25,
				SquelchDbFS: -50, Hangtime: 200 * time.Millisecond,
				NAC: 0x293, NACSet: true, P25Phase1DemodMode: "c4fm",
			}},
			SampleRateHz:       amTestRate,
			MinDwellPerChannel: 30 * time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() { _ = s.Run(ctx) }()
		drained := func() bool {
			iq.mu.Lock()
			defer iq.mu.Unlock()
			return len(iq.chunks[freq]) == 0
		}
		deadline := time.Now().Add(2 * time.Minute)
		for time.Now().Before(deadline) {
			if wantStart && eng.startCount() > 0 {
				break
			}
			if !wantStart && drained() {
				// The scanner has the last chunks in flight; give it a
				// moment to process them before judging.
				time.Sleep(500 * time.Millisecond)
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		return eng
	}

	foreign := run(0x4A5, false)
	if n := foreign.startCount(); n != 0 {
		t.Fatalf("a P25 carrier with NAC 4A5 opened the nac-293 channel (%d call starts)", n)
	}

	own := run(0x293, true)
	if own.startCount() == 0 {
		t.Fatal("the channel's own NAC 293 did not open it")
	}
	own.mu.Lock()
	g := own.starts[0]
	own.mu.Unlock()
	if g.P25Phase1DemodMode != "c4fm" {
		t.Errorf("grant P25Phase1DemodMode = %q, want the channel's c4fm (the voice chain reads it from the grant)", g.P25Phase1DemodMode)
	}
}

// nac and p25_phase1_demod_mode apply to p25 channels only, and a bad demod
// is rejected.
func TestNACOnlyOnP25Channels(t *testing.T) {
	base := Options{Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Engine: &fakeEngine{}, Recorder: fakeRecorder{}, SampleRateHz: amTestRate}
	for name, ch := range map[string]Channel{
		"nac on fm":      {Label: "a", FrequencyHz: 1, Mode: "fm", NAC: 0x293, NACSet: true},
		"demod on fm":    {Label: "a", FrequencyHz: 1, Mode: "fm", P25Phase1DemodMode: "cqpsk"},
		"bad demod":      {Label: "a", FrequencyHz: 1, Mode: ModeP25, P25Phase1DemodMode: "gmsk"},
		"nac over 0xFFF": {Label: "a", FrequencyHz: 1, Mode: ModeP25, NAC: 0x1000, NACSet: true},
	} {
		o := base
		o.Channels = []Channel{ch}
		if _, err := New(o); err == nil {
			t.Errorf("%s: New accepted it", name)
		}
	}
}
