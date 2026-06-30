package nxdn

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	radionxdn "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
)

func TestNXDNRegistered(t *testing.T) {
	t.Parallel()
	tool, ok := cryptolab.Lookup("nxdn")
	if !ok {
		t.Fatal("nxdn tool not registered")
	}
	if _, _, err := cryptolab.LookupMode("nxdn", "brute"); err != nil {
		t.Fatalf("brute mode missing: %v", err)
	}
	if _, _, err := cryptolab.LookupMode("nxdn", "descramble"); err != nil {
		t.Fatalf("descramble mode missing: %v", err)
	}
	if len(tool.Modes()) != 2 {
		t.Fatalf("want 2 modes, got %d", len(tool.Modes()))
	}
}

// lcgBits returns n deterministic pseudo-random 0/1 bits (no math/rand, so the
// test is reproducible and -race clean).
func lcgBits(seed uint32, n int) []byte {
	out := make([]byte, n)
	x := seed
	for i := range out {
		x = x*1664525 + 1013904223
		out[i] = byte((x >> 24) & 1)
	}
	return out
}

// validCACFrame builds an 88-bit CAC message (with a valid CRC-CCITT) as a bit
// slice, using the production AssembleCAC encoder.
func validCACFrame(seed uint32) []byte {
	var payload [8]byte
	b := lcgBits(seed, 64)
	for i := 0; i < 64; i++ {
		payload[i>>3] |= b[i] << uint(7-(i&7))
	}
	msg := radionxdn.CACMessage{Type: radionxdn.RCCHVCALL, Payload: payload}
	return framing.UnpackBitsMSB(radionxdn.AssembleCAC(msg), 88)
}

// validSACCHFrame builds a 32-bit SACCH info block with a valid CRC-6.
func validSACCHFrame(seed uint32) []byte {
	return radionxdn.AppendSACCHCRC6(lcgBits(seed, 26))
}

