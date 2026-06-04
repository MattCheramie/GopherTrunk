package siglab

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
