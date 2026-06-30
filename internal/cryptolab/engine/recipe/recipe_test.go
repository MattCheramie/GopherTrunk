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

// pcm builds s16le mono PCM bytes from a ramp so the descramble ops have a
// non-trivial spectrum to invert.
func pcm(n int) []byte {
	b := make([]byte, n*2)
	for i := 0; i < n; i++ {
		v := int16((i*257 - 12000) % 20000)
		b[2*i] = byte(uint16(v))
		b[2*i+1] = byte(uint16(v) >> 8)
	}
	return b
}

// TestDescrambleSplitbandSelfInverse: split-band inversion at a fixed split is
// its own inverse, so two passes restore the input (within rounding).
func TestDescrambleSplitbandSelfInverse(t *testing.T) {
	t.Parallel()
	in := pcm(256)
	rep, err := Run(in, []Step{
		{Op: "descramble-splitband", Params: map[string]any{"split": 0.4}},
		{Op: "descramble-splitband", Params: map[string]any{"split": 0.4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.FinalBytes) != len(in) {
		t.Fatalf("length changed: got %d want %d", len(rep.FinalBytes), len(in))
	}
	a, _ := pcmToFloat(in)
	b, _ := pcmToFloat(rep.FinalBytes)
	for i := range a {
		if d := a[i] - b[i]; d > 1e-3 || d < -1e-3 {
			t.Fatalf("sample %d not restored: %.4f vs %.4f", i, a[i], b[i])
		}
	}
}

// TestDescrambleSplitbandFloatParam confirms the float param is honored whether
// it arrives as a JSON number (web) or a string (CLI/file recipe).
func TestDescrambleSplitbandFloatParam(t *testing.T) {
	t.Parallel()
	in := pcm(128)
	for _, sp := range []any{0.3, "0.3"} {
		rep, err := Run(in, []Step{{Op: "descramble-splitband", Params: map[string]any{"split": sp}}})
		if err != nil {
			t.Fatalf("split=%v: %v", sp, err)
		}
		if got := rep.Steps[0].Info["split"]; got != 0.3 {
			t.Fatalf("split=%v: info split = %v, want 0.3", sp, got)
		}
	}
}

// TestDescrambleRollingSchedule runs an explicit per-frame schedule and the
// auto detector, asserting both produce same-length PCM and surface frame info.
func TestDescrambleRollingSchedule(t *testing.T) {
	t.Parallel()
	in := pcm(4096)
	sched, err := Run(in, []Step{
		{Op: "descramble-rolling", Params: map[string]any{"frame": 1024, "schedule": "0.5,0.4"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.FinalBytes) != len(in) {
		t.Fatalf("schedule pass changed length: %d", len(sched.FinalBytes))
	}
	if sched.Steps[0].Info["schedule_steps"] != 2 {
		t.Fatalf("schedule_steps = %v, want 2", sched.Steps[0].Info["schedule_steps"])
	}

	auto, err := Run(in, []Step{
		{Op: "descramble-rolling", Params: map[string]any{"frame": 1024, "schedule": "auto"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := auto.Steps[0].Info["frames"]; !ok {
		t.Fatalf("auto mode should report a frame count, got %v", auto.Steps[0].Info)
	}
}

// TestDescrambleOddPCMErrors: the PCM ops reject a buffer that is not a whole
// number of 16-bit samples instead of corrupting the tail.
func TestDescrambleOddPCMErrors(t *testing.T) {
	t.Parallel()
	if _, err := Run([]byte{0x01, 0x02, 0x03}, []Step{{Op: "descramble-splitband"}}); err == nil {
		t.Fatal("expected an error for odd-length PCM")
	}
}
