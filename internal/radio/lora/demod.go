package lora

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
)

// dechirper holds the per-(SF,osf) reference chirps and FFT plan used to
// turn one symbol window into a peak bin. It is the reusable kernel behind
// both the streaming Channel and the offline frame demodulator.
type dechirper struct {
	sf, osf int
	n       int // 2^sf  (number of bins / chips)
	sps     int // n*osf (samples per symbol = FFT size)
	up      []complex64
	down    []complex64
	plan    fft.Plan

	// scratch buffers reused across calls.
	y    []complex64
	in   []complex128
	out  []complex128
	mags []float64
}

func newDechirper(sf, osf int) *dechirper {
	n := 1 << uint(sf)
	sps := n * osf
	return &dechirper{
		sf:   sf,
		osf:  osf,
		n:    n,
		sps:  sps,
		up:   GenChirp(sf, osf, true),
		down: GenChirp(sf, osf, false),
		plan: fft.New(sps),
		y:    make([]complex64, sps),
		in:   make([]complex128, sps),
		out:  make([]complex128, sps),
		mags: make([]float64, n),
	}
}

// peak holds the result of a dechirp+FFT: the argmax bin, the interpolated
// fractional offset of the true peak (for fine-CFO/timing), and a
// peak-to-runner-up power ratio used as a detection confidence.
type peak struct {
	bin  int
	frac float64 // fractional bin offset in (-0.5, 0.5]
	snr  float64 // peak / second-strongest bin power
	conc float64 // peak / mean bin power — robust when a tone straddles two bins
	mag  float64 // peak power
}

// peakOf dechirps a window against ref and returns the folded-FFT peak.
// window is read-only (copied into scratch). ref is up or down.
func (d *dechirper) peakOf(window, ref []complex64) peak {
	for i := 0; i < d.sps; i++ {
		d.y[i] = window[i] * ref[i]
	}
	d.in = fft.Cmplx64ToCmplx128(d.in, d.y)
	d.out = d.plan.Forward(d.out, d.in)

	// Fold the osf spectral copies: bin b aliases with b + r*n.
	best, second := -1, 0.0
	bestMag := -1.0
	var total float64
	for b := 0; b < d.n; b++ {
		var sum float64
		for r := 0; r < d.osf; r++ {
			c := d.out[b+r*d.n]
			re, im := real(c), imag(c)
			sum += re*re + im*im
		}
		d.mags[b] = sum
		total += sum
		if sum > bestMag {
			second = bestMag
			bestMag = sum
			best = b
		} else if sum > second {
			second = sum
		}
	}

	// Quadratic interpolation around the peak for a fractional estimate.
	frac := 0.0
	if best >= 0 {
		l := d.mags[(best-1+d.n)%d.n]
		c := d.mags[best]
		r := d.mags[(best+1)%d.n]
		denom := l - 2*c + r
		if denom != 0 {
			frac = 0.5 * (l - r) / denom
			if frac > 0.5 || frac < -0.5 {
				frac = 0
			}
		}
	}
	snr := 0.0
	conc := 0.0
	switch {
	case bestMag <= 1e-12:
		// empty / silent window — never a valid lock.
	case second <= 0:
		snr = math.Inf(1)
		conc = math.Inf(1)
	default:
		snr = bestMag / second
		meanMag := total / float64(d.n)
		if meanMag > 0 {
			conc = bestMag / meanMag
		}
	}
	return peak{bin: best, frac: frac, snr: snr, conc: conc, mag: bestMag}
}

// dataPeak / sfdPeak are the two dechirp flavours.
func (d *dechirper) dataPeak(window []complex64) peak { return d.peakOf(window, d.down) }
func (d *dechirper) sfdPeak(window []complex64) peak  { return d.peakOf(window, d.up) }

// --- frame-level demodulation -------------------------------------------

// preambleSyms is the minimum number of consecutive matching upchirp
// windows that declares a preamble. Real LoRa preambles are 8+ symbols;
// requiring 4 tolerates a late lock while rejecting noise.
const preambleSyms = 4

// frameSync is the result of locating one frame inside a sample buffer.
type frameSync struct {
	dataStart int     // sample index of the first header symbol
	cfoBins   float64 // carrier offset in bins (integer + fractional)
	timing    int     // residual integer chip offset already compensated
	ok        bool

	// debug fields (not used in production decode paths).
	lockPos, boundary, sfdPos, bu, bd int
}

