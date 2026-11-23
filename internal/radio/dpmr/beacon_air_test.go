package dpmr

import (
	"bufio"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

// beaconVector is one line of testdata/beacon_dsdcc_vectors.txt.
type beaconVector struct {
	kind   string
	mi     [72]uint8
	sy     [3]uint32
	dibits []uint8
}

func readBeaconVectors(t *testing.T) []beaconVector {
	t.Helper()
	f, err := os.Open("testdata/beacon_dsdcc_vectors.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []beaconVector
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		fs := strings.Fields(line)
		v := beaconVector{kind: fs[0]}
		dib := fs[len(fs)-1]
		for _, c := range dib {
			v.dibits = append(v.dibits, uint8(c-'0'))
		}
		switch v.kind {
		case "MI":
			for i, c := range fs[1] {
				v.mi[i] = uint8(c - '0')
			}
		case "SY":
			for i := 0; i < 3; i++ {
				x, err := strconv.ParseUint(fs[1+i], 16, 32)
				if err != nil {
					t.Fatal(err)
				}
				v.sy[i] = uint32(x)
			}
		}
		out = append(out, v)
	}
	if len(out) != 12 {
		t.Fatalf("%d vectors, want 12", len(out))
	}
	return out
}

// TestBeaconFieldsMatchDsdcc: the committed vectors were decoded by dsdcc's
// own Hamming, interleave, scrambler and CRC routines, compiled outside the
// repository. GopherTrunk must encode the same dibits and decode them back.
func TestBeaconFieldsMatchDsdcc(t *testing.T) {
	for i, v := range readBeaconVectors(t) {
		switch v.kind {
		case "MI":
			if got := EncodeBeaconMI(v.mi); !sameDibits(got, v.dibits) {
				t.Errorf("vector %d: encoder output differs from the reference-checked dibits", i)
			}
			bits, corrected, ok := DecodeBeaconMI(v.dibits)
			if !ok || corrected != 0 || bits != v.mi {
				t.Errorf("vector %d: decode ok=%v corrected=%d equal=%v", i, ok, corrected, bits == v.mi)
			}
		case "SY":
			if got := EncodeBeaconSYScast(v.sy); !sameDibits(got, v.dibits) {
				t.Errorf("vector %d: encoder output differs from the reference-checked dibits", i)
			}
			for f, fd := range DecodeBeaconSYScast(v.dibits) {
				if !fd.OK || fd.Bits != v.sy[f] {
					t.Errorf("vector %d field %d: ok=%v %05X want %05X", i, f+1, fd.OK, fd.Bits, v.sy[f])
				}
			}
		}
	}
}

// One bit error in every Hamming block of a message field is corrected.
// On-air bit k belongs to block k%10, so ten consecutive bits hit ten
// different blocks.
func TestBeaconMICorrectsOneErrorPerBlock(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	var mi [72]uint8
	for i := range mi {
		mi[i] = uint8(r.Intn(2))
	}
	d := EncodeBeaconMI(mi)
	for k := 30; k < 40; k++ {
		d[k/2] ^= 1 << uint(1-k%2)
	}
	bits, corrected, ok := DecodeBeaconMI(d)
	if !ok || corrected != 10 || bits != mi {
		t.Fatalf("ok=%v corrected=%d equal=%v", ok, corrected, bits == mi)
	}
}

// Noise passes the message checks rarely: every block has to be
// correctable and then the CRC8 has to hold.
func TestBeaconMIRejectsNoise(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	const n = 200_000
	pass := 0
	d := make([]uint8, BeaconMIDibits)
	for i := 0; i < n; i++ {
		for j := range d {
			d[j] = uint8(r.Intn(4))
		}
		if _, _, ok := DecodeBeaconMI(d); ok {
			pass++
		}
	}
	// (13/16)^10 of noise fields are correctable, 1/256 of those pass the
	// CRC: about 5e-4.
	if rate := float64(pass) / n; rate > 1e-3 {
		t.Fatalf("%.2e of noise fields pass, want about 5e-4", rate)
	}
}

// A SYScast field whose first copy is lost decodes from its second.
func TestBeaconSYScastFallsBackToSecondCopy(t *testing.T) {
	fields := [3]uint32{0x1ABCD, 0x05555, 0x0F00F}
	d := EncodeBeaconSYScast(fields)
	// Field 2's first copy is blocks 0..2 of its 72 bits; on-air bit k of
	// the field is block k%6. Two errors in block 0 make it uncorrectable.
	for _, k := range []int{0, 6} {
		n := 72 + k
		d[n/2] ^= 1 << uint(1-n%2)
	}
	got := DecodeBeaconSYScast(d)
	for f := range got {
		if !got[f].OK || got[f].Bits != fields[f] {
			t.Errorf("field %d: ok=%v %05X want %05X", f+1, got[f].OK, got[f].Bits, fields[f])
		}
	}
}
