package wmbus

import (
	"bytes"
	"testing"
)

// cFrameChips is a C1 format A telegram of n data bytes as on-air chips,
// with the chip index where its frame (preamble) starts in the result.
func cFrameChips(seed byte, n int) (data []byte, chips []uint8) {
	data = make([]byte, n)
	for i := range data {
		data[i] = byte(i)*seed + 7
	}
	data[0] = byte(n - 1)
	return data, encodeFrameChips(ModeC1, FormatA, buildFrame(data, FormatA), 48)
}

func bankFrames(chips []uint8) []Frame {
	var got []Frame
	b := newFramerBank(func(f Frame) { got = append(got, f) })
	for _, c := range chips {
		b.push(c, 1)
	}
	return got
}

// A telegram that starts while another is being framed is framed too
// (issue #1256, 23 Nov run): the second one takes over the air part-way
// through the first, as a stronger meter does. A single framer, still
// counting the first frame's bytes, never saw the second one's access
// code and lost it.
func TestFramerBankFramesTelegramStartingInsideAnother(t *testing.T) {
	_, first := cFrameChips(37, 60)
	second, secondChips := cFrameChips(91, 30)
	// The second telegram replaces the first from chip 300 on, well inside
	// the first one's data.
	chips := append(append([]uint8(nil), first[:300]...), secondChips...)

	got := bankFrames(chips)
	var found bool
	for _, f := range got {
		if f.CRCOK && bytes.Equal(f.Data, second) {
			found = true
		}
	}
	if !found {
		t.Fatalf("frames %d, none is the second telegram CRC-valid", len(got))
	}

	// The single framer the receiver used before misses it.
	var single []Frame
	fr := NewFramer(func(f Frame) { single = append(single, f) })
	for _, c := range chips {
		fr.PushSoft(c, 1)
	}
	for _, f := range single {
		if f.CRCOK && bytes.Equal(f.Data, second) {
			t.Fatal("a single framer found it too; the test no longer shows the case")
		}
	}
}

// An access code that turns up inside a frame's data by chance starts a
// nested frame whose header fails its CRC. That is not delivered: it is
// noise, not a telegram. The frame around it still is.
func TestFramerBankDropsNestedFrameWithBadHeader(t *testing.T) {
	data, chips := cFrameChips(37, 60)
	// The access code, a valid C mode format word, then a header block
	// whose CRC fails, spliced into the frame's data.
	inner := encodeFrameChips(ModeC1, FormatA, bytes.Repeat([]byte{0x30}, 12), 0)
	at := 200
	spliced := append(append(append([]uint8(nil), chips[:at]...), inner...), chips[at+len(inner):]...)
	// Enough chips after the outer frame for the nested one to complete.
	for i := 0; i < 1200; i++ {
		spliced = append(spliced, uint8(i*i/3)&1)
	}

	got := bankFrames(spliced)
	if len(got) != 1 {
		t.Fatalf("%d frames delivered, want only the outer one", len(got))
	}
	if got[0].CRCOK || got[0].Raw[0] != data[0] {
		t.Fatalf("delivered frame L=%d crc=%v, want the outer frame, failing its CRC", got[0].Raw[0], got[0].CRCOK)
	}
}

// Back-to-back telegrams, and a lone one, frame as they did with one
// framer.
func TestFramerBankSequentialTelegrams(t *testing.T) {
	a, ac := cFrameChips(37, 40)
	b, bc := cFrameChips(91, 25)
	got := bankFrames(append(append(append([]uint8(nil), ac...), make([]uint8, 50)...), bc...))
	if len(got) != 2 || !got[0].CRCOK || !got[1].CRCOK || !bytes.Equal(got[0].Data, a) || !bytes.Equal(got[1].Data, b) {
		t.Fatalf("got %d frames, want the two telegrams in order, CRC-valid", len(got))
	}
}

// Two telegrams from the reporter's 23 Nov run (S5 + RTL-SDR in a parked
// car), each a meter that keyed up while the phone was framing a weaker
// one: the channel power steps up part-way through the weaker telegram,
// which then fails its CRC. v1.4.3 logged only that failure. Both stronger
// meters are real: the run decodes each of them elsewhere too.
func TestReceiverRealAirTelegramInsideAnother(t *testing.T) {
	for _, tc := range []struct{ file, id string }{
		{"testdata/issue1256_collision_eie_02732300.wav", "02732300"},
		{"testdata/issue1256_collision_eie_02742644.wav", "02742644"},
	} {
		iq, rate := loadIQWav(t, tc.file)
		var ids []string
		rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OnFrame: func(f Frame, info FrameInfo) {
			if !f.CRCOK || info.Inverted {
				return
			}
			tg, err := ParseTelegram(f)
			if err != nil {
				t.Errorf("%s: %v", tc.file, err)
				return
			}
			ids = append(ids, tg.Manufacturer+" "+tg.ID)
		}})
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < len(iq); j += 16384 {
			rx.Process(iq[j:min(j+16384, len(iq))])
		}
		rx.Flush()
		if len(ids) != 1 || ids[0] != "EIE "+tc.id {
			t.Errorf("%s: CRC-valid telegrams %v, want EIE %s", tc.file, ids, tc.id)
		}
	}
}
