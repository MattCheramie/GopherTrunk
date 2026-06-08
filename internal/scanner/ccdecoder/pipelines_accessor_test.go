package ccdecoder

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// discardLogger is a no-op logger the construction tests hand to factories
// that warn-log on unrecognised (here: zero-valued) per-system config.
func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// TestRegisteredProtocolsConstruct asserts every protocol the factory map
// advertises via RegisteredProtocols() can actually be constructed through
// the public NewPipeline accessor — the contract the offline siglab toolkit
// (and its protocol picker) relies on. A factory that errors on a minimal
// PipelineOptions would silently drop that protocol from the toolkit; this
// catches that at unit-test time.
func TestRegisteredProtocolsConstruct(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()

	protos := RegisteredProtocols()
	if len(protos) == 0 {
		t.Fatal("RegisteredProtocols returned no protocols")
	}

	for _, p := range protos {
		if !HasFactory(p) {
			t.Errorf("RegisteredProtocols listed %v but HasFactory is false", p)
		}
		pipe, ok, err := NewPipeline(p, PipelineOptions{
			Bus:          bus,
			Log:          discardLogger(),
			SystemName:   "test",
			FrequencyHz:  851_000_000,
			SampleRateHz: DDCTargetForProtocol(p),
		})
		if !ok {
			t.Errorf("NewPipeline(%v): ok=false for a registered protocol", p)
			continue
		}
		if err != nil {
			t.Errorf("NewPipeline(%v): %v", p, err)
			continue
		}
		if pipe == nil {
			t.Errorf("NewPipeline(%v): nil pipeline with nil error", p)
			continue
		}
		// Reset + Close must be safe no-ops on a freshly built pipeline.
		pipe.Reset()
		if cerr := pipe.Close(); cerr != nil {
			t.Errorf("Close(%v): %v", p, cerr)
		}
	}
}

// TestNewPipelineUnregistered confirms an unregistered protocol reports
// ok=false (not an error) so callers can distinguish "not supported yet"
// from "construction failed".
func TestNewPipelineUnregistered(t *testing.T) {
	pipe, ok, err := NewPipeline(trunking.ProtocolUnknown, PipelineOptions{})
	if ok {
		t.Fatal("expected ok=false for ProtocolUnknown")
	}
	if err != nil {
		t.Fatalf("expected nil error for unregistered protocol, got %v", err)
	}
	if pipe != nil {
		t.Fatal("expected nil pipeline for unregistered protocol")
	}
}

// TestDDCTargetForProtocol pins the protocol-dependent channel rate the
// siglab engine must size its down-converter to: TETRA needs the wide
// 144 kHz target, everything else the 48 kHz C4FM-family default.
func TestDDCTargetForProtocol(t *testing.T) {
	if got := DDCTargetForProtocol(trunking.ProtocolTETRA); got != tetraDDCTargetRateHz {
		t.Errorf("TETRA DDC target = %g, want %g", got, tetraDDCTargetRateHz)
	}
	for _, p := range []trunking.Protocol{
		trunking.ProtocolP25, trunking.ProtocolDMR, trunking.ProtocolP25Phase2,
		trunking.ProtocolNXDN, trunking.ProtocolYSF,
	} {
		if got := DDCTargetForProtocol(p); got != DDCTargetRateHz {
			t.Errorf("%v DDC target = %g, want %g", p, got, DDCTargetRateHz)
		}
	}
}

// TestSymbolTapForwarding confirms a wired SymbolTap observes recovered
// symbols for a representative dibit protocol (P25 P1) and a representative
// bit protocol (EDACS) — the hook the analyzer depends on for every
// protocol. It drives synthesized IQ through the real pipeline and asserts
// the tap fired with the expected symbol cardinality flag.
func TestSymbolTapForwarding(t *testing.T) {
	cases := []struct {
		name   string
		proto  trunking.Protocol
		isBits bool
	}{
		{"p25p1 dibits", trunking.ProtocolP25, false},
		{"edacs bits", trunking.ProtocolEDACS, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bus := events.NewBus(16)
			defer bus.Close()

			var taps int
			var sawBits bool
			pipe, ok, err := NewPipeline(tc.proto, PipelineOptions{
				Bus:          bus,
				Log:          discardLogger(),
				SystemName:   "test",
				SampleRateHz: DDCTargetForProtocol(tc.proto),
				SymbolTap: func(symbols []uint8, isBits bool, baseIdx int) {
					taps++
					sawBits = isBits
				},
			})
			if !ok || err != nil {
				t.Fatalf("NewPipeline(%v): ok=%v err=%v", tc.proto, ok, err)
			}
			// Feed enough zero IQ to flush several symbol periods through
			// the receiver's matched filter + clock loop. We only assert
			// the tap mechanism fires, not a lock.
			iq := make([]complex64, 48000)
			pipe.Process(iq)
			if taps == 0 {
				t.Fatalf("SymbolTap never fired for %v", tc.proto)
			}
			if sawBits != tc.isBits {
				t.Errorf("SymbolTap isBits = %v, want %v", sawBits, tc.isBits)
			}
		})
	}
}
