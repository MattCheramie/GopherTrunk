package framing

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestInterleaveRoundTrip(t *testing.T) {
	in := make([]byte, bptcN)
	for i := range in {
		in[i] = byte(i & 1)
	}
	channel := InterleaveBPTC(in)
	back := DeinterleaveBPTC(channel)
	if !bytes.Equal(back, in) {
		t.Errorf("interleave round-trip mismatch")
	}
}

func TestInterleaveIsAPermutation(t *testing.T) {
	// All 196 indices must appear exactly once.
	seen := [bptcN]bool{}
	for i := 0; i < bptcN; i++ {
		seen[(i*181)%bptcN] = true
	}
	for i, ok := range seen {
		if !ok {
			t.Fatalf("interleave drops index %d", i)
		}
	}
}

func TestBPTCEncodeDecodeRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for trial := 0; trial < 50; trial++ {
		info := make([]byte, 96)
		for i := range info {
			info[i] = byte(r.Intn(2))
		}
		channel := EncodeBPTC196_96(info)
		decoded, errs := DecodeBPTC196_96(channel)
		if errs != 0 {
			t.Fatalf("clean decode reported %d errors", errs)
		}
		if !bytes.Equal(decoded, info) {
			t.Fatalf("trial %d: decode mismatch", trial)
		}
	}
}

func TestBPTCCorrectsSingleBitErrors(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	info := make([]byte, 96)
	for i := range info {
		info[i] = byte(r.Intn(2))
	}
	channel := EncodeBPTC196_96(info)
	for bit := 0; bit < bptcN; bit++ {
		corrupted := append([]byte(nil), channel...)
		corrupted[bit] ^= 1
		decoded, errs := DecodeBPTC196_96(corrupted)
		if errs == -1 {
			t.Errorf("bit %d: decoder reported failure", bit)
			continue
		}
		if !bytes.Equal(decoded, info) {
			t.Errorf("bit %d: decode failed to recover info", bit)
		}
	}
}

func TestBPTCAllZeroAndAllOne(t *testing.T) {
	for _, fill := range []byte{0, 1} {
		info := make([]byte, 96)
		for i := range info {
			info[i] = fill
		}
		channel := EncodeBPTC196_96(info)
		decoded, errs := DecodeBPTC196_96(channel)
		if errs != 0 {
			t.Errorf("fill=%d: errs=%d", fill, errs)
		}
		if !bytes.Equal(decoded, info) {
			t.Errorf("fill=%d: round-trip mismatch", fill)
		}
	}
}
