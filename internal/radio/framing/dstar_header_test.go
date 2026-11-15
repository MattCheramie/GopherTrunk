package framing

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

type dstarHeaderVector struct {
	name   string
	header []byte
	onAir  []byte // sync + 4 zero bits + 660 FEC bits, transmission order
}

// dstarMMDVMSyncPrefix is what precedes the FEC section in every vector:
// 0xEA 0xA6 sent LSB first, then four zero bits.
const dstarMMDVMSyncPrefix = "0101011101100101" + "0000"

func loadDStarHeaderVectors(t *testing.T) []dstarHeaderVector {
	t.Helper()
	f, err := os.Open("testdata/dstar_header_mmdvm.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []dstarHeaderVector
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4096), 4096)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			t.Fatalf("bad vector line %q", line)
		}
		h, err := hex.DecodeString(fields[1])
		if err != nil || len(h) != 41 {
			t.Fatalf("vector %s: bad header hex", fields[0])
		}
		v := dstarHeaderVector{name: fields[0], header: h}
		for _, c := range fields[2] {
			v.onAir = append(v.onAir, byte(c-'0'))
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		t.Fatal("no vectors")
	}
	return out
}

func bitString(b []byte) string {
	var sb strings.Builder
	for _, x := range b {
		sb.WriteByte('0' + x)
	}
	return sb.String()
}

// The header bursts MMDVM's own encoder produced decode to their header
// bytes with no corrections, and GT's encoder reproduces them bit for
// bit. The old invented chain (K=5, PN15, 22×30) fails both.
func TestDStarHeaderFECMatchesMMDVMEncoder(t *testing.T) {
	for _, v := range loadDStarHeaderVectors(t) {
		if len(v.onAir) != len(dstarMMDVMSyncPrefix)+DStarHeaderChannelBits {
			t.Fatalf("%s: %d on-air bits", v.name, len(v.onAir))
		}
		if got := bitString(v.onAir[:len(dstarMMDVMSyncPrefix)]); got != dstarMMDVMSyncPrefix {
			t.Fatalf("%s: vector does not open with the sync prefix: %s", v.name, got)
		}
		fec := v.onAir[len(dstarMMDVMSyncPrefix):]
		got, metric, ok := DecodeDStarHeaderFECErrors(fec)
		if !ok || !bytes.Equal(got, v.header) || metric != 0 {
			t.Errorf("%s: decode = % x (metric %d), want % x", v.name, got, metric, v.header)
		}
		if enc := EncodeDStarHeaderFEC(v.header); !bytes.Equal(enc, fec) {
			t.Errorf("%s: GT's encoder differs from MMDVM's", v.name)
		}
	}
}

// A burst of scattered errors is corrected: the interleaver spreads a
// run of errors across the code, and the code corrects isolated ones.
func TestDStarHeaderFECCorrectsErrors(t *testing.T) {
	for _, v := range loadDStarHeaderVectors(t) {
		fec := append([]byte(nil), v.onAir[len(dstarMMDVMSyncPrefix):]...)
		// A 6-bit burst and three scattered errors.
		for _, k := range []int{100, 101, 102, 103, 104, 105, 300, 450, 600} {
			fec[k] ^= 1
		}
		got, metric, _ := DecodeDStarHeaderFECErrors(fec)
		if !bytes.Equal(got, v.header) {
			t.Errorf("%s: not corrected: % x", v.name, got)
		}
		if metric != 9 {
			t.Errorf("%s: metric %d, want 9", v.name, metric)
		}
	}
}

// The scrambling sequence is x⁷+x⁴+1 from the all-ones state, period
// 127, and opens with the slow-data scrambler bytes 0x70 0x4F 0x93
// (LSB first) that MMDVMHost and DSD XOR onto a DV frame's slow data.
func TestDStarPN(t *testing.T) {
	pn := DStarPN(127 + 24)
	want := []byte{0x70, 0x4F, 0x93}
	for i, w := range want {
		var b byte
		for k := 0; k < 8; k++ {
			b |= pn[8*i+k] << uint(k)
		}
		if b != w {
			t.Errorf("PN byte %d = %#02x, want %#02x", i, b, w)
		}
	}
	if !bytes.Equal(pn[127:], pn[:24]) {
		t.Error("PN period is not 127")
	}
}

func TestDStarHeaderFECRejectsWrongLength(t *testing.T) {
	if EncodeDStarHeaderFEC(make([]byte, 40)) != nil {
		t.Error("encoder accepted 40 bytes")
	}
	if _, ok := DecodeDStarHeaderFEC(make([]byte, 659)); ok {
		t.Error("decoder accepted 659 bits")
	}
}