// syncFrame scans buf for a LoRa preamble + SFD and returns where the
// payload symbols begin together with the estimated carrier offset. It
// implements the classic dechirp synchroniser:
//
//   - a base upchirp dechirped against the downchirp peaks at bin
//     (cfo + timing) mod N — stable across the repeated preamble;
//   - the SFD downchirps dechirped against the upchirp peak at
//     (cfo - timing) mod N;
//   - solving the pair separates carrier offset from symbol timing.
//
// start bounds the search; buf must hold at least a few symbols past the
// preamble. Returns ok=false if no preamble is found.
func (d *dechirper) syncFrame(buf []complex64, start int) frameSync {
	sps, n, osf := d.sps, d.n, d.osf
	// concThresh gates "a chirp is present here". A single tone — even when
	// it straddles two bins under fractional carrier offset — concentrates
	// far more energy in its peak bin than the per-bin mean, so this stays
	// reliable where a peak/runner-up ratio would collapse.
	const concThresh = 16.0

	// 1. Slide a window one chip at a time looking for preambleSyms
	//    consecutive strong, stable upchirp peaks. The repeated preamble
	//    dechirps (against the downchirp) to a stable bin bu = (timing +
	//    cfo) mod N.
	var bu int
	var buFrac float64
	hits := 0
	lockPos := -1
	for pos := start; pos+sps <= len(buf); pos += osf {
		p := d.dataPeak(buf[pos : pos+sps])
		if p.conc > concThresh {
			if hits == 0 || abs(p.bin-bu) <= 1 {
				bu, buFrac = p.bin, p.frac
				hits++
				if hits >= preambleSyms {
					lockPos = pos
					break
				}
			} else {
				hits, bu, buFrac = 1, p.bin, p.frac
			}
		} else {
			hits = 0
		}
	}
	if lockPos < 0 {
		return frameSync{}
	}

	// 2. Realign to a symbol boundary using the preamble bin. Subtracting
	//    bu*osf samples puts a preamble window's energy back at bin 0
	//    (timing and integer-CFO are jointly absorbed); for a clean signal
	//    this is the exact symbol grid.
	boundary := lockPos - bu*osf
	for boundary < 0 {
		boundary += sps
	}

	// 3. From the realigned boundary, step symbol-by-symbol until the SFD
	//    downchirps appear (strong peak when dechirped with the upchirp,
	//    stronger than the data dechirp).
	sfdPos := -1
	var bd int
	var bdFrac float64
	maxScan := boundary + (PreambleSymbols+8)*sps
	for p := boundary; p+sps <= len(buf) && p <= maxScan; p += sps {
		sp := d.sfdPeak(buf[p : p+sps])
		dp := d.dataPeak(buf[p : p+sps])
		if sp.conc > concThresh && sp.mag > dp.mag {
			sfdPos, bd, bdFrac = p, sp.bin, sp.frac
			break
		}
	}
	if sfdPos < 0 {
		return frameSync{}
	}

	// 4. Recover the carrier offset from the SFD. On the bu-realigned grid
	//    the residual timing equals -cfo, so up-dechirping the downchirp
	//    yields bd = (cfo - timing) = 2*cfo. Hence cfo = bd/2.
	cfo := float64(modCenter(bd, n))/2 + bdFrac/2
	_ = buFrac

	// 5. The header begins 2.25 symbols after the SFD start. The grid is
	//    cfo chips early (timing = -cfo), so nudge the read pointer forward
	//    by cfo chips — at sample resolution, so a half-chip fractional
	//    offset is corrected too. The fractional carrier offset that
	//    remains is removed later by mixing.
	timingSamples := int(math.Round(cfo * float64(osf)))
	dataStart := sfdPos + 2*sps + sps/4 + timingSamples
	if dataStart < 0 || dataStart+sps > len(buf) {
		return frameSync{}
	}
	return frameSync{
		dataStart: dataStart, cfoBins: cfo, timing: timingSamples, ok: true,
		lockPos: lockPos, boundary: boundary, sfdPos: sfdPos, bu: bu, bd: bd,
	}
}

// demodSymbols reads count data symbols starting at sample index start,
// applies the integer-bin carrier-offset correction, and returns the raw
// (pre-Gray) symbol values. Fractional CFO is removed by pre-mixing buf.
func (d *dechirper) demodSymbols(buf []complex64, start, count int, cfoBins float64) []uint16 {
	out := make([]uint16, 0, count)
	cfoI := int(math.Round(cfoBins))
	for i := 0; i < count; i++ {
		s := start + i*d.sps
		if s+d.sps > len(buf) {
			break
		}
		p := d.dataPeak(buf[s : s+d.sps])
		v := modN(p.bin-cfoI, d.n)
		out = append(out, uint16(v))
	}
	return out
}

// --- small integer helpers ----------------------------------------------

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func modN(x, n int) int {
	x %= n
	if x < 0 {
		x += n
	}
	return x
}

// modCenter reduces x mod n into the symmetric range (-n/2, n/2], which is
// what the cfo/timing half-sum solve needs so that, e.g., bu+bd just below
// 2N maps to a small negative offset rather than ~2N.
func modCenter(x, n int) int {
	x = modN(x, n)
	if x > n/2 {
		x -= n
	}
	return x
}
