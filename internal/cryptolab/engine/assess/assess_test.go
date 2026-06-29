package assess

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/keystream"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
)

func xorb(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func find(r Report, name string) *MethodResult {
	for i := range r.Methods {
		if r.Methods[i].Name == name {
			return &r.Methods[i]
		}
	}
	return nil
}

// TestAssessWeakKeyCompleteBreak: frames encrypted with a DEFAULT (all-zero)
// ADP key, with one frame's plaintext known. The weak-key method must verify
// the key and the verdict must be BROKEN (the encryption failed the test).
func TestAssessWeakKeyCompleteBreak(t *testing.T) {
	t.Parallel()
	key := make([]byte, p25crypto.KeySize(p25crypto.AlgADP)) // all-zero = a default key
	mk := func(label string, mi, pt []byte) (keystream.Frame, []byte) {
		ks, err := p25crypto.Keystream(p25crypto.AlgADP, key, mi, len(pt))
		if err != nil {
			t.Fatal(err)
		}
		return keystream.Frame{Label: label, IV: mi, CT: xorb(pt, ks), AlgID: p25crypto.AlgADP}, pt
	}
	f1, pt1 := mk("call-1", []byte{1, 1, 1, 1, 1, 1, 1, 1, 1}, []byte("DISPATCH TO MAIN AND 3RD NOW"))
	f2, _ := mk("call-2", []byte{2, 2, 2, 2, 2, 2, 2, 2, 2}, []byte("UNITS HOLD POSITION PLEASE!!"))

	rep := Run(Input{Frames: []keystream.Frame{f1, f2}, KnownLabel: "call-1", KnownPT: pt1})

	wk := find(rep, "weak-key")
	if wk == nil || !wk.Applicable || !wk.Verified {
		t.Fatalf("weak-key method should verify the default key, got %+v", wk)
	}
	if rep.Verdict != VerdictBroken {
		t.Fatalf("verdict = %s, want BROKEN; methods: %+v", rep.Verdict, rep.Methods)
	}
}

// TestAssessIVReuseDetected: distinct frames that reuse an MI must register on
// the iv-reuse method and yield at least a PARTIAL verdict.
func TestAssessIVReuseDetected(t *testing.T) {
	t.Parallel()
	ks := []byte{0x9e, 0x21, 0x55, 0x7c, 0x03, 0xaa, 0x11, 0x42, 0xfe, 0x10}
	mi := []byte{7, 7, 7}
	f1 := keystream.Frame{Label: "a", IV: mi, CT: xorb([]byte("HELLO WORLD"), ks)}
	f2 := keystream.Frame{Label: "b", IV: mi, CT: xorb([]byte("GOODBYE ALL"), ks)}
	rep := Run(Input{Frames: []keystream.Frame{f1, f2}})
	reuse := find(rep, "iv-reuse")
	if reuse == nil || !reuse.Applicable || reuse.Effectiveness == 0 {
		t.Fatalf("iv-reuse should fire on a shared MI, got %+v", reuse)
	}
	if rep.Verdict == VerdictResistant {
		t.Fatalf("verdict should not be RESISTANT when an IV is reused: %+v", rep)
	}
}

// TestAssessResistant: distinct IVs, an unsupported algorithm, random-looking
// ciphertext, and no known plaintext — nothing should recover, verdict
// RESISTANT.
func TestAssessResistant(t *testing.T) {
	t.Parallel()
	// Pseudo-random ciphertext via an LCG so the cipher-strength test passes.
	gen := func(seed uint32, n int) []byte {
		out := make([]byte, n)
		x := seed
		for i := range out {
			x = x*1664525 + 1013904223
			out[i] = byte(x >> 24)
		}
		return out
	}
	frames := []keystream.Frame{
		{Label: "a", IV: []byte{1, 0, 0}, CT: gen(1, 4000), AlgID: 0x00},
		{Label: "b", IV: []byte{2, 0, 0}, CT: gen(2, 4000), AlgID: 0x00},
	}
	rep := Run(Input{Frames: frames})
	if rep.Verdict != VerdictResistant {
		t.Fatalf("verdict = %s, want RESISTANT; methods: %+v", rep.Verdict, rep.Methods)
	}
}

// TestAssessTEA1KnownWeakness: a TETRA TEA1 capture (algid 0x01) must trigger
// the known-weakness advisory and lift the verdict off RESISTANT even though
// the TEA1 cipher is not bundled (nothing is actually decrypted).
func TestAssessTEA1KnownWeakness(t *testing.T) {
	t.Parallel()
	// Random-looking ciphertext, distinct IVs, no key — only the advisory fires.
	gen := func(seed uint32, n int) []byte {
		out := make([]byte, n)
		x := seed
		for i := range out {
			x = x*1664525 + 1013904223
			out[i] = byte(x >> 24)
		}
		return out
	}
	frames := []keystream.Frame{
		{Label: "a", IV: []byte{1, 0}, CT: gen(1, 4000), AlgID: 0x01},
		{Label: "b", IV: []byte{2, 0}, CT: gen(2, 4000), AlgID: 0x01},
	}
	rep := Run(Input{Frames: frames, Protocol: "tetra"})
	kw := find(rep, "known-weakness")
	if kw == nil || !kw.Applicable {
		t.Fatalf("known-weakness should fire for TEA1, got %+v", kw)
	}
	if rep.Verdict == VerdictResistant {
		t.Fatalf("TEA1 (broken by design) should not read RESISTANT: %+v", rep)
	}
}

// TestAssessKeyBruteRecoversSmallKey: a P25 ADP key whose only non-zero bits
// are in the low 12 bits must be recovered by the reduced-keyspace brute, and
// the verdict must be BROKEN.
func TestAssessKeyBruteRecoversSmallKey(t *testing.T) {
	t.Parallel()
	key := make([]byte, p25crypto.KeySize(p25crypto.AlgADP)) // 5 bytes, zero
	key[4] = 0xBC
	key[3] = 0x0A // low 12 bits = 0xABC
	mk := func(label string, mi, pt []byte) (keystream.Frame, []byte) {
		ks, err := p25crypto.Keystream(p25crypto.AlgADP, key, mi, len(pt))
		if err != nil {
			t.Fatal(err)
		}
		return keystream.Frame{Label: label, IV: mi, CT: xorb(pt, ks), AlgID: p25crypto.AlgADP}, pt
	}
	f1, pt1 := mk("call-1", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}, []byte("DISPATCH NORTH NOW"))
	rep := Run(Input{
		Frames: []keystream.Frame{f1}, Protocol: "p25",
		KnownLabel: "call-1", KnownPT: pt1, BruteBits: 13,
	})
	kb := find(rep, "key-brute")
	if kb == nil || !kb.Applicable || !kb.Verified {
		t.Fatalf("key-brute should recover a 12-bit key, got %+v", kb)
	}
	if rep.Verdict != VerdictBroken {
		t.Fatalf("verdict = %s, want BROKEN; methods: %+v", rep.Verdict, rep.Methods)
	}
}

