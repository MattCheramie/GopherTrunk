package wmbus

import (
	"bytes"
	"math/rand"
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

// Three bit errors among a block's eight least reliable bits are repaired
// (issue #1256, 14 Nov). On the reporter's walks the CRC failures that
// survived the one-or-two-bit search were mostly strong telegrams with
// three to five errors, nearly all of them among the block's least reliable
// bits; the two-bit limit could never fix them.
func TestFramerRepairsThreeWeakBitErrors(t *testing.T) {
	data, _, chips, first := repairFixture(t)
	weak := map[int]float32{}
	for _, bit := range []int{8*13 + 2, 8*15 + 4, 8*20 + 7} {
		chips[first+bit] ^= 1
		weak[first+bit] = 0.1
	}
	got := softFramesFromChips(chips, weak)
	if len(got) != 1 {
		t.Fatalf("%d frames", len(got))
	}
	f := got[0]
	if !f.CRCOK || f.CorrectedBits != 3 || !bytes.Equal(f.Data, data) {
		t.Fatalf("crc=%v corrected=%d data %x, want the sent frame repaired with 3 bits", f.CRCOK, f.CorrectedBits, f.Data)
	}
}

// A third error that is NOT among the eight least reliable bits is left
// alone: triples are searched only there, which is what keeps a garbage
// block's chance of passing near the one-or-two-bit search's own.
func TestFramerDoesNotRepairThreeBitErrorsOutsideTheWeakest(t *testing.T) {
	_, _, chips, first := repairFixture(t)
	weak := map[int]float32{}
	for _, bit := range []int{8*13 + 2, 8*15 + 4, 8*20 + 7} {
		chips[first+bit] ^= 1
	}
	weak[first+8*13+2], weak[first+8*15+4] = 0.1, 0.1
	// Seven decoys weaker than the third error push it to rank 10.
	for k := range 7 {
		weak[first+8*17+k] = 0.2
	}
	weak[first+8*20+7] = 0.5
	got := softFramesFromChips(chips, weak)
	if len(got) != 1 || got[0].CRCOK || got[0].CorrectedBits != 0 {
		t.Fatalf("got %+v, want block 2 failed and nothing repaired", got)
	}
}

// Four bit errors are beyond the repair, however unreliable they are.
func TestFramerDoesNotRepairFourBitErrors(t *testing.T) {
	_, _, chips, first := repairFixture(t)
	weak := map[int]float32{}
	for _, bit := range []int{8*13 + 2, 8*15 + 4, 8*18 + 1, 8*20 + 7} {
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

// A Qundis heat cost allocator from the issue #1256 reporter's 8 Nov walk
// (S5 + RTL-SDR, -28 dBFS), decoded by a fresh receiver: one block carries
// three bit errors among its least reliable bits, which only the three-bit
// search repairs. The same meter decodes cleanly four more times across the
// reporter's walks, so the repaired telegram is a real one.
func TestReceiverRealAirThreeBitRepair(t *testing.T) {
	iq, rate := loadIQWav(t, "testdata/issue1256_qds_28235674_repair3bit.wav")
	type got struct {
		tg        Telegram
		corrected int
	}
	var ok []got
	rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OnFrame: func(f Frame, info FrameInfo) {
		if !f.CRCOK || info.Inverted {
			return
		}
		tg, err := ParseTelegram(f)
		if err != nil {
			t.Error(err)
			return
		}
		ok = append(ok, got{tg, f.CorrectedBits})
	}})
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < len(iq); j += 16384 {
		rx.Process(iq[j:min(j+16384, len(iq))])
	}
	rx.Flush()
	if len(ok) != 1 || ok[0].tg.Manufacturer != "QDS" || ok[0].tg.ID != "28235674" || ok[0].corrected != 3 {
		t.Fatalf("CRC-valid telegrams %+v, want QDS 28235674 repaired with 3 bits", ok)
	}
}

// TestRepairGarbageBlockFalseAcceptRate bounds how often the repair makes a
// garbage block pass its CRC: the price of every pattern it tries. The one-
// and two-bit search alone tries 136 patterns per block and measured 1 in
// 487 on garbage blocks carrying real reliabilities; the triples add 56, for
// 192 (about 1 in 340 for random blocks). A change that widens the search
// must move this bound on purpose.
func TestRepairGarbageBlockFalseAcceptRate(t *testing.T) {
	rng := rand.New(rand.NewSource(1256))
	const trials = 40000
	pass := 0
	block := make([]byte, 18)
	rel := make([]float32, 8*len(block))
	for range trials {
		rng.Read(block)
		for i := range rel {
			rel[i] = rng.Float32()
		}
		if CRC16(block[:16]) == uint16(block[16])<<8|uint16(block[17]) {
			continue
		}
		if repairBlock(block, rel) > 0 {
			pass++
		}
	}
	if rate := float64(pass) / trials; rate > 1.0/250 {
		t.Fatalf("%d of %d garbage blocks repaired (1 in %.0f), want at most 1 in 250", pass, trials, 1/rate)
	} else {
		t.Logf("%d of %d garbage blocks repaired (1 in %.0f)", pass, trials, trials/float64(max(pass, 1)))
	}
}
