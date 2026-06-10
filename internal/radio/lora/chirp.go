// Package lora implements a pure-Go LoRa (Semtech CSS) physical-layer
// modem: chirp synthesis, dechirp/FFT demodulation, the symbol -> bit
// forward-error-correction chain (Gray, diagonal interleaver, Hamming,
// whitening, CRC), an explicit-header codec, and a matching modulator
// used to synthesise test IQ.
//
// LoRa modulates data as cyclic time-shifts of a linear chirp ("symbol").
// One symbol spans N = 2^SF chips; the integer shift in [0,N) is the
// symbol value. A receiver "dechirps" by multiplying against a locally
// generated reference chirp and taking an FFT — the peak bin recovers the
// symbol value. See demod.go for the receive state machine and
// modulator.go for the transmit path.
//
// The modulator and demodulator in this package are exact inverses, so the
// FEC / whitening / header transforms are verifiable by round-trip tests
// independent of any SDR. Bit-exact interoperability with Semtech silicon
// (whitening table, CRC seed, interleaver rotation direction) is validated
// separately against captured golden vectors — see the package tests.
package lora

import "math"

// Bandwidth is a LoRa channel bandwidth in Hz. Only the three standard
// values are used in the field.
type Bandwidth int

const (
	BW125 Bandwidth = 125_000
	BW250 Bandwidth = 250_000
	BW500 Bandwidth = 500_000
)

// SF bounds. LoRa defines spreading factors 7..12 (SF6 exists but is
// rarely deployed and uses implicit-header only — out of scope here).
const (
	MinSF = 7
	MaxSF = 12
)

// DefaultOversample is the samples-per-chip factor the modem runs at. A
// value of 2 puts the channel sample rate at 2*BW, which is enough to
// resolve the dechirp peak and estimate fractional carrier offset while
// keeping the FFT small. Folding (demod.go) reclaims the full SNR.
const DefaultOversample = 2

// SymbolLen returns the number of complex samples in one LoRa symbol for
// the given spreading factor and oversample factor: sps = 2^SF * osf.
func SymbolLen(sf, osf int) int { return (1 << uint(sf)) * osf }

// GenChirp builds one base chirp symbol of length SymbolLen(sf, osf). When
// up is true it returns the base upchirp whose instantaneous frequency
// sweeps from -BW/2 to +BW/2 across the symbol; when false it returns the
// downchirp (the complex conjugate), used as the dechirp reference for
// data symbols and as the SFD template.
//
// The waveform is defined at sample rate Fs = osf*BW so it is independent
// of the absolute bandwidth — only osf and sf shape it:
//
//	phase(n) = 2*pi * ( -n/(2*osf) + n^2 / (2*N*osf^2) )
//	chirp[n] = exp(j*phase(n))           (conjugated when up == false)
func GenChirp(sf, osf int, up bool) []complex64 {
	n := 1 << uint(sf)
	sps := n * osf
	out := make([]complex64, sps)
	nf := float64(n)
	of := float64(osf)
	for k := 0; k < sps; k++ {
		kf := float64(k)
		phase := 2 * math.Pi * (-kf/(2*of) + (kf*kf)/(2*nf*of*of))
		s := complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
		if !up {
			s = complex(real(s), -imag(s)) // conjugate -> downchirp
		}
		out[k] = s
	}
	return out
}

// GenSymbolChirp builds the modulated upchirp carrying symbol value v in
// [0, 2^SF). A LoRa data symbol is the base upchirp advanced by v chips,
// i.e. a cyclic shift of the base upchirp by v*osf samples. Dechirping it
// against the base downchirp and folding the FFT yields a single peak at
// bin v (see demod.peakBin).
func GenSymbolChirp(sf, osf, v int) []complex64 {
	base := GenChirp(sf, osf, true)
	sps := len(base)
	shift := (v * osf) % sps
	out := make([]complex64, sps)
	for k := 0; k < sps; k++ {
		out[k] = base[(k+shift)%sps]
	}
	return out
}
