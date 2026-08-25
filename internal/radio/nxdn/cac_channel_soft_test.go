package nxdn

import (
	"math/rand"
	"testing"
)

// llrFromBit maps a hard channel bit to its ideal LLR (framing convention:
// LLR > 0 ⇒ bit 0).
func llrFromBit(b byte) float32 {
	if b&1 == 1 {
		return -1
	}
	return 1
}

// randomCACInfo returns a deterministic pseudo-random 155-bit info block.
func randomCACInfo(rng *rand.Rand) []byte {
	info := make([]byte, CACInfoBits)
	for i := range info {
		info[i] = byte(rng.Intn(2))
	}
	return info
}

// TestDecodeCACChannelSoftCleanRoundTrip anchors the soft chain against the
// hard encoder: ideal LLRs decode to exactly the encoded info block with a
// clean CRC.
func TestDecodeCACChannelSoftCleanRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 50; trial++ {
		info := randomCACInfo(rng)
		channel := EncodeCACChannel(info)
		llr := make([]float32, len(channel))
		for i, b := range channel {
			llr[i] = llrFromBit(b)
		}
		got, ok := DecodeCACChannelSoft(llr)
		if !ok {
			t.Fatalf("trial %d: CRC failed on a clean soft round trip", trial)
		}
		for i := range info {
			if got[i] != info[i] {
				t.Fatalf("trial %d: info bit %d differs", trial, i)
			}
		}
	}
}

// TestDecodeCACChannelSoftBeatsHardOnNoisyChannel is the failing-first
// yield claim behind nxdn_soft_decision: over an AWGN channel at a level
// where the hard-sliced DecodeCACChannel loses most CAC bursts to CRC
// failure, the soft decode recovers substantially more. CRC yield is the
// metric (the CLAUDE.md lesson: never judge a decode lever by anything
// softer).
func TestDecodeCACChannelSoftBeatsHardOnNoisyChannel(t *testing.T) {
	const sigma = 0.7
	rng := rand.New(rand.NewSource(9))
	trials, hardOK, softOK := 200, 0, 0
	for trial := 0; trial < trials; trial++ {
		info := randomCACInfo(rng)
		channel := EncodeCACChannel(info)
		llr := make([]float32, len(channel))
		hard := make([]byte, len(channel))
		for i, b := range channel {
			v := float64(llrFromBit(b)) + rng.NormFloat64()*sigma
			llr[i] = float32(v)
			if v < 0 {
				hard[i] = 1
			}
		}
		if _, ok := DecodeCACChannel(hard); ok {
			hardOK++
		}
		if _, ok := DecodeCACChannelSoft(llr); ok {
			softOK++
		}
	}
	t.Logf("sigma=%.2f: hard CRC ok %d/%d, soft CRC ok %d/%d", sigma, hardOK, trials, softOK, trials)
	if softOK <= hardOK {
		t.Errorf("soft CRC yield %d not better than hard %d — the soft chain buys nothing", softOK, hardOK)
	}
	if softOK < hardOK+trials/10 {
		t.Errorf("soft CRC yield %d vs hard %d: expected a substantial (>10%% of trials) margin at this SNR", softOK, hardOK)
	}
}
