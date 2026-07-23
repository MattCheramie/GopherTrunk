package acelp

import "testing"

// TestBinIntRoundTrip checks MSB-first bit packing/unpacking round-trips and
// that binToInt reads the most-significant bit first.
func TestBinIntRoundTrip(t *testing.T) {
	if got := binToInt([]byte{1, 0, 1}, 3); got != 5 {
		t.Errorf("binToInt(101) = %d, want 5 (MSB first)", got)
	}
	if got := binToInt([]byte{1, 0, 0, 0}, 4); got != 8 {
		t.Errorf("binToInt(1000) = %d, want 8", got)
	}
	for _, w := range []int{1, 6, 8, 9, 14} {
		max := int16(1<<uint(w) - 1)
		for _, v := range []int16{0, 1, max, max / 2, max / 3} {
			dst := make([]byte, w)
			intToBin(dst, v, w)
			if got := binToInt(dst, w); got != v {
				t.Errorf("round-trip w=%d v=%d -> %d", w, v, got)
			}
		}
	}
}

// TestBits2PrmRoundTrip builds a 137-bit frame from known indices, unpacks it,
// and checks every field survives; also checks total bit width is 137.
func TestBits2PrmRoundTrip(t *testing.T) {
	total := 0
	for _, w := range bitWidths {
		total += w
	}
	if total != speechFrameBits {
		t.Fatalf("bit widths sum to %d, want %d", total, speechFrameBits)
	}

	var prm [numFields]int16
	for i := 0; i < numParams; i++ {
		// distinct value that fits the field width
		prm[1+i] = int16((i*7 + 1) & (1<<uint(bitWidths[i]) - 1))
	}
	bits := prm2bits(prm)
	if len(bits) != speechFrameBits {
		t.Fatalf("prm2bits length %d, want %d", len(bits), speechFrameBits)
	}
	got := bits2prm(bits, false)
	if got[pBFI] != 0 {
		t.Errorf("BFI should be 0")
	}
	for i := 0; i < numParams; i++ {
		if got[1+i] != prm[1+i] {
			t.Errorf("param %d: got %d, want %d", i, got[1+i], prm[1+i])
		}
	}
	if bits2prm(bits, true)[pBFI] != 1 {
		t.Errorf("BFI flag not set when bfi=true")
	}
}

// TestDLsp334Pristine checks that, when the dequantised set is accepted (the
// ordered path, not the fallback), it dequantises to the raw codebook entries
// at the positions the min-distance correction cannot touch (lsp[0],lsp[1] from
// dico1 and lsp[7..9] from dico3). The three split indices are independent, so
// not every triple yields an ordered set; find the first one that does.
func TestDLsp334Pristine(t *testing.T) {
	lspOld := [lpcOrder]int16{30000, 26000, 21000, 15000, 8000, 0, -8000, -15000, -21000, -26000}
	var a, b, c int
	var lsp [lpcOrder]int16
	found := false
search:
	for a = 0; a < lspDico1Size; a++ {
		for b = 0; b < lspDico2Size; b++ {
			for c = 0; c < lspDico3Size; c++ {
				lsp = dLsp334([3]int16{int16(a), int16(b), int16(c)}, lspOld)
				if lsp != lspOld {
					found = true
					break search
				}
			}
		}
	}
	if !found {
		t.Fatal("no ordered index triple found")
	}
	if lsp[0] != lspDico1[a*3] || lsp[1] != lspDico1[a*3+1] {
		t.Errorf("lsp[0:2] = %d,%d want %d,%d", lsp[0], lsp[1], lspDico1[a*3], lspDico1[a*3+1])
	}
	if lsp[7] != lspDico3[c*4+1] || lsp[8] != lspDico3[c*4+2] || lsp[9] != lspDico3[c*4+3] {
		t.Errorf("lsp[7:10] mismatch with dico3 at index %d", c)
	}
}

// TestDLsp334Ordering checks the invariant D_Lsp334 guarantees: the returned
// LSP set is strictly decreasing in the cosine domain, or exactly the previous
// frame's set (the reference's out-of-order fallback).
func TestDLsp334Ordering(t *testing.T) {
	lspOld := [lpcOrder]int16{30000, 26000, 21000, 15000, 8000, 0, -8000, -15000, -21000, -26000}
	fellBack := 0
	tried := 0
	for a := 0; a < lspDico1Size; a += 17 {
		for b := 0; b < lspDico2Size; b += 53 {
			for c := 0; c < lspDico3Size; c += 53 {
				tried++
				lsp := dLsp334([3]int16{int16(a), int16(b), int16(c)}, lspOld)
				if lsp == lspOld {
					fellBack++
					continue
				}
				for i := 0; i < lpcOrder-1; i++ {
					if lsp[i] <= lsp[i+1] {
						t.Fatalf("idx(%d,%d,%d): lsp not strictly decreasing at %d: %v", a, b, c, i, lsp)
					}
				}
			}
		}
	}
	if tried == 0 {
		t.Fatal("no indices tried")
	}
	t.Logf("dLsp334: %d/%d index triples fell back to old LSPs", fellBack, tried)
}
