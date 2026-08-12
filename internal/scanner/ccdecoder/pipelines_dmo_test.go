package ccdecoder

import (
	"math"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// tchSpeechFrameBits / tchType3Bits mirror the unexported tetra constants: a TCH/S
// speech frame is 137 type-1 bits, and the full slot is 432 type-4 (post-FEC, pre-
// scramble) bits.
const (
	dmoTestSpeechBits = 137
	dmoTestType4Bits  = 432
)

func dmoSeqBits(seed, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte((seed + i*5) % 2)
	}
	return out
}

func dmoFiller(seed, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8((seed + i*3) & 3)
	}
	return out
}

// buildDMODibitStream synthesizes a DMO transmission's demodulated dibit stream: a
// long receiver lead-in, one DSB (SCH/S + SCH/H), then nDNB TCH/S DNBs scrambled with
// colour, separated by filler. It mirrors the buildDSB/buildDNB layout the tetra
// package tests use (EN 300 396-2 Tables 15/16), built from the exported encoders so
// it can live in the ccdecoder package.
func buildDMODibitStream(colour uint32, nDNB int) []uint8 {
	b2d := tetra.TetraBitsToDibits
	schs := b2d(tetra.EncodeBSCH(dmoSeqBits(1, 60)))
	schh := b2d(tetra.EncodeSCHHD(dmoSeqBits(2, 124), 0))

	var out []uint8
	out = append(out, dmoFiller(0, 900)...) // lead-in: let Gardner/AFC/equalizer converge

	// DSB: 40-dibit freq-corr + 60-dibit SCH/S + 19-dibit sync train + 108-dibit SCH/H.
	out = append(out, dmoFiller(1, 40)...)
	out = append(out, schs...)
	out = append(out, tetra.SyncTrainingDibits()...)
	out = append(out, schh...)

	for i := 0; i < nDNB; i++ {
		frameA := dmoSeqBits(7+i, dmoTestSpeechBits)
		frameB := dmoSeqBits(9+i, dmoTestSpeechBits)
		type4 := framing.UnpackBitsMSB(tetra.EncodeTCHS(frameA, frameB), dmoTestType4Bits)
		onair := framing.ScrambleTetra(type4, colour)
		blk1 := b2d(onair[:dmoTestType4Bits/2])
		blk2 := b2d(onair[dmoTestType4Bits/2:])
		out = append(out, dmoFiller(11+i, 30)...)
		out = append(out, blk1...)
		out = append(out, tetra.NormalSyncDibits()...)
		out = append(out, blk2...)
	}
	out = append(out, dmoFiller(99, 600)...) // trailing filler so the last burst is inside the window
	return out
}

// TestTETRADMOPipelineLocksAndGrants drives newTETRADMOPipeline with a synthetic DMO
// transmission modulated to the 144 kHz channel rate (the production DDC target) and
// asserts the production control path: it locks on the DSB SCH/S (publishing
// cc.locked), brute-force-recovers the DM traffic colour code, and publishes a
// tetra-dmo grant so the composer starts a voice chain. This is the production-path
// analogue of the offline TestTETRADMOReplay — a green synthetic decode is a
// necessary (not sufficient) gate; on-air A/B on the operator's capture is still
// required to close #1003 (#764/#771).
func TestTETRADMOPipelineLocksAndGrants(t *testing.T) {
	const (
		sampleRate = 144_000.0 // ddcTargetForProtocol(ProtocolTETRADMO)
		sps        = 8          // 144000 / 18000
		span       = 8
		alpha      = 0.35
		freqHz     = 438_900_000
		testColour = uint32(3) // the DM traffic colour to recover (0 = config default)
		nDNB       = 40
	)

	dibits := buildDMODibitStream(testColour, nDNB)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)
	// Light noise floor: the blind CMA equalizer's constant-modulus adaptation is
	// well-defined only against a signal with a noise floor (mirrors the TMO
	// integration test + receiver clean-channel tests).
	iq = demod.ApplyImpairments(iq, sampleRate, demod.Impairments{SNRdB: 40, Seed: 7})

	bus := events.NewBus(256)
	defer bus.Close()
	sub := bus.Subscribe()
	var (
		mu     sync.Mutex
		locked bool
		grants []trunking.Grant
	)
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for ev := range sub.C {
			mu.Lock()
			switch ev.Kind {
			case events.KindCCLocked:
				locked = true
			case events.KindGrant:
				if g, ok := ev.Payload.(trunking.Grant); ok {
					grants = append(grants, g)
				}
			}
			mu.Unlock()
		}
	}()

	pl, err := newTETRADMOPipeline(PipelineOptions{
		Bus:          bus,
		SystemName:   "DMO",
		FrequencyHz:  freqHz,
		SampleRateHz: sampleRate,
		System:       trunking.System{Protocol: trunking.ProtocolTETRADMO},
	})
	if err != nil {
		t.Fatalf("newTETRADMOPipeline: %v", err)
	}
	dmo := pl.(*tetraDMOPipeline)

	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		pl.Process(iq[i:end])
	}

	// Inspect the pipeline's own decode state (same package → unexported access).
	if !dmo.locked {
		t.Errorf("pipeline did not lock (dsb_total=%d dsb_crc=%d)", dmo.dsbTotal, dmo.dsbCRC)
	}
	if dmo.dsbCRC == 0 {
		t.Errorf("no CRC-valid DSB SCH/S decoded")
	}
	if !dmo.colourKnown {
		t.Errorf("DM colour code not recovered (dnb_total=%d tch_crc=%d)", dmo.dnbTotal, dmo.tchCRC)
	} else if dmo.colour != testColour {
		t.Errorf("recovered colour=%d, want %d", dmo.colour, testColour)
	}
	if dmo.tchCRC == 0 {
		t.Errorf("no CRC-valid TCH/S decoded at the recovered colour")
	}

	bus.Close()
	<-drainDone
	mu.Lock()
	defer mu.Unlock()
	if !locked {
		t.Errorf("no cc.locked event published")
	}
	if len(grants) == 0 {
		t.Fatalf("no tetra-dmo grant published")
	}
	g := grants[0]
	if g.Protocol != "tetra-dmo" {
		t.Errorf("grant protocol=%q, want tetra-dmo", g.Protocol)
	}
	if g.FrequencyHz != freqHz {
		t.Errorf("grant freq=%d, want %d", g.FrequencyHz, freqHz)
	}
}
