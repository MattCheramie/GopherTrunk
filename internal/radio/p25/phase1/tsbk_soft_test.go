package phase1

import (
	"math/rand"
	"testing"
)

// tsbkChannelToLLRs maps the 98 hard channel dibits to ideal per-bit LLR
// pairs (framing convention: LLR > 0 ⇒ bit 0; MSB then LSB per dibit).
func tsbkChannelToLLRs(channel []uint8) []float32 {
	llr := make([]float32, 2*len(channel))
	for i, d := range channel {
		llr[2*i], llr[2*i+1] = 1, 1
		if (d>>1)&1 == 1 {
			llr[2*i] = -1
		}
		if d&1 == 1 {
			llr[2*i+1] = -1
		}
	}
	return llr
}

// tsbkTestInfo builds a valid 12-byte TSBK info block (CRC appended) from
// pseudo-random opcode/payload content.
func tsbkTestInfo(rng *rand.Rand) []byte {
	t := TSBK{
		LB:     true,
		Opcode: Opcode(rng.Intn(0x40)),
		MFID:   0,
	}
	for i := range t.Payload {
		t.Payload[i] = byte(rng.Intn(256))
	}
	return AssembleTSBK(t)
}

// TestDecodeTSBKChannelSoftCleanRoundTrip anchors the soft TSBK pipeline
// (LLR deinterleave → scalar soft trellis → repack → ParseTSBK) against the
// hard encoder: ideal LLRs decode to the same TSBK with a clean CRC —
// proving the phase1 encoder and the shared framing soft trellis agree on
// tables and bit order.
func TestDecodeTSBKChannelSoftCleanRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 100; trial++ {
		info := tsbkTestInfo(rng)
		channel := EncodeTSBKChannel(info)
		want, _, err := DecodeTSBKChannel(channel)
		if err != nil {
			t.Fatalf("trial %d: hard round trip failed: %v", trial, err)
		}
		got, _, err := DecodeTSBKChannelSoft(tsbkChannelToLLRs(channel))
		if err != nil {
			t.Fatalf("trial %d: soft round trip failed: %v", trial, err)
		}
		if got.Opcode != want.Opcode || got.Payload != want.Payload || got.LB != want.LB {
			t.Fatalf("trial %d: soft decode differs from hard on a clean channel", trial)
		}
	}
}

// TestDecodeTSBKChannelSoftBeatsHardOnNoisyChannel is the failing-first
// yield claim behind p25_phase1_soft_decision: over an AWGN per-bit
// channel at a level where the hard path loses most TSBKs to trellis/CRC
// failure, the soft path recovers substantially more. CRC-verified TSBK
// yield is the metric.
func TestDecodeTSBKChannelSoftBeatsHardOnNoisyChannel(t *testing.T) {
	const sigma = 0.75
	rng := rand.New(rand.NewSource(9))
	trials, hardOK, softOK := 300, 0, 0
	for trial := 0; trial < trials; trial++ {
		info := tsbkTestInfo(rng)
		channel := EncodeTSBKChannel(info)
		llr := make([]float32, 2*len(channel))
		hard := make([]uint8, len(channel))
		for i, d := range channel {
			m, l := float64(1), float64(1)
			if (d>>1)&1 == 1 {
				m = -1
			}
			if d&1 == 1 {
				l = -1
			}
			vm := m + rng.NormFloat64()*sigma
			vl := l + rng.NormFloat64()*sigma
			llr[2*i] = float32(vm)
			llr[2*i+1] = float32(vl)
			var msb, lsb uint8
			if vm < 0 {
				msb = 1
			}
			if vl < 0 {
				lsb = 1
			}
			hard[i] = msb<<1 | lsb
		}
		if _, _, err := DecodeTSBKChannel(hard); err == nil {
			hardOK++
		}
		if _, _, err := DecodeTSBKChannelSoft(llr); err == nil {
			softOK++
		}
	}
	t.Logf("sigma=%.2f: hard TSBK ok %d/%d, soft TSBK ok %d/%d", sigma, hardOK, trials, softOK, trials)
	if softOK <= hardOK {
		t.Errorf("soft TSBK yield %d not better than hard %d", softOK, hardOK)
	}
	if softOK < hardOK+trials/10 {
		t.Errorf("soft TSBK yield %d vs hard %d: expected a substantial (>10%% of trials) margin at this SNR", softOK, hardOK)
	}
}
