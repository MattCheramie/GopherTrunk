package ka9qradio

import (
	"encoding/binary"
	"math"
	"testing"
)

// buildRTP assembles a minimal RTP packet (version 2, no CSRC/extension) with
// the given payload type, sequence, ssrc and payload — enough to exercise the
// parser and the streaming path.
func buildRTP(pt uint8, seq uint16, ssrc uint32, payload []byte) []byte {
	pkt := make([]byte, 12+len(payload))
	w := uint32(2)<<30 | uint32(pt&0x7f)<<16 | uint32(seq)
	binary.BigEndian.PutUint32(pkt[0:4], w)
	binary.BigEndian.PutUint32(pkt[4:8], 0) // timestamp
	binary.BigEndian.PutUint32(pkt[8:12], ssrc)
	copy(pkt[12:], payload)
	return pkt
}

func TestParseRTPSkipsCSRCAndExtension(t *testing.T) {
	// version 2, extension bit set, cc=2, pt=96, seq=7.
	pkt := make([]byte, 0, 64)
	w := uint32(2)<<30 | uint32(1)<<28 /*ext*/ | uint32(2)<<24 /*cc*/ | uint32(96)<<16 | 7
	var hdr [12]byte
	binary.BigEndian.PutUint32(hdr[0:4], w)
	binary.BigEndian.PutUint32(hdr[8:12], 0xDEADBEEF)
	pkt = append(pkt, hdr[:]...)
	pkt = append(pkt, make([]byte, 4*2)...) // two CSRCs
	// Extension header: profile(2) + length(2 words) then 1 word of ext data.
	ext := []byte{0x00, 0x00, 0x00, 0x01, 0xAA, 0xBB, 0xCC, 0xDD}
	pkt = append(pkt, ext...)
	payload := []byte{1, 2, 3, 4}
	pkt = append(pkt, payload...)

	h, off, err := parseRTP(pkt)
	if err != nil {
		t.Fatalf("parseRTP: %v", err)
	}
	if h.ssrc != 0xDEADBEEF || h.pt != 96 || h.seq != 7 || h.cc != 2 || !h.ext {
		t.Fatalf("bad header: %+v", h)
	}
	if got := pkt[off:]; string(got) != string(payload) {
		t.Fatalf("payload offset wrong: got % x want % x", got, payload)
	}
}

func TestParseRTPRejects(t *testing.T) {
	if _, _, err := parseRTP([]byte{0, 0, 0}); err == nil {
		t.Fatal("expected error on short packet")
	}
	// version 1 (top two bits 01).
	bad := buildRTP(96, 0, 0, nil)
	bad[0] = 0x40
	if _, _, err := parseRTP(bad); err == nil {
		t.Fatal("expected error on bad version")
	}
}

func TestDecodeIQ(t *testing.T) {
	// S16LE: I=16384 (->0.5), Q=-16384 (->-0.5).
	s16le := make([]byte, 4)
	binary.LittleEndian.PutUint16(s16le[0:2], u16(16384))
	binary.LittleEndian.PutUint16(s16le[2:4], u16(-16384))
	got, err := decodeIQ(s16le, encS16LE)
	if err != nil {
		t.Fatalf("decodeIQ s16le: %v", err)
	}
	if len(got) != 1 || !closeC(got[0], complex(0.5, -0.5), 1e-4) {
		t.Fatalf("s16le got %v", got)
	}

	// S16BE.
	s16be := make([]byte, 4)
	binary.BigEndian.PutUint16(s16be[0:2], u16(16384))
	binary.BigEndian.PutUint16(s16be[2:4], u16(-16384))
	got, _ = decodeIQ(s16be, encS16BE)
	if len(got) != 1 || !closeC(got[0], complex(0.5, -0.5), 1e-4) {
		t.Fatalf("s16be got %v", got)
	}

	// F32LE: pass-through.
	f32le := make([]byte, 8)
	binary.LittleEndian.PutUint32(f32le[0:4], math.Float32bits(0.25))
	binary.LittleEndian.PutUint32(f32le[4:8], math.Float32bits(-0.75))
	got, _ = decodeIQ(f32le, encF32LE)
	if len(got) != 1 || !closeC(got[0], complex(0.25, -0.75), 1e-6) {
		t.Fatalf("f32le got %v", got)
	}

	// Trailing odd byte dropped (3 bytes < one s16 IQ pair).
	got, _ = decodeIQ([]byte{1, 2, 3}, encS16LE)
	if len(got) != 0 {
		t.Fatalf("expected no samples from short payload, got %v", got)
	}

	// Opus is not IQ.
	if _, err := decodeIQ([]byte{0, 0}, encOpus); err == nil {
		t.Fatal("expected error decoding opus as IQ")
	}
}

func closeC(a, b complex64, tol float64) bool {
	return math.Abs(float64(real(a)-real(b))) < tol && math.Abs(float64(imag(a)-imag(b))) < tol
}

// u16 reinterprets a signed 16-bit value as its unsigned wire bits. A direct
// uint16(int16(-n)) conversion of a constant is rejected by the compiler, so
// this non-constant helper is used in tests.
func u16(v int16) uint16 { return uint16(v) }
