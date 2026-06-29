package tools

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
)

func TestAssessToolRegistered(t *testing.T) {
	t.Parallel()
	if _, ok := cryptolab.Lookup("assess"); !ok {
		t.Fatal("assess tool not registered")
	}
}

// TestAssessCryptoBrokenOnDefaultKey drives the full tool: frames encrypted
// with the all-zero ADP default key plus a known plaintext must yield a BROKEN
// verdict and a verified weak-key finding.
func TestAssessCryptoBrokenOnDefaultKey(t *testing.T) {
	t.Parallel()
	key := make([]byte, p25crypto.KeySize(p25crypto.AlgADP)) // default all-zero key
	enc := func(mi, pt []byte) string {
		ks, err := p25crypto.Keystream(p25crypto.AlgADP, key, mi, len(pt))
		if err != nil {
			t.Fatal(err)
		}
		ct := make([]byte, len(pt))
		for i := range pt {
			ct[i] = pt[i] ^ ks[i]
		}
		return hex.EncodeToString(ct)
	}
	mi1 := "010101010101010101"
	mi2 := "020202020202020202"
	pt1 := []byte("DISPATCH MAIN ST NOW")
	lines := fmt.Sprintf(
		"{\"label\":\"call-1\",\"iv\":\"%s\",\"ct\":\"%s\",\"algid\":170}\n"+
			"{\"label\":\"call-2\",\"iv\":\"%s\",\"ct\":\"%s\",\"algid\":170}\n",
		mi1, enc(mustHex(t, mi1), pt1),
		mi2, enc(mustHex(t, mi2), []byte("UNITS HOLD POSITION!")))
	in := writeFile(t, "frames.jsonl", []byte(lines))
	ptFile := writeFile(t, "known.bin", pt1)

	res := runMode(t, "assess", "crypto", []string{"-in", in, "-known-label", "call-1", "-known-pt", ptFile})
	if got := fieldString(t, res, "verdict"); got != "BROKEN" {
		t.Fatalf("verdict = %q, want BROKEN; result: %+v", got, res)
	}
	// The weak-key finding must be present and verified.
	var found bool
	for _, f := range res.Findings {
		if f.Label == "weak-key" {
			if v, _ := f.Detail["verified"].(bool); v {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("weak-key method not verified: %+v", res.Findings)
	}
}

// TestAssessCryptoTEA1Advisory drives the tool with -protocol tetra and a TEA1
// algid: the known-weakness advisory must fire and the verdict must not be
// RESISTANT (the backdoor floor), even with no key.
func TestAssessCryptoTEA1Advisory(t *testing.T) {
	t.Parallel()
	lines := "tetra-1,1234,deadbeefcafe\ntetra-2,5678,0011223344556677\n"
	// CSV omits algid, so use JSONL to carry it.
	lines = "{\"label\":\"t1\",\"iv\":\"1234\",\"ct\":\"deadbeefcafe0011\",\"algid\":1}\n" +
		"{\"label\":\"t2\",\"iv\":\"5678\",\"ct\":\"00112233445566ff\",\"algid\":1}\n"
	in := writeFile(t, "tea1.jsonl", []byte(lines))
	res := runMode(t, "assess", "crypto", []string{"-in", in, "-protocol", "tetra"})
	if got := fieldString(t, res, "verdict"); got == "RESISTANT" {
		t.Fatalf("TEA1 should not read RESISTANT, got %q", got)
	}
	var advisory bool
	for _, f := range res.Findings {
		if f.Label == "known-weakness" {
			advisory = true
		}
	}
	if !advisory {
		t.Fatalf("known-weakness finding missing: %+v", res.Findings)
	}
}
