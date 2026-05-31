package m17

import (
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

func encodeAddress(call string) uint64 {
	var addr uint64
	for _, r := range call {
		idx := strings.IndexRune(base40, r)
		if idx < 0 {
			idx = 0
		}
		addr = addr*40 + uint64(idx)
	}
	return addr
}

func TestAddressRoundTrip(t *testing.T) {
	for _, call := range []string{"AB1CDE", "W1AW", "N0CALL", "M17-USA"} {
		addr := encodeAddress(call)
		if got := DecodeAddress(addr); got != call {
			t.Errorf("DecodeAddress(encode(%q)) = %q", call, got)
		}
	}
	if DecodeAddress(broadcastAddr) != "BROADCAST" {
		t.Error("broadcast address not decoded as BROADCAST")
	}
	if DecodeAddress(0) != "" {
		t.Error("zero address should decode empty")
	}
}

// buildLSF assembles a 30-byte LSF with a valid CRC.
func buildLSF(dst, src string, typ uint16) [LSFBytes]byte {
	var b [LSFBytes]byte
	putBE48(b[0:6], encodeAddress(dst))
	putBE48(b[6:12], encodeAddress(src))
	b[12] = byte(typ >> 8)
	b[13] = byte(typ)
	// META left zero.
	crc := CRC16(b[0:28])
	b[28] = byte(crc >> 8)
	b[29] = byte(crc)
	return b
}

func putBE48(dst []byte, v uint64) {
	for i := 5; i >= 0; i-- {
		dst[i] = byte(v & 0xFF)
		v >>= 8
	}
}

// encodeLICHFrameBits builds one stream frame's wire bits: the 16-bit
// stream sync, the Golay-coded 96-bit LICH for chunk cnt, then 272 zero
// payload bits.
func encodeLICHFrameBits(lsf [LSFBytes]byte, cnt int) []byte {
	var frag uint64 // 40-bit LSF fragment
	for i := 0; i < 5; i++ {
		frag = frag<<8 | uint64(lsf[cnt*5+i])
	}
	chunk := (frag << 8) | (uint64(cnt) << 5) // 48 bits: frag(40) | cnt(3) | reserved(5)

	var bits []byte
	push := func(v uint32, n int) {
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, byte((v>>uint(i))&1))
		}
	}
	push(uint32(SyncStream), 16)
	for i := 0; i < 4; i++ {
		data12 := uint16((chunk >> uint(36-i*12)) & 0x0FFF)
		push(GolayEncode24_12Wire(data12), 24)
	}
	for i := 0; i < streamPayloadBits-LICHBits; i++ {
		bits = append(bits, 0)
	}
	return bits
}

// GolayEncode24_12Wire wraps the framing encoder for the test.
func GolayEncode24_12Wire(data uint16) uint32 { return framing.GolayEncode24_12(data) }

func TestDecodeLSFFromLICHStream(t *testing.T) {
	lsf := buildLSF("M17-USA", "AB1CDE", 0x0005) // stream + voice
	d := NewDecoder()

	var got LSF
	var ok bool
	for cnt := 0; cnt < lichChunks; cnt++ {
		for _, b := range encodeLICHFrameBits(lsf, cnt) {
			if l, done := d.Push(b); done {
				got, ok = l, true
			}
		}
	}
	if !ok {
		t.Fatalf("no LSF decoded after 6 LICH chunks (stats %+v)", d.Stats())
	}
	if !got.CRCOK {
		t.Error("CRCOK = false, want true")
	}
	if got.Src != "AB1CDE" {
		t.Errorf("Src = %q, want AB1CDE", got.Src)
	}
	if got.Dst != "M17-USA" {
		t.Errorf("Dst = %q, want M17-USA", got.Dst)
	}
	if !got.Stream {
		t.Error("Stream = false, want true")
	}
	if got.DataMode != 2 {
		t.Errorf("DataMode = %d, want 2 (voice)", got.DataMode)
	}
}

func TestCRC16DetectsCorruption(t *testing.T) {
	lsf := buildLSF("W1AW", "N0CALL", 0)
	if !ParseLSF(lsf[:]).CRCOK {
		t.Fatal("valid LSF: CRCOK false")
	}
	lsf[3] ^= 0x20
	if ParseLSF(lsf[:]).CRCOK {
		t.Error("corrupted LSF: CRCOK true")
	}
}
