package widebandt2

import (
	"context"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestEngineEndToEndT2GrantFromSynthesizedIQ proves the wideband
// engine's wiring lands a real DMR Tier II grant event when fed
// synthesized C4FM IQ:
//
//	dibit-sequence (Voice LC Header bursts)
//	  → demod.ModulateC4FM at 48 kHz (matches the bank's per-tap rate)
//	  → mock sdr.Device streaming IQ chunks
//	  → widebandt2.Engine
//	    → tuner.DDCBank tap @ offset=0
//	      → dmr/receiver.Receiver (RRC matched filter + clock recovery)
//	        → tier2.ConventionalChannel.Process
//	          → events.KindGrant on the bus
//
// The test runs at SampleRateHz=48 kHz so the bank's per-tap
// resampler is a no-op (L=M=1) and the pulse-shape leaving the
// modulator matches exactly what the receiver's matched filter
// expects. The full decimation case (SDR streaming at 240 kHz / 2.4
// MS/s and the bank's resampler down-converting to 48 kHz with a
// non-zero tap offset) introduces a pulse-shape mismatch between
// the modulator's RRC and the receiver's RRC-after-Kaiser-LP that's
// shared with the existing single-frequency ccdecoder DDC path —
// covering that requires a TX-side filter cascade that mirrors the
// RX matched filter through the DDC, which is a separate exercise.
func TestEngineEndToEndT2GrantFromSynthesizedIQ(t *testing.T) {
	const (
		widebandRateHz = 48_000.0 // bank tap rate; resampler is L=M=1 here
		centerHz       = 460_000_000
		offsetHz       = 0
		repeaterHz     = centerHz + offsetHz
		spsWideband    = 10 // 48_000 / 4800
		spanSymbols    = 8
		alpha          = 0.20
		colorCode      = uint8(0x7)
		groupID        = uint32(0x123)
		sourceID       = uint32(0x456789)
		burstRepeats   = 200 // give the Mueller-Müller loop ample time to lock
		chunkSamples   = 4800
	)

	// 1. Build the dibit stream + modulate to baseband IQ at the
	// wideband rate.
	dibits := buildT2VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	baseband := demod.ModulateC4FM(dibits, spsWideband, spanSymbols, alpha, widebandRateHz, dmrDeviationHz)

	// 2. Optionally shift the baseband by offsetHz. At offsetHz=0
	// this is a no-op; kept as a seam so a future variant of this
	// test can place the carrier away from DC once the modulator-
	// side filter cascade is in place.
	shifted := shiftComplex(baseband, offsetHz, widebandRateHz)

	// 3. Chunk it for the mock device.
	chunks := chunkComplex(shifted, chunkSamples)
	dev := newMockDevice(chunks)

	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: uint32(widebandRateHz),
		CenterFreqHz: centerHz,
		Channels: []ChannelConfig{
			{FrequencyHz: repeaterHz, SystemName: "regional-t2"},
		},
		Systems: []trunking.System{t2System("regional-t2")},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 4. Drive the engine — Run returns when the stream closes.
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- e.Run(ctx) }()

	// 5. Wait for the Grant event.
	deadline := time.After(8 * time.Second)
	var sawGrant, sawLock bool
	var grantGroup, grantSource uint32
	var grantFreq uint32
WaitLoop:
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				sawLock = true
			case events.KindGrant:
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Errorf("KindGrant payload type = %T, want trunking.Grant", ev.Payload)
					continue
				}
				grantGroup = g.GroupID
				grantSource = g.SourceID
				grantFreq = g.FrequencyHz
				if g.Protocol != "dmr-tier2" {
					t.Errorf("grant Protocol = %q, want dmr-tier2", g.Protocol)
				}
				sawGrant = true
				break WaitLoop
			}
		case <-deadline:
			break WaitLoop
		}
	}
	cancel()
	<-runDone

	if !sawLock {
		t.Errorf("no cc.locked event observed - receiver never synced")
	}
	if !sawGrant {
		t.Fatalf("no grant event observed within deadline (lock=%v)", sawLock)
	}
	if grantGroup != groupID {
		t.Errorf("grant.GroupID = %#x, want %#x", grantGroup, groupID)
	}
	if grantSource != sourceID {
		t.Errorf("grant.SourceID = %#x, want %#x", grantSource, sourceID)
	}
	if grantFreq != repeaterHz {
		t.Errorf("grant.FrequencyHz = %d, want %d", grantFreq, repeaterHz)
	}
}

