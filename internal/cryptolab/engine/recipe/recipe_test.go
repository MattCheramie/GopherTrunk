package recipe

import (
	"bytes"
	"testing"

	rc4lib "github.com/MattCheramie/GopherTrunk/internal/crypto/rc4"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
)

func TestXORThenAnalyses(t *testing.T) {
	t.Parallel()
	pt := []byte("THE QUICK BROWN FOX OVER THE LAZY DOG AGAIN AND AGAIN")
	key := []byte{0x6b, 0x21}
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ key[i%len(key)]
	}
	steps := []Step{
		{Op: "xor", Params: map[string]any{"key": "6b21"}},
		{Op: "stats"},
		{Op: "randomness"},
	}
	rep, err := Run(ct, steps)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rep.FinalBytes, pt) {
		t.Fatalf("xor did not recover plaintext: %q", rep.FinalBytes)
	}
	if len(rep.Steps) != 3 {
		t.Fatalf("want 3 steps, got %d", len(rep.Steps))
	}
	// Analysis steps must not change the buffer length.
	if rep.Steps[1].Transform || rep.Steps[1].BytesIn != rep.Steps[1].BytesOut {
		t.Fatalf("stats should be a non-transform: %+v", rep.Steps[1])
	}
}

func TestCipherDecryptOp(t *testing.T) {
	t.Parallel()
	pt := []byte("DISPATCH TO MAIN ST NOW PLEASE OK")
	key := []byte{0x11, 0x22, 0x33, 0x44, 0x55}
	mi := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	ks, err := p25crypto.Keystream(p25crypto.AlgADP, key, mi, len(pt))
	if err != nil {
		t.Fatal(err)
	}
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ ks[i]
	}
	rep, err := Run(ct, []Step{
		{Op: "adp-decrypt", Params: map[string]any{"key": "1122334455", "mi": "010203040506070809"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rep.FinalBytes, pt) {
		t.Fatalf("adp-decrypt failed:\n got %q\nwant %q", rep.FinalBytes, pt)
	}
}

func TestRC4DecryptOp(t *testing.T) {
	t.Parallel()
	// Standard RC4 with a full caller-supplied key (the DMR Enhanced-Privacy
	// model: privacy key ‖ IV concatenated by the analyst).
	pt := []byte("DMR ENHANCED PRIVACY TRAFFIC HERE")
	fullKey := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0xAA, 0xBB} // key ‖ iv
	c, err := rc4lib.NewCipher(fullKey)
	if err != nil {
		t.Fatal(err)
	}
	ct := make([]byte, len(pt))
	c.XORKeyStream(ct, pt)
	rep, err := Run(ct, []Step{{Op: "rc4-decrypt", Params: map[string]any{"key": "11223344 55aabb"}}})
	if err != nil {
		t.Fatal(err)
	}
	if string(rep.FinalBytes) != string(pt) {
		t.Fatalf("rc4-decrypt failed: %q", rep.FinalBytes)
	}
}

func TestExternDecryptMarkedExternal(t *testing.T) {
	t.Parallel()
	if !External("extern-decrypt") {
		t.Fatal("extern-decrypt must be flagged external (so the web endpoint refuses it)")
	}
	if External("xor") {
		t.Fatal("xor is not external")
	}
	var found bool
	for _, s := range Specs() {
		if s.Name == "extern-decrypt" {
			found = true
			if !s.External {
				t.Fatal("Specs() should mark extern-decrypt External")
			}
		}
	}
	if !found {
		t.Fatal("extern-decrypt missing from Specs()")
	}
}

func TestSpecsExposeParams(t *testing.T) {
	t.Parallel()
	var sawXORKey, sawCipherMI bool
	for _, s := range Specs() {
		if s.Name == "xor" {
			for _, p := range s.Params {
				if p.Name == "key" && p.Kind == "hex" {
					sawXORKey = true
				}
			}
		}
		if s.Name == "adp-decrypt" {
			for _, p := range s.Params {
				if p.Name == "mi" {
					sawCipherMI = true
				}
			}
		}
	}
	if !sawXORKey || !sawCipherMI {
		t.Fatal("Specs() should expose per-op params (xor.key, adp-decrypt.mi)")
	}
}

func TestNotIsSelfInverse(t *testing.T) {
	t.Parallel()
	in := []byte{0x00, 0xFF, 0xA5, 0x5A}
	rep, err := Run(in, []Step{{Op: "not"}, {Op: "not"}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rep.FinalBytes, in) {
		t.Fatalf("double NOT != identity: %x", rep.FinalBytes)
	}
}

func TestHexDecodeThenXOR(t *testing.T) {
	t.Parallel()
	// "48490a" -> "HI\n"; XOR key 0x00 leaves it unchanged.
	rep, err := Run([]byte("48 49 0a"), []Step{
		{Op: "hex-decode"},
		{Op: "xor", Params: map[string]any{"key": "00"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rep.FinalBytes, []byte{0x48, 0x49, 0x0a}) {
		t.Fatalf("hex-decode chain wrong: %x", rep.FinalBytes)
	}
}

func TestUnknownOpErrors(t *testing.T) {
	t.Parallel()
	if _, err := Run([]byte("x"), []Step{{Op: "nope"}}); err == nil {
		t.Fatal("expected an error for an unknown op")
	}
}

func TestOpsListed(t *testing.T) {
	t.Parallel()
	ops := Ops()
	if len(ops) < 8 {
		t.Fatalf("expected the op registry to be populated, got %d", len(ops))
	}
	var sawTransform, sawAnalysis bool
	for _, o := range ops {
		if o.Transform {
			sawTransform = true
		} else {
			sawAnalysis = true
		}
	}
	if !sawTransform || !sawAnalysis {
		t.Fatal("expected both transform and analysis ops")
	}
}
