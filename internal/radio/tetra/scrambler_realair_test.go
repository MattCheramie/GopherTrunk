package tetra

import (
	"encoding/base64"
	"testing"
)

// realBNCHBlock is a single SCH/HD type-5 block (108 dibits, one dibit per
// byte, MSB-first bit convention) recovered off-air from the BNCH of a live
// TETRA TMO downlink — the MCCH of a cell with MCC 250 / MNC 13, extended
// colour code 262144845 (base colour 13). It was captured through the
// production IQ → π/4-DQPSK receiver, so it carries the exact scrambling a
// real base station applied.
//
// A real base station seeds the §8.2.5 scrambler with the extended colour
// code bit-reversed into the LFSR (e(1) = the code's MSB). The pre-fix
// GopherTrunk scrambler seeded the raw value instead; that is a no-op for
// the BSCH (extended colour 0) so cold-start lock still worked, but it left
// every non-zero-colour burst — BNCH SYSINFO and all SCH signalling —
// uncorrectable, which is why a locked TETRA control channel decoded no
// messages. This block only descrambles CRC-clean under the corrected seed,
// so it is the on-air regression guard for that fix.
const realBNCHBlock = "AwABAAABAAABAAIBAgICAgMDAgEAAAMDAwMDAAEAAwEDAQMDAAMBAwMCAwMDAQEDAQMDAgABAQMDAgABAwIAAQECAAEBAQADAAEDAQACAgECAAACAwEDAgMAAQEAAwMDAQAAAgAAAQEBAQEA"

const realBNCHColour = 262144845

// TestDescrambleRealBNCHBlock pins the §8.2.5 scrambler seed to real air:
// the captured BNCH block descrambles + FEC-decodes CRC-clean only when the
// extended colour code is bit-reversed into the LFSR. It fails with the
// pre-fix (raw-value) seed.
func TestDescrambleRealBNCHBlock(t *testing.T) {
	dibits, err := base64.StdEncoding.DecodeString(realBNCHBlock)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if len(dibits) != 108 {
		t.Fatalf("fixture has %d dibits, want 108 (one SCH/HD block)", len(dibits))
	}
	if _, ok := DecodeSCHHD(TetraDibitsToBits(dibits), realBNCHColour); !ok {
		t.Fatal("real BNCH block failed to descramble/decode CRC-clean — " +
			"the §8.2.5 scrambler seed must bit-reverse the extended colour code")
	}
}
