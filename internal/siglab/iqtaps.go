package siglab

import "github.com/MattCheramie/GopherTrunk/internal/dsp/phase"

// IQPoint is one complex sample on the wire. Float32 keeps the captured
// stream compact; web consumers render it directly without conversion. The
// JSON tags match internal/api's IQPoint (the live Constellation panel) so
// the offline siglab taps and the live diag stream share a wire shape.
type IQPoint struct {
	I float32 `json:"i"`
	Q float32 `json:"q"`
}

// IQTaps is the optional decimated signal capture attached to a Result when
// Config.CaptureIQ is set. It carries a stride-decimated copy of the
// post-DDC (channelized) IQ — what the constellation, spectrogram and PSD
// views render — plus the pre-slicer soft samples (deep P25 C4FM path) the
// eye diagram folds. Both slices are independently hard-capped at
// Config.CaptureIQMaxPoints, so the taps never grow without bound and never
// ship full-rate IQ.
//
// It is deliberately heavy (potentially MBs), so the API/export layer strips
// it from the Result before serializing the summary or running the CSV/JSON
// exporters; it is retrieved separately (GET .../iq) when a view needs it.
type IQTaps struct {
	// DecimatedIQ is the channelized IQ sampled every Stride samples.
	DecimatedIQ []IQPoint `json:"decimated_iq" yaml:"decimated_iq"`
	// SoftSamples is the pre-slicer soft-sample stream (deep P25 path);
	// empty for protocols/paths that do not emit soft samples.
	SoftSamples []float32 `json:"soft_samples" yaml:"soft_samples"`
	// DecimatedRateHz is the effective sample rate of DecimatedIQ
	// (PipelineRate / Stride).
	DecimatedRateHz float64 `json:"decimated_rate_hz" yaml:"decimated_rate_hz"`
	// Stride is the input-to-output downsample ratio applied to the
	// channelized stream.
	Stride int `json:"stride" yaml:"stride"`

	// SymbolDibits is the recovered-symbol decision stream (values
	// 0..SymbolCardinality-1), decimated to the same cap as the IQ. It
	// backs the offline Symbol-scope viz (the OP25-style time series).
	// Empty for protocols/paths that buffer no symbols.
	SymbolDibits []uint8 `json:"symbol_dibits,omitempty" yaml:"symbol_dibits,omitempty"`
	// SymbolSoft is the pre-slicer soft waveform aligned index-for-index
	// with SymbolDibits (deep P25 C4FM path). Empty when the demod path
	// emits no soft samples (e.g. CQPSK); the viz then shows the dibit
	// rows only.
	SymbolSoft []float32 `json:"symbol_soft,omitempty" yaml:"symbol_soft,omitempty"`
	// SymbolCardinality is 4 for the dibit (4-level) protocols, 2 for the
	// bit (2-level) protocols.
	SymbolCardinality int `json:"symbol_cardinality,omitempty" yaml:"symbol_cardinality,omitempty"`

	// DiffPhase is the per-symbol differential phase (radians, in (-π, π]) of
	// the complex constellation — angle(z[i]·conj(z[i-1])) — decimated to the
	// same cap as the IQ. It is the rotation signal that backs the offline
	// rotation-tracker viz: a π/4-DQPSK control channel clusters these around
	// ±π/4 and ±3π/4. Populated only on the CQPSK / π4-DQPSK path (the deep
	// constellation buffer); empty for C4FM and the 2-level paths.
	DiffPhase []float32 `json:"diff_phase,omitempty" yaml:"diff_phase,omitempty"`

	// SymbolIQ is the recovered complex symbol-decision constellation — the
	// post-timing-recovery decision points, one per symbol, decimated to the
	// same cap as the IQ. Unlike DecimatedIQ (the raw channelized stream, which
	// renders as a diffuse disc), these are the actual demod outputs, so a
	// π/4-DQPSK control channel renders as discrete decision clusters. It backs
	// the constellation viz's "Symbols" mode. Populated only on the CQPSK /
	// π4-DQPSK path (the deep constellation buffer); empty for C4FM and the
	// 2-level paths, which buffer no complex constellation.
	SymbolIQ []IQPoint `json:"symbol_iq,omitempty" yaml:"symbol_iq,omitempty"`
}

// decimateSymbolSeries stride-decimates an aligned dibit + soft stream
// down to at most maxPoints points, preserving alignment. The soft track
// is carried only when it matches the dibit length (a soft-less path
// returns dibits with a nil soft slice).
func decimateSymbolSeries(dibits []uint8, soft []float32, maxPoints int) (outD []uint8, outS []float32) {
	n := len(dibits)
	if n == 0 || maxPoints <= 0 {
		return nil, nil
	}
	hasSoft := len(soft) == n
	stride := 1
	if n > maxPoints {
		stride = (n + maxPoints - 1) / maxPoints
	}
	for i := 0; i < n; i += stride {
		outD = append(outD, dibits[i])
		if hasSoft {
			outS = append(outS, soft[i])
		}
	}
	return outD, outS
}