// TestAssessTDESRoundsTripThroughWeakKey: a Triple-DES frame under the all-zero
// default key must be broken by the weak-key dictionary (exercises the new
// 3DES construction end-to-end through the harness).
func TestAssessTDESWeakKey(t *testing.T) {
	t.Parallel()
	key := make([]byte, p25crypto.KeySize(p25crypto.AlgTDES)) // 24-byte zero key
	pt := []byte("CONFIDENTIAL TRAFFIC HERE")
	ks, err := p25crypto.Keystream(p25crypto.AlgTDES, key, []byte{1, 2, 3, 4, 5, 6, 7, 8}, len(pt))
	if err != nil {
		t.Fatal(err)
	}
	f := keystream.Frame{Label: "c1", IV: []byte{1, 2, 3, 4, 5, 6, 7, 8}, CT: xorb(pt, ks), AlgID: p25crypto.AlgTDES}
	rep := Run(Input{Frames: []keystream.Frame{f}, KnownLabel: "c1", KnownPT: pt})
	if rep.Verdict != VerdictBroken {
		t.Fatalf("all-zero-key TDES should break, verdict = %s", rep.Verdict)
	}
}

// TestAssessKnownPlaintextRecovers: a reuse group with one known frame must be
// fully decrypted by the known-plaintext method.
func TestAssessKnownPlaintextRecovers(t *testing.T) {
	t.Parallel()
	ks := []byte{0x9e, 0x21, 0x55, 0x7c, 0x03, 0xaa, 0x11, 0x42, 0xfe, 0x10, 0x77, 0x33}
	mi := []byte{5, 5, 5}
	pt1 := []byte("KNOWN FRAME!")
	pt2 := []byte("SECRET FRAME")
	f1 := keystream.Frame{Label: "k", IV: mi, CT: xorb(pt1, ks)}
	f2 := keystream.Frame{Label: "s", IV: mi, CT: xorb(pt2, ks)}
	rep := Run(Input{Frames: []keystream.Frame{f1, f2}, KnownLabel: "k", KnownPT: pt1})
	kp := find(rep, "known-plaintext")
	if kp == nil || !kp.Verified || kp.Effectiveness == 0 {
		t.Fatalf("known-plaintext should recover the group, got %+v", kp)
	}
}
