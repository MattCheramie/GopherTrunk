package main

import (
	"encoding/binary"
	"io"
	"sync"
)

// pcmStreamWriter implements composer.PCMSink (and the recorder's
// voice.DecodedPCMSink, which has the same WritePCM shape) over an io.Writer,
// encoding each chunk as raw signed 16-bit little-endian mono PCM. It is the
// `replay -audio-out` live audio tap (issue #314): an external consumer
// (OpenWebRX+, aplay, a FIFO) reads a continuous PCM stream at the composer's
// PCM rate (8000 Hz) as calls decode, instead of waiting for per-call WAV
// files to land on disk.
//
// It is wired at two points, mirroring the daemon's live-audio fan-out:
// analog FM chains write PCM through the composer's main sink fan-out, while
// digital protocols emit raw vocoder frames that only the recorder decodes —
// so the recorder's decoded-PCM tap (SetDecodedPCMSink) carries digital voice
// here. The two paths are disjoint per call (a call is either analog or
// digital), so nothing is written twice.
//
// The first write error is retained and later chunks are dropped: a torn-down
// downstream reader must not wedge or kill the decode (the composer already
// ignores sink errors, so sticky-and-drop keeps the failure local and cheap).
type pcmStreamWriter struct {
	mu  sync.Mutex
	w   io.Writer
	buf []byte
	err error
}

func newPCMStreamWriter(w io.Writer) *pcmStreamWriter {
	return &pcmStreamWriter{w: w}
}

// WritePCM encodes samples as s16le and writes them through. The serial is
// ignored: the replay rig decodes one carrier, so every call's audio belongs
// on the one output stream.
func (p *pcmStreamWriter) WritePCM(_ string, samples []int16) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	if cap(p.buf) < 2*len(samples) {
		p.buf = make([]byte, 2*len(samples))
	}
	buf := p.buf[:2*len(samples)]
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(s))
	}
	_, p.err = p.w.Write(buf)
	return p.err
}

// Err reports the first write error, if any.
func (p *pcmStreamWriter) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}
