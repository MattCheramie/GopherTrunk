package phase1

import (
	"log/slog"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// buildSoftCCStream assembles a control-channel dibit stream (warmup +
// repeats × (FSW + NID + TSBK) with status symbols injected) plus the
// parallel per-bit LLR track (2 per dibit, MSB then LSB). AWGN of the given
// sigma is applied ONLY to the TSBK channel dibits — the FSW/NID stay clean
// so framing always succeeds and the TSBK trellis is the only thing under
// test. Hard dibits are sliced from the same noisy LLRs, so the hard and
// soft decodes see one consistent channel. Status symbols carry erasure
// (0, 0) LLRs; the gather strips them by position either way.
func buildSoftCCStream(nac uint16, repeats int, sigma float64, rng *rand.Rand) ([]uint8, []float32) {
	type softDibit struct {
		d    uint8
		m, l float32
	}
	ideal := func(d uint8) softDibit {
		sd := softDibit{d: d, m: 1, l: 1}
		if (d>>1)&1 == 1 {
			sd.m = -1
		}
		if d&1 == 1 {
			sd.l = -1
		}
		return sd
	}
	noisy := func(d uint8) softDibit {
		sd := ideal(d)
		vm := float64(sd.m) + rng.NormFloat64()*sigma
		vl := float64(sd.l) + rng.NormFloat64()*sigma
		sd.m, sd.l = float32(vm), float32(vl)
		var msb, lsb uint8
		if vm < 0 {
			msb = 1
		}
		if vl < 0 {
			lsb = 1
		}
		sd.d = msb<<1 | lsb
		return sd
	}

	tsbkInfo := AssembleTSBK(TSBK{LB: true, Opcode: OpRFSSStatusBroadcast})

	frame := make([]softDibit, 0, 24+32+98)
	for _, d := range FrameSyncWord {
		frame = append(frame, ideal(d))
	}
	nidBits := EncodeNIDBits(nac, DUIDTrunkingSignaling)
	for i := 0; i < 32; i++ {
		frame = append(frame, ideal((nidBits[2*i]<<1)|nidBits[2*i+1]))
	}
	for _, d := range EncodeTSBKChannel(tsbkInfo) {
		frame = append(frame, noisy(d))
	}
	// Inject status symbols in lockstep with InjectControlStatusSymbols'
	// cadence (one after every p25StatusStride-1 data dibits), carrying
	// erasure LLRs.
	const dataRun = p25StatusStride - 1
	injected := make([]softDibit, 0, len(frame)+len(frame)/dataRun+1)
	for i, sd := range frame {
		injected = append(injected, sd)
		if i%dataRun == dataRun-1 {
			injected = append(injected, softDibit{d: uint8((i / dataRun) & 3)})
		}
	}

	var stream []softDibit
	for i := 0; i < 200; i++ {
		stream = append(stream, ideal(uint8(i&3)))
	}
	for r := 0; r < repeats; r++ {
		stream = append(stream, injected...)
		for i := 0; i < 50; i++ {
			stream = append(stream, ideal(0))
		}
	}
	for i := 0; i < 100; i++ {
		stream = append(stream, ideal(0))
	}

	dibits := make([]uint8, len(stream))
	llrs := make([]float32, 2*len(stream))
	for i, sd := range stream {
		dibits[i] = sd.d
		llrs[2*i] = sd.m
		llrs[2*i+1] = sd.l
	}
	return dibits, llrs
}

// runSoftCC feeds the stream through a fresh ControlChannel in chunks,
// optionally stashing the LLRs before each Process call (the receiver's
// BitLLRSink → StashSoft contract), and returns the CRC-clean TSBK count.
func runSoftCC(t *testing.T, dibits []uint8, llrs []float32, soft bool) int64 {
	t.Helper()
	bus := events.NewBus(64)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "soft-test", FrequencyHz: 851_000_000})
	const chunk = 64
	for i := 0; i < len(dibits); i += chunk {
		end := i + chunk
		if end > len(dibits) {
			end = len(dibits)
		}
		if soft {
			cc.StashSoft(llrs[2*i:2*end], i)
		}
		cc.Process(dibits[i:end], i)
	}
	return cc.Stats().TSBKDecoded
}

// TestControlChannelSoftTSBKDecodesWhereHardFails is the failing-first
// end-to-end pin for p25_phase1_soft_decision at the control-channel level:
// at an AWGN level (seeded — deterministic) where every hard TSBK trellis
// decode fails, the StashSoft path's per-bit soft Viterbi recovers TSBKs
// through the same framing, buffering, status-symbol stripping and CRC
// gate. Before the soft path existed the soft count below was structurally
// identical to the hard count (0).
func TestControlChannelSoftTSBKDecodesWhereHardFails(t *testing.T) {
	const sigma = 0.85
	rng := rand.New(rand.NewSource(7))
	dibits, llrs := buildSoftCCStream(0x293, 12, sigma, rng)

	hard := runSoftCC(t, dibits, llrs, false)
	soft := runSoftCC(t, dibits, llrs, true)
	t.Logf("sigma=%.2f: hard TSBKs=%d soft TSBKs=%d", sigma, hard, soft)
	if hard != 0 {
		t.Errorf("hard path decoded %d TSBKs at a noise level chosen to defeat it — fixture no longer discriminates", hard)
	}
	if soft == 0 {
		t.Error("soft path decoded no TSBKs (StashSoft plumbing or soft trellis broken)")
	}
}

// TestControlChannelSoftCleanMatchesHard: on a clean stream the soft path
// must decode exactly the TSBKs the hard path does — no regression when
// the channel needs no help.
func TestControlChannelSoftCleanMatchesHard(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	dibits, llrs := buildSoftCCStream(0x293, 8, 0, rng)

	hard := runSoftCC(t, dibits, llrs, false)
	soft := runSoftCC(t, dibits, llrs, true)
	if hard == 0 {
		t.Fatal("clean fixture decoded no TSBKs on the hard path (fixture broken)")
	}
	if soft != hard {
		t.Errorf("clean stream: soft TSBKs=%d != hard TSBKs=%d", soft, hard)
	}
}

// TestControlChannelSoftSurvivesMissingStash: a chunk whose LLRs were never
// stashed (a receiver hiccup) must drop the soft track for that stretch and
// decode hard — never panic, never mis-align — and later stashed chunks
// restore the soft path once the buffer realigns.
func TestControlChannelSoftSurvivesMissingStash(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	dibits, llrs := buildSoftCCStream(0x293, 8, 0, rng)

	bus := events.NewBus(64)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "soft-test", FrequencyHz: 851_000_000})
	const chunk = 64
	n := 0
	for i := 0; i < len(dibits); i += chunk {
		end := i + chunk
		if end > len(dibits) {
			end = len(dibits)
		}
		// Skip the stash on every fifth chunk.
		if n%5 != 4 {
			cc.StashSoft(llrs[2*i:2*end], i)
		}
		cc.Process(dibits[i:end], i)
		n++
	}
	if got := cc.Stats().TSBKDecoded; got == 0 {
		t.Errorf("no TSBKs decoded with intermittent stashes (hard fallback broken): %d", got)
	}
}
