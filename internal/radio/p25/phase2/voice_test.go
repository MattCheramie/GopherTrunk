package phase2

import (
	"bytes"
	"errors"
	"testing"
)

// ambePayload builds a VoiceFrameBytes-long AMBE+2 frame with the 7
// padding bits (the low 7 bits of the last byte) cleared, so an
// encode→extract round-trip is bit-exact.
func ambePayload(b ...byte) []byte {
	if len(b) != VoiceFrameBytes {
		panic("ambePayload needs VoiceFrameBytes bytes")
	}
	out := append([]byte(nil), b...)
	out[VoiceFrameBytes-1] &= 0x80 // 49 bits = 6 bytes + 1 bit
	return out
}

func voicePayloads(n int) [][]byte {
	seeds := []byte{0x12, 0x34, 0x56, 0x78}
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		s := seeds[i%len(seeds)]
		out[i] = ambePayload(s, s^0xFF, s+1, s+2, s+3, s+4, s+5)
	}
	return out
}

func TestExtractVoiceFramesRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		slot SlotType
		n    int
	}{
		{"Voice4V", SlotTypeVoice4V, Voice4VFrameCount},
		{"Voice2V", SlotTypeVoice2V, Voice2VFrameCount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := voicePayloads(tc.n)
			sub := EncodeVoiceSubframe(tc.slot, 3, want)
			frame := Subframe{Index: 3, Timeslot: 1, SlotType: tc.slot, Dibits: sub}

			got, errs, err := ExtractVoiceFrames(frame)
			if err != nil {
				t.Fatalf("ExtractVoiceFrames: %v", err)
			}
			if errs != 0 {
				t.Errorf("clean sub-frame corrected %d errors", errs)
			}
			if len(got) != tc.n {
				t.Fatalf("got %d frames, want %d", len(got), tc.n)
			}
			for i := range got {
				if !bytes.Equal(got[i], want[i]) {
					t.Errorf("frame %d = %x, want %x", i, got[i], want[i])
				}
			}
		})
	}
}

func TestExtractVoiceFramesCorrectsBitError(t *testing.T) {
	want := voicePayloads(Voice4VFrameCount)
	sub := EncodeVoiceSubframe(SlotTypeVoice4V, 3, want)
	// The high bit of the first voice frame's first dibit maps into the
	// C0 sub-vector, which AMBE+2 protects with Golay(23,12). Flipping
	// it injects one correctable error.
	sub[VoiceFrameOffset] ^= 0x2

	frame := Subframe{SlotType: SlotTypeVoice4V, Dibits: sub}
	got, errs, err := ExtractVoiceFrames(frame)
	if err != nil {
		t.Fatalf("ExtractVoiceFrames with 1 bit error: %v", err)
	}
	if errs < 1 {
		t.Errorf("expected at least 1 corrected error, got %d", errs)
	}
	if !bytes.Equal(got[0], want[0]) {
		t.Errorf("frame 0 = %x, want %x (FEC should have corrected it)", got[0], want[0])
	}
}

func TestExtractVoiceFramesRejectsMACSubframe(t *testing.T) {
	sub := IdleSubframe()
	WriteISCH(sub, SlotTypeMACSignaling, 0)
	frame := Subframe{SlotType: SlotTypeMACSignaling, Dibits: sub}
	if _, _, err := ExtractVoiceFrames(frame); !errors.Is(err, ErrNotVoiceSubframe) {
		t.Errorf("err = %v, want ErrNotVoiceSubframe", err)
	}
}

func TestSuperframeDecodeToVoiceFrames(t *testing.T) {
	// Build a superframe whose every sub-frame is a 4V voice slot, run
	// it through the full SuperframeDecoder, and extract voice from
	// each — the end-to-end Phase A→B→C path.
	var subs [SubframesPerSuperframe][]uint8
	want := make([][][]byte, SubframesPerSuperframe)
	for i := range subs {
		want[i] = voicePayloads(Voice4VFrameCount)
		subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i), want[i])
	}
	const warmup = 50
	stream := append(make([]uint8, warmup), EncodeSuperframe(subs)...)

	d := NewSuperframeDecoder()
	got := d.Process(stream, 0)
	if len(got) != 1 {
		t.Fatalf("expected 1 superframe, got %d", len(got))
	}
	for i, s := range got[0].Subframes {
		if s.SlotType != SlotTypeVoice4V {
			t.Fatalf("sub-frame %d SlotType = %v, want Voice4V", i, s.SlotType)
		}
		frames, _, err := ExtractVoiceFrames(s)
		if err != nil {
			t.Fatalf("sub-frame %d ExtractVoiceFrames: %v", i, err)
		}
		for j := range frames {
			if !bytes.Equal(frames[j], want[i][j]) {
				t.Errorf("sub-frame %d frame %d mismatch", i, j)
			}
		}
	}
}
