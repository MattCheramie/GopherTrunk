package assess

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/crypto/rc4"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/keystream"
)

func mockSpec() string { return os.Args[0] + " -test.run=TestHelperProcess --" }

// TestHelperProcess is a mock external cipher (RC4 over key‖iv standing in for
// an unbundled cipher such as TEA1). A no-op unless invoked with a "--" marker.
func TestHelperProcess(t *testing.T) {
	args := os.Args
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep < 0 || sep+1 >= len(args) {
		return
	}
	rest := args[sep+1:]
	ksOf := func(key, iv []byte, n int) []byte {
		c, _ := rc4.NewCipher(append(append([]byte{}, key...), iv...))
		return c.KeyStream(n)
	}
	switch rest[0] {
	case "keystream":
		key, _ := hex.DecodeString(rest[1])
		iv, _ := hex.DecodeString(rest[2])
		n, _ := strconv.Atoi(rest[3])
		fmt.Println(hex.EncodeToString(ksOf(key, iv, n)))
	case "brute":
		iv, _ := hex.DecodeString(rest[1])
		known, _ := hex.DecodeString(rest[2])
		bits, _ := strconv.Atoi(rest[3])
		base, _ := hex.DecodeString(rest[4])
		found := ""
		for v := 0; v < (1 << uint(bits)); v++ {
			key := append([]byte(nil), base...)
			for b := 0; b*8 < bits; b++ {
				key[len(key)-1-b] = byte(v >> uint(8*b))
			}
			if bytes.Equal(ksOf(key, iv, len(known)), known) {
				found = hex.EncodeToString(key)
				break
			}
		}
		fmt.Println(found)
	}
	os.Exit(0)
}

// TestAssessExternalBruteBreaksTEA1Stand_in: an "unbundled" cipher (algid 0x01,
// here mocked by RC4) is broken via the external-cipher brute path, exactly the
// shape of the TEA1 32-bit backdoor attack.
func TestAssessExternalBruteBreaks(t *testing.T) {
	t.Parallel()
	const alg = 0x01 // pretend TEA1 (tetra)
	key := []byte{0, 0, 0x0A, 0xBC}
	iv := []byte{0x11, 0x22}
	pt := []byte("EMERGENCY DISPATCH ALL UNITS NOW")
	c, _ := rc4.NewCipher(append(append([]byte{}, key...), iv...))
	ct := xorb(pt, c.KeyStream(len(pt)))
	f := keystream.Frame{Label: "t1", IV: iv, CT: ct, AlgID: alg}

	rep := Run(Input{
		Frames:      []keystream.Frame{f},
		Protocol:    "tetra",
		KnownLabel:  "t1",
		KnownPT:     pt,
		BruteBits:   13,
		BaseKey:     make([]byte, len(key)),
		ExternCmd:   mockSpec(),
		ExternAlgID: alg,
	})
	if rep.Verdict != VerdictBroken {
		t.Fatalf("external brute should break the cipher, verdict = %s; methods: %+v", rep.Verdict, rep.Methods)
	}
	kb := find(rep, "key-brute")
	if kb == nil || !kb.Verified {
		t.Fatalf("key-brute should be verified via the external cipher, got %+v", kb)
	}
}
