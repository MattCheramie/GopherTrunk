package ccdecoder

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildMotorolaRealAirBits renders a SmartNet control-channel stream
// in the real air format (frame.go codec): warmup, then repeats of
// [system-ID pair + group grant pair + idle], plus the trailing sync
// the bracket framer needs.
func buildMotorolaRealAirBits(repeats int) []byte {
	seq := []motorola.OSW{
		{Address: 0x4567, Command: motorola.CmdFirstNormal},
		{Address: 0x1F00, Command: 0x8E}, // CC broadcast: 854.5625 MHz
		{Address: 0x2E9A, Command: motorola.CmdFirstNormal},
		{Address: 0xB010, Group: true, Command: 0x1A5}, // grant: 861.5375 MHz
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

// TestMotorolaPipelineDecodesThroughProductionDDC modulates the real
// air format at a wideband 90 kHz capture rate and runs it through
// the SAME DDC target the daemon uses (18 kHz) into the production
// pipeline — the configuration the pre-#1143 synthetic tests never
// exercised (they synthesized at a bespoke 97.2 kHz with a framing no
// real system transmits, the #764/#771 self-consistent trap).
func TestMotorolaPipelineDecodesThroughProductionDDC(t *testing.T) {
	const inRate = 90_000.0

	bus := events.NewBus(1024)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	p, err := newMotorolaPipeline(PipelineOptions{
		Bus: bus, Log: slog.Default(), SystemName: "DDC",
		FrequencyHz:  854_562_500,
		SampleRateHz: motorolaDDCTargetRateHz,
		System: trunking.System{
			Name: "DDC", Protocol: trunking.ProtocolMotorola,
			ControlChannels: []uint32{854_562_500},
		},
	})
	if err != nil {
		t.Fatalf("newMotorolaPipeline: %v", err)
	}

	// 25 samples/symbol at 90 kHz, real ±1.2 kHz deviation.
	bits := buildMotorolaRealAirBits(8)
	iq := demod.ModulateGFSK(bits, 25, 4, 0.5, inRate, 1200.0)

	ddc := NewDownconverter(inRate, ddcTargetForProtocol(trunking.ProtocolMotorola))
	if got := ddc.OutRateHz(); got != motorolaDDCTargetRateHz {
		t.Fatalf("DDC out rate = %v, want %v", got, motorolaDDCTargetRateHz)
	}
	var chBuf []complex64
	const chunk = 4096
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		chBuf = ddc.Process(chBuf[:0], iq[off:end])
		p.Process(chBuf)
	}

	var lockedSys uint16
	var grantFreq uint32
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				if ls, ok := ev.Payload.(motorola.LockState); ok {
					lockedSys = ls.SystemID
				}
			case events.KindGrant:
				if g, ok := ev.Payload.(trunking.Grant); ok && g.GroupID == 0xB010 {
					grantFreq = g.FrequencyHz
				}
			}
		default:
			if lockedSys != 0x4567 {
				t.Fatalf("no lock through the production DDC path (sys=%#x)", lockedSys)
			}
			if grantFreq != 861_537_500 {
				t.Fatalf("grant freq = %d, want 861537500", grantFreq)
			}
			return
		}
	}
}
