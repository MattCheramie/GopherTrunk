package survey

import (
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// RecoverPayload blind-demodulates a digital burst's baseband IQ into a byte
// payload for entropy/structure triage. It FM-discriminates the burst, slices
// the discriminator at the detected symbol rate into 2- or 4-level symbols
// (integrate-and-dump), maps each symbol to bits, and packs MSB-first.
//
// This is deliberately triage-grade, not a protocol-exact demod: its job is to
// produce a bitstream whose entropy/structure tells random-keystream from
// structured-scrambler-from-plaintext, the question the cryptolab engines answer.
// FM discrimination covers the FSK/C4FM family that dominates non-WiFi digital
// RF; a PSK carrier still yields a usable (if noisier) stream. Shared by the
// rfscope entropy analyzer and the hunt survey's encryption triage.
func RecoverPayload(iq []complex64, rateHz float64, feat ClassFeatures) []byte {
	if rateHz <= 0 || feat.BaudHz <= 0 || len(iq) == 0 {
		return nil
	}
	sps := rateHz / feat.BaudHz
	if sps < 2 {
		return nil
	}
	disc := demod.NewFM().Process(nil, iq)
	nSym := int(float64(len(disc)) / sps)
	if nSym < 8 {
		return nil
	}
	syms := make([]float64, 0, nSym)
	for i := 0; i < nSym; i++ {
		start := int(float64(i) * sps)
		end := int(float64(i+1) * sps)
		if end > len(disc) {
			end = len(disc)
		}
		if start >= end {
			break
		}
		var s float64
		for j := start; j < end; j++ {
			s += float64(disc[j])
		}
		syms = append(syms, s/float64(end-start))
	}

	levels := 2
	if feat.IFModality == 4 {
		levels = 4
	}
	return packBits(sliceSymbols(syms, levels))
}

// sliceSymbols converts integrate-and-dump symbol samples to bits. For 2 levels
// it slices at the median (1 bit/symbol); for 4 levels it slices at the 25/50/75
// percentiles and emits the 2-bit level MSB-first. Thresholds are derived from
// the data so no AGC/DC calibration is required.
func sliceSymbols(syms []float64, levels int) []byte {
	if len(syms) == 0 {
		return nil
	}
	sorted := append([]float64(nil), syms...)
	sort.Float64s(sorted)
	q := func(p float64) float64 {
		idx := int(p * float64(len(sorted)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return sorted[idx]
	}

	var bits []byte
	if levels == 4 {
		t1, t2, t3 := q(0.25), q(0.50), q(0.75)
		for _, s := range syms {
			lvl := 0
			if s >= t1 {
				lvl++
			}
			if s >= t2 {
				lvl++
			}
			if s >= t3 {
				lvl++
			}
			bits = append(bits, byte((lvl>>1)&1), byte(lvl&1))
		}
		return bits
	}
	// Strict '>' so the median (which lands on a cluster value for a clean
	// 2-level signal) splits the two clusters instead of flooding to 1s.
	t := q(0.50)
	for _, s := range syms {
		if s > t {
			bits = append(bits, 1)
		} else {
			bits = append(bits, 0)
		}
	}
	return bits
}

// packBits packs 0/1 values into bytes, MSB-first, matching the bit order
// cryptolab's lfsr.BitsFromBytes unpacks. A trailing partial byte is dropped so
// the payload is whole bytes.
func packBits(bits []byte) []byte {
	n := len(bits) / 8
	if n == 0 {
		return nil
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		var b byte
		for j := 0; j < 8; j++ {
			b = (b << 1) | (bits[i*8+j] & 1)
		}
		out[i] = b
	}
	return out
}
