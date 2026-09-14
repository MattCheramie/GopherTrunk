package ccdecoder

import (
	"io"
	"log/slog"
	"math"
	"math/cmplx"
	"math/rand"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Direct-mode (DMR Tier I / simplex) timing per ETSI TS 102 361-1 §4.2: a
// 60 ms TDMA frame of two 30 ms slots; a direct-mode MS transmits one 27.5 ms
// burst (132 dibits at 4800 baud) per frame and is OFF for the other 32.5 ms
// (156 dibit periods). A base station fills both slots, which is what every
// earlier synthetic DMR fixture modelled.
const (
	dmFrameDibits = 288 // 60 ms
	dmOffDibits   = dmFrameDibits - dmr.BurstDibits
)

// dmVoiceLCHeaderBurst is one direct-mode Voice LC Header burst (a DATA burst,
// framed by the DM data sync) — the same construction as the siglab Tier I
// fixture.
func dmVoiceLCHeaderBurst(colorCode uint8, groupID, sourceID uint32) []uint8 {
	flcBytes := dmr.AssembleFLC(dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: groupID, SrcAddr: sourceID})
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedVoiceLCHeader[i]
	}
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (cw[i>>3] >> uint(7-(i&7))) & 1
	}
	payload := framing.BitsToDibits(framing.EncodeBPTC196_96(bits))
	slot := framing.BitsToDibits(dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTVoiceLCHeader}))
	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payload[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slot[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.DMData1.Dibits[:]...)
	burst = append(burst, slot[dmr.SlotTypeDibits:]...)
	burst = append(burst, payload[dmr.HalfPayloadDibits:]...)
	return burst
}

// dmVoiceBurst is a direct-mode voice burst with a random (AMBE-like) payload
// around the DM voice sync — balanced filler with no embedded LC, so a call
// is granted from its Voice LC Headers alone, never by late entry.
func dmVoiceBurst(rng *rand.Rand) []uint8 {
	burst := make([]uint8, 0, dmr.BurstDibits)
	for i := 0; i < dmr.HalfPayloadDibits+dmr.SlotTypeDibits; i++ {
		burst = append(burst, uint8(rng.Intn(4)))
	}
	burst = append(burst, dmr.DMVoice1.Dibits[:]...)
	for len(burst) < dmr.BurstDibits {
		burst = append(burst, uint8(rng.Intn(4)))
	}
	return burst
}

// directModeTransmissionIQ lays `ptts` direct-mode transmissions (PTTs) of one
// source on one talkgroup on the real 60 ms frame grid, the way a handheld
// transmits them: 3 Voice LC Header bursts at keyup, then voiceFrames voice
// bursts, each burst followed by 32.5 ms of transmitter-OFF time rendered as
// receiver noise; 20 idle frames (1.2 s) before, between and after the
// PTTs. The carrier sits offsetHz off centre (a tuner ppm error) and
// noiseSigma is AWGN per axis against a unit-amplitude carrier (0.03 ≈ 27 dB
// SNR).
func directModeTransmissionIQ(ptts, voiceFrames int, offsetHz, noiseSigma float64, seed int64) []complex64 {
	const sampleRate, sps, modSpan = 48_000.0, 10, 8
	const headers, idleFrames = 3, 20
	rng := rand.New(rand.NewSource(seed))
	header := dmVoiceLCHeaderBurst(0x1, 99, 0x123456)
	perPTT := headers + voiceFrames
	total := idleFrames + ptts*(perPTT+idleFrames)
	on := make([]bool, total)
	dibits := make([]uint8, 0, total*dmFrameDibits)
	f := 0
	appendFrame := func(burst []uint8, live bool) {
		on[f] = live
		if burst == nil {
			for i := 0; i < dmr.BurstDibits; i++ {
				dibits = append(dibits, uint8(rng.Intn(4)))
			}
		} else {
			dibits = append(dibits, burst...)
		}
		for i := 0; i < dmOffDibits; i++ {
			dibits = append(dibits, uint8(rng.Intn(4)))
		}
		f++
	}
	for i := 0; i < idleFrames; i++ {
		appendFrame(nil, false)
	}
	for p := 0; p < ptts; p++ {
		for i := 0; i < headers; i++ {
			appendFrame(header, true)
		}
		for i := 0; i < voiceFrames; i++ {
			appendFrame(dmVoiceBurst(rng), true)
		}
		for i := 0; i < idleFrames; i++ {
			appendFrame(nil, false)
		}
	}
	iq := demod.ModulateC4FM(dibits, sps, modSpan, 0.20, sampleRate, 1944.0)
	frameSamples := dmFrameDibits * sps
	burstSamples := dmr.BurstDibits * sps
	delay := modSpan * sps // RRC shaping-filter group delay
	guard := sps
	for f := 0; f < total; f++ {
		base := f*frameSamples + delay
		for i := -delay; i < frameSamples-delay; i++ {
			j := base + i
			if j < 0 || j >= len(iq) {
				continue
			}
			if !(on[f] && i >= -guard && i < burstSamples+guard) {
				iq[j] = 0
			}
		}
	}
	for i := range iq {
		if offsetHz != 0 {
			iq[i] *= complex64(cmplx.Rect(1, 2*math.Pi*offsetHz*float64(i)/sampleRate))
		}
		iq[i] += complex(float32(rng.NormFloat64()*noiseSigma), float32(rng.NormFloat64()*noiseSigma))
	}
	return iq
}