// TestEngineEndToEndT3GrantFromSynthesizedIQ proves the wideband
// engine resolves a DMR Tier III voice grant's LCN to a downlink
// frequency via the system's dmr_band_plan and publishes it on the
// bus. Same synthesized-IQ path as the Tier II test, but the bursts
// carry an OpTVGrant CSBK (DTCSBK) instead of a Voice LC Header, and
// the system is given a LinearBandPlan so resolveLCN succeeds.
//
// LCN 8 on {base 866 MHz, spacing 25 kHz, offset 1} resolves to
// 866.175 MHz — the same vector asserted by tier3's own
// control_test.go, so this test pins the *wiring* (Options.Resolver
// reaching the decoder), not the resolver math.
func TestEngineEndToEndT3GrantFromSynthesizedIQ(t *testing.T) {
	const (
		widebandRateHz = 48_000.0
		centerHz       = 851_000_000
		ccFreqHz       = centerHz
		spsWideband    = 10
		spanSymbols    = 8
		alpha          = 0.20
		colorCode      = uint8(0x3)
		burstRepeats   = 200
		chunkSamples   = 4800
		wantLCN        = uint16(8)
		wantFreqHz     = uint32(866_175_000) // 866M + (8-1)×25k
		wantGroup      = uint32(0x123456)
		wantSource     = uint32(0xABCDEF)
	)

	sys := t3System("regional-t3", ccFreqHz)
	sys.DMRBandPlan = &trunking.DMRBandPlan{
		Linear: &trunking.DMRLinearBandPlan{BaseHz: 866_000_000, SpacingHz: 25_000, Offset: 1},
	}

	dibits := buildT3TVGrantDibits(burstRepeats, colorCode)
	baseband := demod.ModulateC4FM(dibits, spsWideband, spanSymbols, alpha, widebandRateHz, dmrDeviationHz)
	chunks := chunkComplex(baseband, chunkSamples)
	dev := newMockDevice(chunks)

	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: uint32(widebandRateHz),
		CenterFreqHz: centerHz,
		Channels:     []ChannelConfig{{FrequencyHz: ccFreqHz, SystemName: sys.Name}},
		Systems:      []trunking.System{sys},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- e.Run(ctx) }()

	deadline := time.After(8 * time.Second)
	var g trunking.Grant
	var sawGrant bool
WaitLoop:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			gg, ok := ev.Payload.(trunking.Grant)
			if !ok {
				t.Errorf("KindGrant payload type = %T", ev.Payload)
				continue
			}
			g = gg
			sawGrant = true
			break WaitLoop
		case <-deadline:
			break WaitLoop
		}
	}
	cancel()
	<-runDone

	if !sawGrant {
		t.Fatal("no grant event observed within deadline")
	}
	if g.Protocol != "dmr-tier3" {
		t.Errorf("grant.Protocol = %q, want dmr-tier3", g.Protocol)
	}
	if g.ChannelNum != wantLCN {
		t.Errorf("grant.ChannelNum (LCN) = %d, want %d", g.ChannelNum, wantLCN)
	}
	if g.FrequencyHz != wantFreqHz {
		t.Errorf("grant.FrequencyHz = %d, want %d (LCN→Hz resolver not wired?)", g.FrequencyHz, wantFreqHz)
	}
	if g.GroupID != wantGroup || g.SourceID != wantSource {
		t.Errorf("grant group/source = %X/%X, want %X/%X", g.GroupID, g.SourceID, wantGroup, wantSource)
	}
}

// TestEngineEndToEndT3GrantDroppedWithoutBandPlan is the negative
// control for the test above: with no dmr_band_plan the decoder has no
// resolver, so the same TVGrant CSBK must produce a decode.error with
// stage=no-bandplan and NO grant event. This proves the band plan is
// the load-bearing piece, not an incidental one.
func TestEngineEndToEndT3GrantDroppedWithoutBandPlan(t *testing.T) {
	const (
		widebandRateHz = 48_000.0
		centerHz       = 851_000_000
		colorCode      = uint8(0x3)
		burstRepeats   = 200
		chunkSamples   = 4800
	)

	sys := t3System("regional-t3", centerHz) // no DMRBandPlan
	dibits := buildT3TVGrantDibits(burstRepeats, colorCode)
	baseband := demod.ModulateC4FM(dibits, 10, 8, 0.20, widebandRateHz, dmrDeviationHz)
	dev := newMockDevice(chunkComplex(baseband, chunkSamples))

	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: uint32(widebandRateHz),
		CenterFreqHz: centerHz,
		Channels:     []ChannelConfig{{FrequencyHz: centerHz, SystemName: sys.Name}},
		Systems:      []trunking.System{sys},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- e.Run(ctx) }()

	deadline := time.After(8 * time.Second)
	var sawNoBandPlan bool
