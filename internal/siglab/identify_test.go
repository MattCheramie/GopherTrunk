package siglab

import (
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestIdentifyPicksCorrectProtocol synthesizes a clean capture for several
// strong-evidence protocols and asserts the identifier names the right one
// with non-inconclusive confidence. The set spans the modulation families
// (C4FM dibit, GFSK bit, direct-mode DMR) and the DMR Tier-distinction.
func TestIdentifyPicksCorrectProtocol(t *testing.T) {
	for _, proto := range []trunking.Protocol{
		trunking.ProtocolP25,
		trunking.ProtocolNXDN,
		trunking.ProtocolEDACS,
		trunking.ProtocolMotorola,
		trunking.ProtocolDStar,
		trunking.ProtocolDMRTier1,
		trunking.ProtocolDMRTier2,
	} {
		proto := proto
		t.Run(proto.String(), func(t *testing.T) {
			iq, meta, err := Synthesize(SynthOptions{Protocol: proto, Format: FormatF32})
			if err != nil {
				t.Fatalf("Synthesize: %v", err)
			}
			dir := t.TempDir()
			path := filepath.Join(dir, proto.String()+".cfile")
			if err := WriteCapture(path, iq, FormatF32); err != nil {
				t.Fatalf("WriteCapture: %v", err)
			}

			idr, err := Identify(path, IdentifyConfig{
				SampleRateHz: meta.SampleRateHz,
				Format:       FormatF32,
			})
			if err != nil {
				t.Fatalf("Identify: %v", err)
			}
			if idr.Winner != proto.String() {
				t.Errorf("winner = %q (conf %.2f), want %q\n  candidates: %v",
					idr.Winner, idr.Confidence, proto.String(), topN(idr, 3))
			}
			if idr.Inconclusive {
				t.Errorf("%s flagged inconclusive (conf %.2f)", proto, idr.Confidence)
			}
		})
	}
}

// TestIdentifyTETRAByCadence confirms a key-gated protocol (TETRA needs its
// colour code to lock/CRC, which identify doesn't supply) is still identified
// from its sync cadence alone.
func TestIdentifyTETRAByCadence(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolTETRA, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "tetra.cfile")
	if err := WriteCapture(path, iq, FormatF32); err != nil {
		t.Fatalf("WriteCapture: %v", err)
	}
	idr, err := Identify(path, IdentifyConfig{SampleRateHz: meta.SampleRateHz, Format: FormatF32})
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if idr.Winner != "tetra" {
		t.Errorf("winner = %q (conf %.2f), want tetra\n  candidates: %v", idr.Winner, idr.Confidence, topN(idr, 3))
	}
}

// TestIdentifyRejectsBadConfig covers the guard rails.
func TestIdentifyRejectsBadConfig(t *testing.T) {
	if _, err := Identify("/nonexistent", IdentifyConfig{SampleRateHz: 0}); err == nil {
		t.Error("expected error for zero sample rate")
	}
}

func topN(idr *IdentifyResult, n int) []string {
	out := []string{}
	for i, c := range idr.Candidates {
		if i >= n {
			break
		}
		out = append(out, c.Protocol)
	}
	return out
}
