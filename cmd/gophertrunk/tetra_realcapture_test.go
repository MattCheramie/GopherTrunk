package main

import (
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// testdata/tetra-scbs-cc.cs16 is a 1.5 s, 50 kHz headerless cs16 slice of a
// real TETRA Single Carrier Base Station control channel at 467.913 MHz —
// the exact SCBS / dynamic-MCCH-sharing carrier from issue #925, captured by
// the reporter (MCC 250, MNC 013, colour code 0x2C, confirmed live by a
// commercial receiver and SDR++). It is fed through the production siglab
// engine — the same path `gophertrunk replay` uses — which up-converts the
// 50 kHz slice to the 144 kHz TETRA channel rate and runs the full receiver
// + control-channel chain.
//
// Why this exists (issue #925): the reporter's symptom was "18 000-baud
// symbol timing recovers but the control channel never locks." The whole
// TETRA coding suite round-trips its own encoder, so a scrambler that
// diverged from ETSI EN 300 392-2 §8.2.5 decoded synthesized bursts
// perfectly yet could not descramble a single real BSCH — the sync training
// sequence (unscrambled) correlated, but the BSCH FEC never reached a
// CRC-clean codeword, so the colour code was never learned and no lock was
// declared. This capture is genuine over-the-air bits, so it is the guard
// that the scrambler (and the BSCH/BNCH descramble→deinterleave→RCPC→CRC
// chain behind it) actually conforms to the standard, not just to itself.
//
// Before the §8.2.5 scrambler fix this capture never locks; after it, it
// locks in ~11 ms and reads the cell identity below.
func TestReplayTETRASCBSLocksRealCapture(t *testing.T) {
	const (
		sampleRateHz = 50000.0
		wantMCC      = 250
		wantMNC      = 13
	)

	cfg := siglab.Config{
		Protocol:     trunking.ProtocolTETRA,
		SystemName:   "tetra-scbs-test",
		SampleRateHz: sampleRateHz,
		Format:       siglab.FormatS16,
	}
	res, err := siglab.Run(filepath.Join("testdata", "tetra-scbs-cc.cs16"), cfg)
	if err != nil {
		t.Fatalf("siglab.Run: %v", err)
	}
	if !res.Locked || res.Lock == nil {
		t.Fatalf("TETRA SCBS control channel failed to lock on a real capture that a "+
			"commercial receiver decodes — the sync burst correlates but the BSCH never "+
			"descrambles (regression for issue #925); locked=%v", res.Locked)
	}
	if got := fieldInt(res.Lock.Fields["MCC"]); got != wantMCC {
		t.Errorf("locked MCC = %d, want %d (wrong descramble/decode of the BSCH SYNC PDU)", got, wantMCC)
	}
	if got := fieldInt(res.Lock.Fields["MNC"]); got != wantMNC {
		t.Errorf("locked MNC = %d, want %d", got, wantMNC)
	}
}

// fieldInt coerces a flattened LockInfo field (JSON-ish any: float64, or any
// integer kind) to int for comparison.
func fieldInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case uint8:
		return int(n)
	default:
		return -1
	}
}
