package symbolscope

import (
	"errors"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// defaultMixerFFTSize is the Mixer plot's FFT length. At the P25 DDC
// output rate (~48 kHz) 1024 bins is ~47 Hz/bin over the channel
// baseband — fine enough to see the carrier sit off-centre and snap back
// once the loop tunes it, without an over-large wire payload.
const defaultMixerFFTSize = 1024

// defaultMixerFPS caps how often the Mixer plot computes its two FFTs.
// The plot is a slow human-readable view, so 10 fps keeps the per-stream
// CPU cost negligible next to the receiver itself.
const defaultMixerFPS = 10.0

// MixerFrame is one snapshot of the channel baseband spectrum, the data
// behind GopherTrunk's Mixer plot (OP25's Raw Mixer / Tuned Mixer tabs).
//
// RawBins is the FFT of the channelized baseband as the receiver sees it
// — the carrier sits wherever the residual offset leaves it. TunedBins is
// the same window re-mixed by the receiver's own carrier-offset estimate,
// so a locked loop pulls the carrier to 0 Hz (centre bin). Both are dBFS,
// FFT-shifted (index 0 = -SampleRateHz/2, index n/2 = DC), length the
// configured FFT size. The slices are freshly allocated per frame; the
// callee owns them.
type MixerFrame struct {
	TimestampNs     int64
	CenterHz        uint32
	SampleRateHz    uint32
	OffsetHz        int32
	CarrierOffsetHz float64
	RawBins         []float32
	TunedBins       []float32
}

// mixerOptions configures a mixerAccum.
type mixerOptions struct {
	fftSize  int
	fps      float64
	sampleHz float64
	centerHz uint32
	offsetHz int32
	nowNs    func() int64
	emit     func(MixerFrame)
}

// mixerAccum buffers channelized baseband into FFT-sized windows and, at
// the configured frame rate, emits a raw + tuned spectrum pair. It is
// driven from the Engine's single Process goroutine — not safe for
// concurrent feed calls.
type mixerAccum struct {
	fftSize  int
	sampleHz float64
	centerHz uint32
	offsetHz int32
	periodNs int64
	nowNs    func() int64
	emit     func(MixerFrame)

	plan   fft.Plan
	win    []float64
	winSum float64
	bufIn  []complex128 // FFT scratch, reused
	bufOut []complex128 // FFT scratch, reused

	pending []complex64 // window accumulator (len → fftSize)
	mixBuf  []complex64 // tuned re-mix scratch, reused
	nextNs  int64       // earliest nowNs at which to emit the next frame
}

// newMixerAccum builds a mixerAccum, validating the FFT size is a
// positive power of two and the rate is usable.
func newMixerAccum(opts mixerOptions) (*mixerAccum, error) {
	if opts.emit == nil {
		return nil, errors.New("symbolscope: mixer emit is required")
	}
	if opts.sampleHz <= 0 {
		return nil, errors.New("symbolscope: mixer sampleHz must be > 0")
	}
	size := opts.fftSize
	if size <= 0 {
		size = defaultMixerFFTSize
	}
	if size&(size-1) != 0 {
		return nil, errors.New("symbolscope: mixer FFT size must be a power of two")
	}
	fps := opts.fps
	if fps <= 0 {
		fps = defaultMixerFPS
	}
	nowNs := opts.nowNs
	if nowNs == nil {
		return nil, errors.New("symbolscope: mixer nowNs is required")
	}

	win := window.Hann(size)
	var winSum float64
	for _, w := range win {
		winSum += w * w
	}
	return &mixerAccum{
		fftSize:  size,
		sampleHz: opts.sampleHz,
		centerHz: opts.centerHz,
		offsetHz: opts.offsetHz,
		periodNs: int64(float64(1e9) / fps),
		nowNs:    nowNs,
		emit:     opts.emit,
		plan:     fft.New(size),
		win:      win,
		winSum:   winSum,
		bufIn:    make([]complex128, size),
		bufOut:   make([]complex128, size),
		pending:  make([]complex64, 0, size),
		mixBuf:   make([]complex64, size),
	}, nil
}

// feed appends a baseband chunk, emitting a MixerFrame each time a full
// FFT window has accumulated and the frame interval has elapsed. The
// carrier-offset estimate (read fresh from the receiver after this
// chunk) drives the tuned re-mix. baseband is consumed synchronously
// (copied into the window), so the caller may reuse it afterwards.
func (m *mixerAccum) feed(baseband []complex64, carrierOffsetHz float64) {
	src := baseband
	for len(src) > 0 {
		need := m.fftSize - len(m.pending)
		if len(src) < need {
			m.pending = append(m.pending, src...)
			return
		}
		m.pending = append(m.pending, src[:need]...)
		src = src[need:]
		now := m.nowNs()
		if now >= m.nextNs {
			m.emitFrame(carrierOffsetHz, now)
			m.nextNs = now + m.periodNs
		}
		m.pending = m.pending[:0]
	}
}

// emitFrame runs the two FFTs over the current full window and delivers
// the MixerFrame. The tuned view multiplies sample n by
// exp(-j·2π·f_off·n/Fs) so a carrier at +f_off lands on the centre bin;
// resetting the mix phase per window is harmless for a magnitude spectrum.
func (m *mixerAccum) emitFrame(carrierOffsetHz float64, nowNs int64) {
	raw := make([]float32, m.fftSize)
	spectrum.PowerDB(m.plan, m.win, m.winSum, m.pending, m.bufIn, m.bufOut, raw)

	w := -2 * math.Pi * carrierOffsetHz / m.sampleHz
	for n := 0; n < m.fftSize; n++ {
		ph := w * float64(n)
		rot := complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
		m.mixBuf[n] = m.pending[n] * rot
	}
	tuned := make([]float32, m.fftSize)
	spectrum.PowerDB(m.plan, m.win, m.winSum, m.mixBuf, m.bufIn, m.bufOut, tuned)

	m.emit(MixerFrame{
		TimestampNs:     nowNs,
		CenterHz:        m.centerHz,
		SampleRateHz:    uint32(m.sampleHz + 0.5),
		OffsetHz:        m.offsetHz,
		CarrierOffsetHz: carrierOffsetHz,
		RawBins:         raw,
		TunedBins:       tuned,
	})
}
