package ka9qradio

import (
	"encoding/binary"
	"fmt"
	"math"
)

// rtpHeader is the decoded fixed RTP header (src/rtp.h struct rtp_header /
// ntoh_rtp). Only the fields this driver uses are kept.
type rtpHeader struct {
	version int
	pad     bool
	ext     bool
	cc      int
	marker  bool
	pt      uint8
	seq     uint16
	ts      uint32
	ssrc    uint32
}

// parseRTP decodes the RTP header at the front of a datagram and returns the
// header plus the offset where the payload begins, skipping any CSRC list and
// header extension (radiod's ntoh_rtp). It validates the version and bounds so
// a malformed packet is rejected rather than slicing out of range.
func parseRTP(p []byte) (rtpHeader, int, error) {
	var h rtpHeader
	if len(p) < 12 {
		return h, 0, fmt.Errorf("ka9qradio: RTP packet too short (%d bytes)", len(p))
	}
	w := binary.BigEndian.Uint32(p[0:4])
	h.version = int(w >> 30)
	h.pad = (w>>29)&1 != 0
	h.ext = (w>>28)&1 != 0
	h.cc = int((w >> 24) & 0xf)
	h.marker = (w>>23)&1 != 0
	h.pt = uint8((w >> 16) & 0x7f)
	h.seq = uint16(w & 0xffff)
	h.ts = binary.BigEndian.Uint32(p[4:8])
	h.ssrc = binary.BigEndian.Uint32(p[8:12])
	if h.version != 2 {
		return h, 0, fmt.Errorf("ka9qradio: RTP version %d, want 2", h.version)
	}

	off := 12 + 4*h.cc
	if off > len(p) {
		return h, 0, fmt.Errorf("ka9qradio: RTP CSRC list overruns packet")
	}
	if h.ext {
		if off+4 > len(p) {
			return h, 0, fmt.Errorf("ka9qradio: RTP extension header overruns packet")
		}
		// Extension: 16-bit profile + 16-bit length (in 32-bit words); skip both.
		extWords := int(binary.BigEndian.Uint16(p[off+2 : off+4]))
		off += 4 + 4*extWords
		if off > len(p) {
			return h, 0, fmt.Errorf("ka9qradio: RTP extension body overruns packet")
		}
	}
	return h, off, nil
}

// decodeIQ converts a raw-IQ RTP payload (interleaved I,Q in enc) to
// normalized complex64. S16 samples are scaled by 1/32768 so they land in
// roughly [-1, 1), matching the rtltcp / rtlsdr drivers; float samples pass
// through unchanged (radiod already emits them near unit scale). A trailing
// odd sample (incomplete I/Q pair) is dropped.
func decodeIQ(payload []byte, enc encoding) ([]complex64, error) {
	switch enc {
	case encS16LE:
		return decodeS16(payload, binary.LittleEndian), nil
	case encS16BE:
		return decodeS16(payload, binary.BigEndian), nil
	case encF32LE:
		return decodeF32(payload, binary.LittleEndian), nil
	case encF32BE:
		return decodeF32(payload, binary.BigEndian), nil
	default:
		return nil, fmt.Errorf("ka9qradio: encoding %s is not raw IQ", enc)
	}
}

type byteOrder interface {
	Uint16([]byte) uint16
	Uint32([]byte) uint32
}

func decodeS16(p []byte, bo byteOrder) []complex64 {
	n := len(p) / 4 // 2 bytes I + 2 bytes Q per sample
	out := make([]complex64, n)
	const scale = 1.0 / 32768.0
	for i := 0; i < n; i++ {
		iv := int16(bo.Uint16(p[4*i : 4*i+2]))
		qv := int16(bo.Uint16(p[4*i+2 : 4*i+4]))
		out[i] = complex(float32(iv)*scale, float32(qv)*scale)
	}
	return out
}

func decodeF32(p []byte, bo byteOrder) []complex64 {
	n := len(p) / 8 // 4 bytes I + 4 bytes Q per sample
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := math.Float32frombits(bo.Uint32(p[8*i : 8*i+4]))
		qv := math.Float32frombits(bo.Uint32(p[8*i+4 : 8*i+8]))
		out[i] = complex(iv, qv)
	}
	return out
}
