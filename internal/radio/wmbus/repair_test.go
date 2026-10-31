package wmbus

import (
	"bytes"
	"testing"
)

// softFramesFromChips frames chips fed with PushSoft, every chip at
// reliability 1 except those in weak, which get the reliability given.
func softFramesFromChips(chips []uint8, weak map[int]float32) []Frame {
	var got []Frame
	fr := NewFramer(func(f Frame) { got = append(got, f) })
	for i, c := range chips {
		rel := float32(1)
		if r, ok := weak[i]; ok {
			rel = r
		}
		fr.PushSoft(c, rel)
	}
	return got
}

// repairFixture is a C1 format A frame of 3 CRC blocks, as chips, with the
// chip index of the first data bit (the L-field's MSB).
func repairFixture(t *testing.T) (data, raw []byte, chips []uint8, first int) {
	t.Helper()
	data = make([]byte, 30)
	for i := range data {
		data[i] = byte(i*37 + 11)
	}
	data[0] = byte(len(data) - 1)
	raw = buildFrame(data, FormatA)
	const lead, preamble = 37, 48
	chips = append(make([]uint8, lead), encodeFrameChips(ModeC1, FormatA, raw, preamble)...)
	// preamble, 16-chip access code, 16-chip C mode format word
	return data, raw, chips, lead + preamble + 32
}

// Two bit errors the receiver was unsure of are repaired; the frame comes
// out exactly as sent and says how many bits it took (issue #1256).
func TestFramerRepairsUnreliableBits(t *testing.T) {
	data, _, chips, first := repairFixture(t)
	weak := map[int]float32{}
	for _, bit := range []int{8*13 + 2, 8*20 + 7} { // two bits of block 2
		chips[first+bit] ^= 1
		weak[first+bit] = 0.1
	}
	got := softFramesFromChips(chips, weak)
	if len(got) != 1 {
		t.Fatalf("%d frames", len(got))
	}
	f := got[0]
	if !f.CRCOK || f.CorrectedBits != 2 || !bytes.Equal(f.Data, data) {
		t.Fatalf("crc=%v corrected=%d data %x, want the sent frame repaired with 2 bits", f.CRCOK, f.CorrectedBits, f.Data)
	}
}

// The same errors on bits the receiver was sure of are left alone: the
// search only flips a block's least reliable bits, which is what keeps a
// garbage block from passing (about 1 in 500 instead of 1 in 6).
func TestFramerDoesNotRepairConfidentBits(t *testing.T) {
	_, _, chips, first := repairFixture(t)
	weak := map[int]float32{}
	for _, bit := range []int{8*13 + 2, 8*20 + 7} {
		chips[first+bit] ^= 1
	}
	// Decoys: 20 other bits of block 2 are less reliable than the errors.
	for k := range 20 {
		weak[first+8*15+k] = 0.1
	}
	got := softFramesFromChips(chips, weak)
	if len(got) != 1 || got[0].CRCOK || got[0].CorrectedBits != 0 {
		t.Fatalf("got %+v, want one frame with block 2 failed and nothing repaired", got)
	}
	if fb := got[0].FailedBlocks; len(fb) != 1 || fb[0] != 2 {
		t.Fatalf("failed blocks %v, want [2]", fb)
	}
}

// Without soft decisions (Push) a failed frame is delivered as received.
func TestFramerRepairNeedsSoftDecisions(t *testing.T) {
	_, _, chips, first := repairFixture(t)
	chips[first+8*13+2] ^= 1
	got := framesFromChips(chips)
	if len(got) != 1 || got[0].CRCOK || got[0].CorrectedBits != 0 {
		t.Fatalf("got %+v, want the frame CRC-failed and unrepaired", got)
	}
}

// Three bit errors in one block are beyond the repair: the CRC's minimum
// distance of 6 means no one- or two-bit pattern can make it pass.
func TestFramerDoesNotRepairThreeBitErrors(t *testing.T) {
	_, _, chips, first := repairFixture(t)
	weak := map[int]float32{}
	for _, bit := range []int{8*13 + 2, 8*15 + 4, 8*20 + 7} {
		chips[first+bit] ^= 1
		weak[first+bit] = 0.1
	}
	got := softFramesFromChips(chips, weak)
	if len(got) != 1 || got[0].CRCOK || got[0].CorrectedBits != 0 {
		t.Fatalf("got %+v, want block 2 failed and nothing repaired", got)
	}
}

// Two of the issue #1256 reporter's saved telegrams (S5 + RTL-SDR at
// -gain 0, the signal in the bottom few bits of the 8-bit converter), each
// decoded by a fresh receiver. Neither was CRC-valid before the receiver
// clipped FM clicks and repaired its least reliable bits: 31567134 now
// decodes clean, 25068683 with one bit repaired.
func TestReceiverRealAirWeakTelegrams(t *testing.T) {
	for _, tc := range []struct{ file, id string }{
		{"testdata/issue1256_qds_31567134_clicks.wav", "31567134"},
		{"testdata/issue1256_qds_25068683_repair1bit.wav", "25068683"},
	} {
		iq, rate := loadIQWav(t, tc.file)
		var ok []Telegram
		rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OnFrame: func(f Frame, info FrameInfo) {
			if !f.CRCOK || info.Inverted {
				return
			}
			tg, err := ParseTelegram(f)
			if err != nil {
				t.Errorf("%s: %v", tc.file, err)
				return
			}
			ok = append(ok, tg)
		}})
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < len(iq); j += 16384 {
			rx.Process(iq[j:min(j+16384, len(iq))])
		}
		rx.Flush()
		if len(ok) != 1 || ok[0].Manufacturer != "QDS" || ok[0].ID != tc.id {
			t.Errorf("%s: CRC-valid telegrams %+v, want QDS %s", tc.file, ok, tc.id)
		}
	}
}

// TestReceiverRealAirClickClip pins the click clip at 4 samples per chip
// (issue #1256): a Qundis heat cost allocator from the reporter's 25 Oct
// walk (S5 + RTL-SDR, the dongle's -50 kHz offset), decoded by a fresh
// receiver. With the 64 kHz clip chosen at 8 samples per chip it fails its
// CRC even with the repair; at 48 kHz it decodes. That
// change took the walk's 945 telegrams from 543 CRC-valid in sequence
// (fewer than v1.3.4's 570) to 582.
func TestReceiverRealAirClickClip(t *testing.T) {
	iq, rate := loadIQWav(t, "testdata/issue1256_qds_29547582_clickclip.wav")
	var ok []Telegram
	rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OnFrame: func(f Frame, info FrameInfo) {
		if !f.CRCOK || info.Inverted {
			return
		}
		tg, err := ParseTelegram(f)
		if err != nil {
			t.Error(err)
			return
		}
		ok = append(ok, tg)
	}})
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < len(iq); j += 16384 {
		rx.Process(iq[j:min(j+16384, len(iq))])
	}
	rx.Flush()
	if len(ok) != 1 || ok[0].Manufacturer != "QDS" || ok[0].ID != "29547582" {
		t.Fatalf("CRC-valid telegrams %+v, want QDS 29547582", ok)
	}
}
