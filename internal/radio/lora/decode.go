package lora

import "math"

// Demodulator decodes LoRa frames from a baseband IQ buffer sampled at
// osf*BW. It is the offline / block-oriented entry point used by tests and
// by the streaming receiver (Phase 2) once a frame region is buffered.
// One Demodulator is bound to a single spreading factor; the wideband
// receiver runs a Demodulator per candidate SF for auto-detection.
type Demodulator struct {
	d    *dechirper
	bw   Bandwidth
	ldro bool // reduced-rate payload (see ldro.go)
}

// NewDemodulator builds a Demodulator for the given SF / oversample /
// bandwidth, auto-detecting Low Data Rate Optimization from the (SF, BW)
// pair (the Semtech Ts >= 16 ms rule). Use NewDemodulatorMode to force it.
func NewDemodulator(sf, osf int, bw Bandwidth) *Demodulator {
	return NewDemodulatorMode(sf, osf, bw, LDROAuto)
}

// NewDemodulatorMode builds a Demodulator with an explicit LDRO mode.
// LDROAuto applies the (SF, BW) rule; LDROOn / LDROOff force it, for a
// network that deviates from the recommendation.
func NewDemodulatorMode(sf, osf int, bw Bandwidth, mode LDROMode) *Demodulator {
	return &Demodulator{d: newDechirper(sf, osf), bw: bw, ldro: ResolveLDRO(mode, sf, bw)}
}

// LDRO reports whether this demodulator decodes the payload in reduced-rate
// (Low Data Rate Optimize) mode.
func (m *Demodulator) LDRO() bool { return m.ldro }

// SF reports the spreading factor this demodulator is tuned to.
func (m *Demodulator) SF() int { return m.d.sf }

// SymbolLen reports the number of samples in one symbol (sps).
func (m *Demodulator) SymbolLen() int { return m.d.sps }

// DecodeFrom finds and decodes the first frame at or after start. It
// returns the decoded frame (when ok), the sample index just past the
// region it consumed (so a streaming caller can advance), and ok. When no
// preamble is found, end == len(buf) and ok is false.
func (m *Demodulator) DecodeFrom(buf []complex64, start int) (f Frame, end int, ok bool) {
	sync := m.d.syncFrame(buf, start)
	if !sync.ok {
		return Frame{}, len(buf), false
	}
	f, ok = m.decodeFrameAt(buf, sync.dataStart, sync.cfoBins)
	// Always advance at least one symbol past the located preamble. The
	// realignment can place dataStart at or behind the cursor; without this
	// floor a failed (or behind-cursor) lock would stall the caller, which
	// re-scans the same region indefinitely.
	minNext := sync.lockPos + m.d.sps
	if minNext <= start {
		minNext = start + m.d.sps
	}
	if !ok {
		return f, minNext, false
	}
	end = sync.dataStart + len(f.RawSymbols)*m.d.sps
	if end < minNext {
		end = minNext
	}
	return f, end, true
}

// DecodeFirst finds and decodes the first frame in buf, returning ok=false
// if no valid frame is present.
func (m *Demodulator) DecodeFirst(buf []complex64) (Frame, bool) {
	sync := m.d.syncFrame(buf, 0)
	if !sync.ok {
		return Frame{}, false
	}
	return m.decodeFrameAt(buf, sync.dataStart, sync.cfoBins)
}

// DecodeBuffer returns every frame it can find in buf.
func (m *Demodulator) DecodeBuffer(buf []complex64) []Frame {
	var out []Frame
	start := 0
	for start+m.d.sps <= len(buf) {
		sync := m.d.syncFrame(buf, start)
		if !sync.ok {
			break
		}
		f, ok := m.decodeFrameAt(buf, sync.dataStart, sync.cfoBins)
		if ok {
			out = append(out, f)
			// Advance past the decoded frame.
			adv := sync.dataStart + len(f.RawSymbols)*m.d.sps
			if adv <= start {
				adv = start + m.d.sps
			}
			start = adv
		} else {
			start = sync.dataStart + m.d.sps
		}
	}
	return out
}

// decodeFrameAt demodulates the header to learn the payload geometry, then
// reads the full symbol set and decodes the frame. cfoBins is removed by
// mixing so the FFT peaks land on the true symbol bins.
func (m *Demodulator) decodeFrameAt(buf []complex64, dataStart int, cfoBins float64) (Frame, bool) {
	hc := headerSymbolCount()

	// Header first, to recover payload length + coding rate.
	hdrRegion := m.mix(buf, dataStart, hc*m.d.sps, cfoBins)
	hsyms := m.d.demodSymbols(hdrRegion, 0, hc, 0)
	hdrNibs, ok := decodeBlock(hsyms, m.d.sf, HeaderCR)
	if !ok {
		return Frame{}, false
	}
	hdr := ParseHeader(hdrNibs)
	if !hdr.Valid {
		return Frame{SF: m.d.sf, HeaderOK: false, LDRO: m.ldro}, false
	}

	nBlocks := payloadBlockCount(m.d.sf, hdr.PayloadLen, hdr.HasCRC, m.ldro)
	total := hc + nBlocks*blockSymbols(hdr.CR)

	region := m.mix(buf, dataStart, total*m.d.sps, cfoBins)
	syms := m.d.demodSymbols(region, 0, total, 0)
	f, ok := decodeFrameFromSymbols(syms, m.d.sf, m.ldro)
	if !ok {
		return f, false
	}
	f.BW = m.bw
	f.CFOHz = cfoBins * float64(int(m.bw)) / float64(m.d.n)
	if hsyms != nil {
		// peak SNR of the first header symbol, in dB, as a quality proxy.
		p := m.d.dataPeak(hdrRegion[:m.d.sps])
		if p.snr > 0 && !math.IsInf(p.snr, 1) {
			f.SNRdB = 10 * math.Log10(p.snr)
		}
	}
	return f, true
}

// mix returns a length-clamped copy of buf[start:start+length] with the
// carrier offset removed: sample k multiplied by exp(-j2*pi*cfoBins*k /
// (N*osf)). A pure phase ramp; the constant phase term is irrelevant to
// magnitude-based bin detection.
func (m *Demodulator) mix(buf []complex64, start, length int, cfoBins float64) []complex64 {
	if start < 0 {
		start = 0
	}
	end := start + length
	if end > len(buf) {
		end = len(buf)
	}
	out := make([]complex64, end-start)
	if cfoBins == 0 {
		copy(out, buf[start:end])
		return out
	}
	w := 2 * math.Pi * cfoBins / float64(m.d.n*m.d.osf)
	for k := range out {
		ph := -w * float64(k)
		rot := complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
		out[k] = buf[start+k] * rot
	}
	return out
}