func writeHexFrames(t *testing.T, frames [][]byte) string {
	t.Helper()
	var sb strings.Builder
	for _, f := range frames {
		sb.WriteString(hex.EncodeToString(framing.PackBitsMSB(f)))
		sb.WriteByte('\n')
	}
	p := filepath.Join(t.TempDir(), "frames.hex")
	if err := os.WriteFile(p, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func runMode(t *testing.T, mode string, args []string) *cryptolab.Result {
	t.Helper()
	_, m, err := cryptolab.LookupMode("nxdn", mode)
	if err != nil {
		t.Fatalf("LookupMode %s: %v", mode, err)
	}
	res, err := m.Run(context.Background(), args, cryptolab.Env{Stdout: &bytes.Buffer{}, Format: cryptolab.FormatText})
	if err != nil {
		t.Fatalf("%s run: %v", mode, err)
	}
	return res
}

func field(t *testing.T, r *cryptolab.Result, key string) any {
	t.Helper()
	for _, f := range r.Fields {
		if f.Key == key {
			return f.Value
		}
	}
	t.Fatalf("field %q not found in %+v", key, r.Fields)
	return nil
}

func findingsHaveKey(r *cryptolab.Result, key uint16) bool {
	for _, f := range r.Findings {
		if k, ok := f.Detail["key"].(int); ok && k == int(key) {
			return true
		}
	}
	return false
}

// scrambleAll scrambles each plaintext frame with key and returns the scrambled
// frames (mirroring the on-air TX side: per-frame keystream restart).
func scrambleAll(plain [][]byte, key uint16) [][]byte {
	out := make([][]byte, len(plain))
	for i, f := range plain {
		out[i] = radionxdn.Scramble(f, key)
	}
	return out
}

// TestBruteCACOracleRetainsKey confirms the CRC oracle keeps the true key among
// its CRC-consistent candidates. With a linear LFSR scrambler and a linear CRC,
// the survivors form a coset that always contains the true key; CAC's 16-bit CRC
// makes that coset tiny.
func TestBruteCACOracleRetainsKey(t *testing.T) {
	t.Parallel()
	const key = uint16(0x2B1F)
	plain := [][]byte{validCACFrame(1), validCACFrame(2), validCACFrame(3)}
	in := writeHexFrames(t, scrambleAll(plain, key))

	res := runMode(t, "brute", []string{"-in", in, "-oracle", "crc", "-frame-type", "cac", "-top", "40000"})

	if got := field(t, res, "candidates").(int); got < 1 {
		t.Fatalf("no candidates (%s)", res.Summary)
	}
	if !findingsHaveKey(res, key) {
		t.Fatalf("true key 0x%04x not among CAC CRC candidates: %s", key, res.Summary)
	}
}

// TestBruteSACCHOracleIsACoset documents the linear-coset property: SACCH's
// 6-bit CRC leaves a large coset (~2^9) of CRC-indistinguishable keys, which
// same-type frames cannot shrink, but the true key is always inside it.
func TestBruteSACCHOracleIsACoset(t *testing.T) {
	t.Parallel()
	const key = uint16(0x07A4)
	plain := make([][]byte, 8)
	for i := range plain {
		plain[i] = validSACCHFrame(uint32(100 + i))
	}
	in := writeHexFrames(t, scrambleAll(plain, key))

	res := runMode(t, "brute", []string{"-in", in, "-oracle", "crc", "-frame-type", "sacch", "-top", "40000"})

	got := field(t, res, "candidates").(int)
	if got <= 1 {
		t.Fatalf("expected a coset of CRC-indistinguishable keys, got %d", got)
	}
	if !findingsHaveKey(res, key) {
		t.Fatalf("true key 0x%04x not among SACCH CRC coset of %d: %s", key, got, res.Summary)
	}
}

// TestBruteKnownPlaintext confirms the known-plaintext oracle: given a snippet
// of the descrambled content, the exact-match requirement pins the key with a
// single frame (no CRC needed — this is the voice path's fallback oracle).
func TestBruteKnownPlaintext(t *testing.T) {
	t.Parallel()
	const key = uint16(0x1234)
	plain := validCACFrame(42)
	in := writeHexFrames(t, [][]byte{radionxdn.Scramble(plain, key)})
	kp := writeHexFrames(t, [][]byte{plain})

	res := runMode(t, "brute", []string{"-in", in, "-oracle", "known-pt", "-known-pt", kp})

	if got := field(t, res, "candidates").(int); got != 1 {
		t.Fatalf("want exactly 1 candidate, got %d (%s)", got, res.Summary)
	}
	if res.Findings[0].Detail["key"].(int) != int(key) {
		t.Fatalf("recovered wrong key: %+v", res.Findings)
	}
}

// TestDescrambleVerifiesCRC checks the descramble mode round-trips against the
// scrambler and validates the CRC of the recovered frame; the wrong key fails.
func TestDescrambleVerifiesCRC(t *testing.T) {
	t.Parallel()
	const key = uint16(0x0555)
	plain := validCACFrame(7)
	in := writeHexFrames(t, [][]byte{radionxdn.Scramble(plain, key)})

	ok := runMode(t, "descramble", []string{"-in", in, "-key", "1365" /* 0x0555 */, "-frame-type", "cac"})
	if field(t, ok, "crc_ok_frames").(int) != 1 {
		t.Fatalf("correct key should pass CAC CRC: %s", ok.Summary)
	}
	wantHex := hex.EncodeToString(framing.PackBitsMSB(plain))
	if ok.Findings[0].Detail["hex"].(string) != wantHex {
		t.Fatalf("descrambled payload mismatch:\n got %s\nwant %s", ok.Findings[0].Detail["hex"], wantHex)
	}

	bad := runMode(t, "descramble", []string{"-in", in, "-key", "1366", "-frame-type", "cac"})
	if field(t, bad, "crc_ok_frames").(int) != 0 {
		t.Fatalf("wrong key should fail CAC CRC: %s", bad.Summary)
	}
}