// runDMRConventionalPipeline drives iq through the production Tier I or Tier
// II pipeline in RTL-sized chunks and reports whether it locked, how many
// grants it published, and the Tier II state machine's counters.
func runDMRConventionalPipeline(t *testing.T, proto trunking.Protocol, iq []complex64) (locked bool, grants int, cnt tier2.Counters) {
	t.Helper()
	bus := events.NewBus(4096)
	sub := bus.Subscribe()
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range sub.C {
			mu.Lock()
			switch ev.Kind {
			case events.KindCCLocked:
				locked = true
			case events.KindGrant:
				grants++
			}
			mu.Unlock()
		}
	}()
	opts := PipelineOptions{
		Bus:          bus,
		Log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName:   "dmr-simplex",
		FrequencyHz:  446_500_000,
		SampleRateHz: 48_000,
		System:       trunking.System{Protocol: proto},
	}
	var pl ProtocolPipeline
	var cc *tier2.ConventionalChannel
	var err error
	switch proto {
	case trunking.ProtocolDMRTier1:
		pl, err = newDMRTier1Pipeline(opts)
		if err == nil {
			cc = pl.(*dmrTier1Pipeline).cc
		}
	case trunking.ProtocolDMRTier2:
		pl, err = newDMRTier2Pipeline(opts)
		if err == nil {
			cc = pl.(*dmrTier2Pipeline).cc
		}
	default:
		t.Fatalf("unsupported protocol %v", proto)
	}
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		pl.Process(iq[i:end])
	}
	bus.Close()
	<-done
	mu.Lock()
	defer mu.Unlock()
	return locked, grants, cc.Counters()
}

// TestDMRPipelinesDecodeDirectModeTransmission is the issue #836 regression at
// the pipeline level: direct-mode (simplex) transmissions — 27.5 ms bursts
// with 32.5 ms of receiver noise between them, the cadence every DMR handheld
// on a simplex channel transmits — must lock and grant through BOTH the Tier
// I pipeline (the direct-mode sync set) and the Tier II pipeline (all nine
// syncs — the `dmr-tier2` protocol the dmr-simplex sample config uses). Before
// the receiver's carrier-presence gate, both decoded ZERO sync words from
// this stream at 27 dB SNR (the inter-burst noise inflated the symbol AGC so
// every outer symbol sliced as an inner one), while every earlier fixture —
// bursts laid back-to-back like a base station — passed.
//
// With the reporter's measured −1.2 kHz tuner offset the coarse acquirer
// (fed only present samples) engages ~0.5 s into the FIRST transmission and
// stays frozen, so that PTT's keyup headers pass uncorrected (the post-clock
// AFC alone is marginal at 1.2 kHz) but every later PTT decodes from its
// first burst: the fixture carries two PTTs and the second must grant.
func TestDMRPipelinesDecodeDirectModeTransmission(t *testing.T) {
	for _, tc := range []struct {
		name      string
		proto     trunking.Protocol
		offsetHz  float64
		minGrants int
	}{
		{"tier1 centred", trunking.ProtocolDMRTier1, 0, 2},
		{"tier2 centred", trunking.ProtocolDMRTier2, 0, 2},
		{"tier1 -1.2 kHz tuner offset", trunking.ProtocolDMRTier1, -1200, 1},
		{"tier2 -1.2 kHz tuner offset", trunking.ProtocolDMRTier2, -1200, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			iq := directModeTransmissionIQ(2, 30, tc.offsetHz, 0.03, 1)
			locked, grants, c := runDMRConventionalPipeline(t, tc.proto, iq)
			t.Logf("locked=%v grants=%d sync_hits=%d bursts=%d fec_pass=%d fec_fail=%d",
				locked, grants, c.SyncHits, c.Bursts, c.FECPass, c.FECFail)
			if !locked {
				t.Errorf("pipeline never locked on a direct-mode transmission (sync_hits=%d fec_pass=%d)", c.SyncHits, c.FECPass)
			}
			// One grant per PTT: the three keyup headers de-duplicate into one
			// grant and the voice bursts carry no LC, so more than two grants
			// means the state machine is losing and re-finding a call.
			if grants < tc.minGrants || grants > 2 {
				t.Errorf("grants = %d for two direct-mode PTTs, want %d..2", grants, tc.minGrants)
			}
			if c.FECPass < 2 {
				t.Errorf("fec_pass = %d Voice LC Headers, want >= 2 (the second PTT's keyup)", c.FECPass)
			}
		})
	}
}
