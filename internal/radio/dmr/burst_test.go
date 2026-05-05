package dmr

import "testing"

func TestBurstSlicesAreContiguous(t *testing.T) {
	var b Burst
	for i := 0; i < BurstDibits; i++ {
		b.Dibits[i] = uint8(i % 4)
	}

	tests := []struct {
		name string
		got  []uint8
		size int
	}{
		{"first half", b.FirstHalf(), HalfPayloadDibits},
		{"slot before", b.SlotTypeBefore(), SlotTypeDibits},
		{"sync", b.Sync(), SyncDibits},
		{"slot after", b.SlotTypeAfter(), SlotTypeDibits},
		{"second half", b.SecondHalf(), HalfPayloadDibits},
	}
	for _, tc := range tests {
		if len(tc.got) != tc.size {
			t.Errorf("%s: len = %d, want %d", tc.name, len(tc.got), tc.size)
		}
	}
	totalDibits := HalfPayloadDibits*2 + SlotTypeDibits*2 + SyncDibits
	if totalDibits != BurstDibits {
		t.Errorf("layout doesn't sum to BurstDibits: %d vs %d", totalDibits, BurstDibits)
	}
}

func TestPayloadBitsIs196Bits(t *testing.T) {
	var b Burst
	for i := 0; i < BurstDibits; i++ {
		b.Dibits[i] = 0b10
	}
	bits := b.PayloadBits()
	if len(bits) != BPTCPayloadBits {
		t.Fatalf("PayloadBits len = %d, want %d", len(bits), BPTCPayloadBits)
	}
	for i := 0; i < BPTCPayloadBits; i += 2 {
		if bits[i] != 1 || bits[i+1] != 0 {
			t.Fatalf("dibit %d/%d unpacked wrong: %d%d", i/2, i, bits[i], bits[i+1])
		}
	}
}

func TestSlotTypeBitsAllConcatenates(t *testing.T) {
	var b Burst
	// Set the 5 slot-type dibits before sync to 0b01,0b10,0b11,0b00,0b01.
	before := []uint8{0b01, 0b10, 0b11, 0b00, 0b01}
	after := []uint8{0b10, 0b00, 0b11, 0b01, 0b10}
	copy(b.Dibits[HalfPayloadDibits:], before)
	copy(b.Dibits[HalfPayloadDibits+SlotTypeDibits+SyncDibits:], after)
	bits := b.SlotTypeBitsAll()
	if len(bits) != 20 {
		t.Fatalf("len = %d, want 20", len(bits))
	}
	want := []byte{
		0, 1, 1, 0, 1, 1, 0, 0, 0, 1, // before
		1, 0, 0, 0, 1, 1, 0, 1, 1, 0, // after
	}
	for i := range want {
		if bits[i] != want[i] {
			t.Errorf("bit %d = %d, want %d", i, bits[i], want[i])
		}
	}
}
