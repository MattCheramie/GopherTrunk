//go:build integration

package phase2_test

// Validates the MAC structure walk against every distinct MAC PDU SDRtrunk
// decoded from the real Phase 2 capture set. SDRtrunk emits one message per
// structure it finds, so the oracle is the opcode *sequence* of each PDU.
//
//	GT_P2_ORACLE=/path/oracle.json go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run MACStructureOracle -v

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

type oraclePDU struct {
	Hex  string `json:"hex"`
	Bits int    `json:"bits"`
	Ops  []int  `json:"ops"`
}

func TestMACStructureOracle(t *testing.T) {
	path := os.Getenv("GT_P2_ORACLE")
	if path == "" {
		t.Skip("set GT_P2_ORACLE")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var pdus []oraclePDU
	if err := json.Unmarshal(raw, &pdus); err != nil {
		t.Fatal(err)
	}
	t.Logf("oracle: %d distinct MAC PDUs", len(pdus))

	var exact, prefix, wrong, truncated int
	byType := map[string]int{}
	for _, p := range pdus {
		b, err := hex.DecodeString(p.Hex)
		if err != nil {
			t.Fatalf("bad hex %q", p.Hex)
		}
		msg := framing.UnpackBitsMSB(b, p.Bits)
		got, err := p25p2.ParseACCHMessage(msg)
		if err != nil {
			t.Errorf("%s: %v", p.Hex, err)
			continue
		}
		byType[got.Type.String()]++
		if got.Truncated {
			truncated++
		}
		// Every structure we emit must be one SDRtrunk found, in order.
		bad := false
		for i, s := range got.Structures {
			if i >= len(p.Ops) || int(s.Opcode) != p.Ops[i] {
				bad = true
				break
			}
		}
		switch {
		case bad:
			wrong++
			if wrong <= 5 {
				var ours []int
				for _, s := range got.Structures {
					ours = append(ours, int(s.Opcode))
				}
				t.Errorf("%s\n  walked %v\n  sdrtrunk %v", p.Hex, ours, p.Ops)
			}
		case len(got.Structures) == len(p.Ops):
			exact++
		default:
			prefix++
		}
	}
	t.Logf("exact=%d prefix=%d wrong=%d truncated=%d", exact, prefix, wrong, truncated)
	t.Logf("PDU types: %v", byType)
	if wrong > 0 {
		t.Errorf("%d PDUs walked to an opcode SDRtrunk did not report", wrong)
	}
}
