package receiver

import (
	"io"
	"log/slog"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// TestReceiverGardnerLocksTETRACCAt144kHz drives a fully
// SCH/HD-channel-coded MLE SYSINFO stream through the *production*
// receiver configuration — 144 kHz IQ → 8 samples/symbol → Gardner
// timing recovery (gain 0.005, the newTETRAPipeline value) →
// π/4-DQPSK decode → tetra.ControlChannel — and asserts a real
// cc.locked carrying the encoded LocationArea.
//
// This closes the coverage gap behind issue #553. The reporter's USRP
// runs at 2 MHz → 144 kHz = 8 sps (ddc.tetraDDCTargetRateHz), but the
// only end-to-end TETRA lock test ran at 72 kHz / 4 sps (the
// integration fixture, picked so sps rounds to an exact integer) and
// the existing 144 kHz receiver tests only assert dibits land in
// 0..3 — none proved the production 8-sps path actually acquires
// symbol timing well enough to recover a channel-coded burst and
// lock. It does: the issue's premise that the Gardner loop is
// "hard-coded for 4 sps" is false — NewGardner takes sps as a
// parameter (receiver.go), and this test exercises the real 8-sps
// path to a lock. It is the regression guard that any future change
// to the down-converter target, sps math, or Gardner loop must keep
// green at the production rate.
func TestReceiverGardnerLocksTETRACCAt144kHz(t *testing.T) {
	const (
		controlFreqHz        = 412_062_500
		colourCode           = uint32(0x12345)
		locationArea  uint16 = 0x42
		sps                  = 8
		span                 = 8
		alpha                = 0.35
		// The production newTETRAPipeline GardnerGain. This test
		// guards that the production rate + gain lock end-to-end.
		gardnerGain = 0.005
	)

	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := tetra.New(tetra.Options{
		Bus:         bus,
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName:  "TETRASite",
		FrequencyHz: controlFreqHz,
	})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	cc.SetExpectedChannel(tetra.ChannelSCHHD)
	cc.SetColourCode(colourCode)

	rx := New(Options{
		SampleRateHz: 144_000,
		ClockMode:    ClockGardner,
		GardnerGain:  gardnerGain,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
	})

	dibits := buildTETRASCHHDDibits(100, colourCode, locationArea)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)

	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[i:end])
	}

	var locked bool
	var gotLA uint16
drain:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(tetra.LockState)
			if !ok {
				t.Fatalf("CCLocked payload type = %T, want tetra.LockState", ev.Payload)
			}
			locked = true
			gotLA = ls.LocationArea
		default:
			break drain
		}
	}

	if !locked {
		t.Fatalf("no cc.locked from the Gardner 8-sps path at 144 kHz; " +
			"the production TETRA rate failed to acquire (regression for issue #553)")
	}
	if gotLA != locationArea {
		t.Errorf("LockState.LocationArea = %#x, want %#x", gotLA, locationArea)
	}
}

// buildTETRASCHHDDibits assembles the dibit stream the 144 kHz lock
// test modulates: a 400-dibit 0..3 warmup so the matched filter +
// Gardner loop converge, then `repeats` bursts of (38-dibit normal
// training sequence + 108-dibit SCH/HD-encoded MLE SYSINFO + 50 idle
// dibits), then a 100-dibit flush. Mirrors buildTETRASCHHDStream in
// the cmd/gophertrunk integration test; kept local because that
// helper lives in package main.
func buildTETRASCHHDDibits(repeats int, colourCode uint32, locationArea uint16) []uint8 {
	// MLE SYSINFO payload: MCC=MNC=0; LA carries the requested value,
	// packed per AsSystemBroadcast's parse layout.
	payload := []byte{0x00, 0x00, 0x00, 0, 0}
	payload[3] = byte((locationArea >> 6) & 0xFF)
	payload[4] = byte((locationArea & 0x3F) << 2)

	pdu := tetra.PDU{
		Disc:    tetra.DiscMLE,
		Type:    uint8(tetra.MLESystemInfo),
		Payload: payload,
	}
	info := pduToType1BitsTETRA(pdu, 124)
	type5 := tetra.EncodeSCHHD(info, colourCode)
	burstDibits := tetra.TetraBitsToDibits(type5)

	frame := make([]uint8, 0, len(tetra.NormalSyncDibits())+len(burstDibits))
	frame = append(frame, tetra.NormalSyncDibits()...)
	frame = append(frame, burstDibits...)

	out := make([]uint8, 0, 400+repeats*(len(frame)+50)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// pduToType1BitsTETRA pads a PDU's assembled bytes into exactly k1
// bits, MSB-first per byte. Mirror of the in-package tetra test
// helper of the same shape.
func pduToType1BitsTETRA(pdu tetra.PDU, k1 int) []byte {
	bytes := tetra.AssemblePDU(pdu)
	if len(bytes)*8 > k1 {
		return nil
	}
	out := make([]byte, k1)
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			out[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	return out
}
