package tools

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
)

func TestNewToolsRegistered(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"randomness", "classify", "ks"} {
		if _, ok := cryptolab.Lookup(name); !ok {
			t.Fatalf("tool %q not registered", name)
		}
	}
}

func TestStatsPeriodMode(t *testing.T) {
	t.Parallel()
	pt := []byte(strings.Repeat("the quick brown fox jumps over the lazy dog ", 8))
	key := []byte("SECRET!")
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ key[i%len(key)]
	}
	in := writeFile(t, "ct.bin", ct)
	res := runMode(t, "stats", "period", []string{"-in", in, "-max-lag", "40"})
	if res.Mode != "period" || len(res.Findings) == 0 {
		t.Fatalf("expected period findings, got %+v", res)
	}
}

func TestRandomnessBatteryToolFlagsStructure(t *testing.T) {
	t.Parallel()
	// All-zero stream is maximally structured; the battery must report
	// failures and not "looks_random".
	in := writeFile(t, "zeros.bin", make([]byte, 20000))
	res := runMode(t, "randomness", "battery", []string{"-in", in})
	looksRandom := fieldBool(t, res, "looks_random")
	if looksRandom {
		t.Fatal("all-zero stream reported as random")
	}
}

func TestClassifyPlaintext(t *testing.T) {
	t.Parallel()
	in := writeFile(t, "pt.txt", []byte(strings.Repeat("the meeting is at noon by the river bridge today ", 40)))
	res := runMode(t, "classify", "auto", []string{"-in", in})
	if len(res.Findings) == 0 {
		t.Fatalf("classify returned no verdict: %+v", res)
	}
	// Top verdict should be a plaintext/substitution-family class, never
	// strong-encrypted, for clear English.
	if res.Findings[0].Label == "strong-encrypted" {
		t.Fatalf("English text misclassified as strong-encrypted: %+v", res.Findings)
	}
}

func TestKsReuseAndMTPKnownPlaintext(t *testing.T) {
	t.Parallel()
	ks := mustHex(t, "9e21557c03aa1142fe10773388291a4b5c6d7e8f")
	iv := "010203"
	pt1 := []byte("ATTACK AT DAWN OK")
	pt2 := []byte("RETREAT BY THE SE")
	ct1 := xorBytes(pt1, ks)
	ct2 := xorBytes(pt2, ks)
	lines := fmt.Sprintf("a1,%s,%s\na2,%s,%s\nb1,ffff,%s\n",
		iv, hex.EncodeToString(ct1),
		iv, hex.EncodeToString(ct2),
		hex.EncodeToString([]byte("solo")))
	in := writeFile(t, "frames.csv", []byte(lines))

	reuse := runMode(t, "ks", "reuse", []string{"-in", in})
	if fieldInt(t, reuse, "reuse_groups") != 1 {
		t.Fatalf("expected exactly one reuse group, got %+v", reuse)
	}

	ptFile := writeFile(t, "known.bin", pt1)
	mtp := runMode(t, "ks", "mtp", []string{"-in", in, "-known-label", "a1", "-known-pt", ptFile})
	// The recovered plaintext for a2 must match pt2.
	found := false
	for _, f := range mtp.Findings {
		if f.Label == "a2" {
			if got, _ := f.Detail["plaintext_ascii"].(string); got != string(pt2) {
				t.Fatalf("a2 recovered as %q, want %q", got, pt2)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("a2 not recovered: %+v", mtp.Findings)
	}
}

func TestKsExtractKeystream(t *testing.T) {
	t.Parallel()
	pt := []byte("KNOWN PLAINTEXT FRAME")
	ks := mustHex(t, "0102030405060708090a0b0c0d0e0f10111213141516171819")
	ct := xorBytes(pt, ks)
	ptFile := writeFile(t, "pt.bin", pt)
	ctFile := writeFile(t, "ct.bin", ct)
	res := runMode(t, "ks", "extract", []string{"-pt", ptFile, "-ct", ctFile})
	gotHex := fieldString(t, res, "keystream_hex")
	wantHex := hex.EncodeToString(ks[:len(pt)])
	if gotHex != wantHex {
		t.Fatalf("keystream %s, want %s", gotHex, wantHex)
	}
}

// --- small local helpers ---

func xorBytes(a, b []byte) []byte {
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

func fieldBool(t *testing.T, r *cryptolab.Result, key string) bool {
	t.Helper()
	for _, f := range r.Fields {
		if f.Key == key {
			b, ok := f.Value.(bool)
			if !ok {
				t.Fatalf("field %q is not bool: %T", key, f.Value)
			}
			return b
		}
	}
	t.Fatalf("field %q not found", key)
	return false
}

func fieldInt(t *testing.T, r *cryptolab.Result, key string) int {
	t.Helper()
	for _, f := range r.Fields {
		if f.Key == key {
			if v, ok := f.Value.(int); ok {
				return v
			}
			t.Fatalf("field %q is not int: %T", key, f.Value)
		}
	}
	t.Fatalf("field %q not found", key)
	return 0
}

func fieldString(t *testing.T, r *cryptolab.Result, key string) string {
	t.Helper()
	for _, f := range r.Fields {
		if f.Key == key {
			if v, ok := f.Value.(string); ok {
				return v
			}
			t.Fatalf("field %q is not string: %T", key, f.Value)
		}
	}
	t.Fatalf("field %q not found", key)
	return ""
}