// diffPhaseSeries computes the per-symbol differential phase of a complex
// constellation buffer — the π/4-DQPSK rotation signal — and decimates it to
// at most maxPoints points. It returns nil when there are too few points to
// form a rotation (need at least two constellation samples) or maxPoints ≤ 0.
func diffPhaseSeries(constBuf []complex64, maxPoints int) []float32 {
	if len(constBuf) < 2 || maxPoints <= 0 {
		return nil
	}
	z := make([]complex128, len(constBuf))
	for i, c := range constBuf {
		z[i] = complex(float64(real(c)), float64(imag(c)))
	}
	d := phase.Diff(z) // len == len(z)-1, wrapped to (-π, π]
	stride := 1
	if len(d) > maxPoints {
		stride = (len(d) + maxPoints - 1) / maxPoints
	}
	out := make([]float32, 0, (len(d)+stride-1)/stride)
	for i := 0; i < len(d); i += stride {
		out = append(out, float32(d[i]))
	}
	return out
}

// symbolIQSeries stride-decimates a complex constellation buffer — the
// recovered per-symbol decision points — to at most maxPoints IQPoints,
// preserving each point's I/Q. It is the 2-D companion to diffPhaseSeries
// (same buffer, but the whole complex point instead of the differential
// phase). Returns nil when the buffer is empty or maxPoints ≤ 0.
func symbolIQSeries(constBuf []complex64, maxPoints int) []IQPoint {
	if len(constBuf) == 0 || maxPoints <= 0 {
		return nil
	}
	stride := 1
	if len(constBuf) > maxPoints {
		stride = (len(constBuf) + maxPoints - 1) / maxPoints
	}
	out := make([]IQPoint, 0, (len(constBuf)+stride-1)/stride)
	for i := 0; i < len(constBuf); i += stride {
		out = append(out, IQPoint{I: real(constBuf[i]), Q: imag(constBuf[i])})
	}
	return out
}

// iqTapBuffer accumulates the decimated channelized IQ and soft samples for
// an IQTaps. It is driven from the engine read loop (single goroutine), so it
// needs no locking. Both buffers are capped independently; once a buffer is
// full further observations for it are dropped.
type iqTapBuffer struct {
	stride    int
	maxPoints int

	phase int // running sample index mod stride, for the IQ decimation
	iq    []IQPoint
	soft  []float32
}

// newIQTapBuffer builds a tap buffer that decimates a stream at pipelineRate
// down toward captureIQTargetRateHz, capped at maxPoints retained points.
func newIQTapBuffer(pipelineRateHz float64, maxPoints int) *iqTapBuffer {
	stride := 1
	if pipelineRateHz > captureIQTargetRateHz && captureIQTargetRateHz > 0 {
		stride = int(pipelineRateHz / captureIQTargetRateHz)
	}
	if stride < 1 {
		stride = 1
	}
	return &iqTapBuffer{
		stride:    stride,
		maxPoints: maxPoints,
		iq:        make([]IQPoint, 0, 4096),
	}
}

// observeIQ folds a chunk of channelized IQ into the decimated buffer,
// striding across chunk boundaries and stopping once the cap is reached.
func (b *iqTapBuffer) observeIQ(chunk []complex64) {
	if b == nil || len(b.iq) >= b.maxPoints {
		return
	}
	for _, s := range chunk {
		if b.phase == 0 {
			b.iq = append(b.iq, IQPoint{I: real(s), Q: imag(s)})
			if len(b.iq) >= b.maxPoints {
				b.phase = 0
				return
			}
		}
		b.phase++
		if b.phase >= b.stride {
			b.phase = 0
		}
	}
}

// observeSoft appends pre-slicer soft samples up to the cap.
func (b *iqTapBuffer) observeSoft(soft []float32) {
	if b == nil || len(b.soft) >= b.maxPoints {
		return
	}
	room := b.maxPoints - len(b.soft)
	if room < len(soft) {
		soft = soft[:room]
	}
	b.soft = append(b.soft, soft...)
}

// result builds the IQTaps from the accumulated buffers. pipelineRateHz is
// the channelized (post-DDC) rate so the decimated rate is reported faithfully.
func (b *iqTapBuffer) result(pipelineRateHz float64) *IQTaps {
	if b == nil {
		return nil
	}
	rate := pipelineRateHz
	if b.stride > 0 {
		rate = pipelineRateHz / float64(b.stride)
	}
	if b.iq == nil {
		b.iq = []IQPoint{}
	}
	if b.soft == nil {
		b.soft = []float32{}
	}
	return &IQTaps{
		DecimatedIQ:     b.iq,
		SoftSamples:     b.soft,
		DecimatedRateHz: rate,
		Stride:          b.stride,
	}
}