WaitLoop:
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindGrant:
				t.Fatalf("unexpected grant without a band plan: %+v", ev.Payload)
			case events.KindDecodeError:
				if de, ok := ev.Payload.(events.DecodeError); ok && de.Stage == events.StageNoBandPlan {
					sawNoBandPlan = true
					break WaitLoop
				}
			}
		case <-deadline:
			break WaitLoop
		}
	}
	cancel()
	<-runDone

	if !sawNoBandPlan {
		t.Fatal("expected a decode.error stage=no-bandplan event, got none")
	}
}

// buildT3TVGrantDibits builds a dibit stream of repeated OpTVGrant CSBK
// bursts (DTCSBK) for the synthesized-IQ end-to-end tests. The CSBK
// payload mirrors tier3/control_test.go's TVGrant vector: LPCN 8 in the
// leading 12 bits (p[1]=0x88 → TS2), group 0x123456 at octets 2-4,
// source 0xABCDEF at octets 5-7. Burst framing matches
// buildT2VoiceLCHeaderDibits.
func buildT3TVGrantDibits(repeats int, colorCode uint8) []uint8 {
	info := tier3.AssembleCSBK(tier3.CSBK{
		LB:      true,
		Opcode:  tier3.OpTVGrant,
		Payload: [8]byte{0x00, 0x88, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF},
	})
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channelBits := framing.EncodeBPTC196_96(bits)
	payloadDibits := framing.BitsToDibits(channelBits)

	slotBits := dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTCSBK})
	slotDibits := framing.BitsToDibits(slotBits)

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)

	out := make([]uint8, 0, 800+repeats*(len(burst)+32)+100)
	for i := 0; i < 800; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, burst...)
		for i := 0; i < 32; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// buildT2VoiceLCHeaderDibits is a local copy of the
// integration_cc_dmr_tier2_test.go helper, kept here so this
// package's end-to-end test doesn't depend on cmd/gophertrunk
// internals. Same shape, same parameters, same FEC chain.
func buildT2VoiceLCHeaderDibits(repeats int, colorCode uint8, groupID, sourceID uint32) []uint8 {
	flc := dmr.FLC{
		FLCO:    dmr.FLCOGroupVoiceUser,
		DstAddr: groupID,
		SrcAddr: sourceID,
	}
	flcBytes := dmr.AssembleFLC(flc)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedVoiceLCHeader[i]
	}
	info := cw[:]
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channelBits := framing.EncodeBPTC196_96(bits)
	payloadDibits := framing.BitsToDibits(channelBits)

	slotBits := dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTVoiceLCHeader})
	slotDibits := framing.BitsToDibits(slotBits)

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)

	out := make([]uint8, 0, 800+repeats*(len(burst)+32)+100)
	for i := 0; i < 800; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, burst...)
		for i := 0; i < 32; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// shiftComplex multiplies an IQ buffer by e^{j 2π f n / Fs}, moving
// a baseband signal up to a non-zero centre frequency inside the
// wideband stream. Float64 phase accumulator avoids drift across
// the test's hundreds of thousands of samples.
func shiftComplex(in []complex64, offsetHz, sampleRateHz float64) []complex64 {
	out := make([]complex64, len(in))
	dtheta := 2 * math.Pi * offsetHz / sampleRateHz
	theta := 0.0
	for i, x := range in {
		c := float32(math.Cos(theta))
		s := float32(math.Sin(theta))
		r := real(x)*c - imag(x)*s
		im := real(x)*s + imag(x)*c
		out[i] = complex(r, im)
		theta += dtheta
		if theta > 2*math.Pi {
			theta -= 2 * math.Pi
		}
	}
	return out
}

// chunkComplex slices src into per-chunkSize buffers (the last chunk
// may be short). A copy is taken per chunk so the mock device's
// readers can hold them after we return.
func chunkComplex(src []complex64, chunkSize int) [][]complex64 {
	if chunkSize <= 0 {
		chunkSize = 4096
	}
	var out [][]complex64
	for i := 0; i < len(src); i += chunkSize {
		end := i + chunkSize
		if end > len(src) {
			end = len(src)
		}
		buf := make([]complex64, end-i)
		copy(buf, src[i:end])
		out = append(out, buf)
	}
	return out
}

// discardWriter silences engine logging during the test so the
// `go test -v` output stays focused on the assertions.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
