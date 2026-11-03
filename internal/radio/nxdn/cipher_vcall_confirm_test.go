package nxdn

import "testing"

// vcallBits lays a VCALL out as the 72-bit SACCH superframe message
// (ParseVCALLSuperframe's field map, built independently of it).
func vcallBits(v VCALLMessage) []byte {
	out := make([]byte, 0, SACCHSuperframeBits)
	put := func(val uint64, n int) {
		for i := n - 1; i >= 0; i-- {
			out = append(out, byte(val>>uint(i))&1)
		}
	}
	put(0, 2)
	put(MessageTypeVCALL, 6)
	put(uint64(v.CallControlOption), 8)
	put(uint64(v.CallType), 3)
	put(uint64(v.VoiceCallOption), 5)
	put(uint64(v.Source), 16)
	put(uint64(v.Destination), 16)
	put(uint64(v.Cipher), 2)
	put(uint64(v.KeyID), 6)
	put(0, 8)
	return out
}

// vcallFrames returns the four traffic frames of one SACCH superframe
// carrying v, starting at frame index first, each with two voice slots.
func vcallFrames(first uint64, v VCALLMessage) []TrafficFrame {
	msg := vcallBits(v)
	frames := make([]TrafficFrame, 4)
	for i := range frames {
		structure := SACCHStructureFirst - i
		info := make([]byte, 0, 26)
		info = append(info, byte(structure>>1)&1, byte(structure)&1)
		info = append(info, 0, 0, 0, 0, 0, 1) // RAN 1
		info = append(info, msg[i*SACCHMessageBits:(i+1)*SACCHMessageBits]...)
		voice := []VCHFrame{
			{Slot: 0, Payload: make([]byte, VoiceSlotBits)},
			{Slot: 1, Payload: make([]byte, VoiceSlotBits)},
		}
		frames[i] = TrafficFrame{Index: first + uint64(i), SACCH: EncodeSACCH(AppendSACCHCRC6(info)), Voice: voice}
	}
	return frames
}

// Issue #1261, 27 Oct log: a CLEAR call (all 104 of its voice frames
// counted clear, by elimination from the chain-ended counters) decoded one
// late VCALL superframe as "scrambler, key 0" — a corrupt SACCH block that
// passed its 6-bit CRC on a call that failed 12 of 52. That single VCALL
// flipped the call to encrypted, published call.encryption, and with
// recordings.skip_encrypted the recorder deleted the clear recording. A
// VCALL cannot really change inside one transmission, so a change now needs
// a second superframe that agrees before it takes effect.
func TestVoiceCipherTrackerIgnoresOneOffVCALLChange(t *testing.T) {
	clear := VCALLMessage{CallType: 1, Source: 8814, Cipher: CipherClear}
	bogus := VCALLMessage{CallType: 1, Source: 8814, Cipher: CipherScrambler, KeyID: 0}
	var seen []VCALLMessage
	tr := &VoiceCipherTracker{Hold: true, OnVCALL: func(v VCALLMessage) { seen = append(seen, v) }}
	var frames []TrafficFrame
	for i, v := range []VCALLMessage{clear, clear, bogus, clear} {
		frames = append(frames, vcallFrames(uint64(4*i), v)...)
	}
	delivered := 0
	for _, f := range frames {
		delivered += len(tr.Frame(f))
	}
	delivered += len(tr.Flush())

	if len(seen) != 1 || seen[0] != clear {
		t.Errorf("OnVCALL saw %+v; want only the first, clear VCALL (the one-off scrambler VCALL must not take effect)", seen)
	}
	if tr.VCALL != clear {
		t.Errorf("call's VCALL is %+v, want it to stay clear", tr.VCALL)
	}
	if tr.Clear != uint64(delivered) || tr.Ciphertext != 0 {
		t.Errorf("clear=%d ciphertext=%d of %d delivered frames; want all clear", tr.Clear, tr.Ciphertext, delivered)
	}
	if tr.VCALLUnconfirmed != 1 {
		t.Errorf("VCALLUnconfirmed = %d, want 1", tr.VCALLUnconfirmed)
	}
}

// A change two superframes in a row agree on is real, and is adopted on the
// second; the first VCALL of a call still takes effect at once, so a keyed
// call starts descrambling as promptly as before.
func TestVoiceCipherTrackerAdoptsConfirmedVCALLChange(t *testing.T) {
	clear := VCALLMessage{CallType: 1, Source: 8814, Cipher: CipherClear}
	scr := VCALLMessage{CallType: 1, Source: 8814, Cipher: CipherScrambler, KeyID: 3}
	var seen []VCALLMessage
	tr := &VoiceCipherTracker{Hold: true, OnVCALL: func(v VCALLMessage) { seen = append(seen, v) }}
	for i, v := range []VCALLMessage{clear, scr, scr} {
		for _, f := range vcallFrames(uint64(4*i), v) {
			tr.Frame(f)
		}
		switch i {
		case 0:
			if len(seen) != 1 || seen[0] != clear {
				t.Fatalf("after the first superframe OnVCALL saw %+v, want the clear VCALL at once", seen)
			}
		case 1:
			if len(seen) != 1 {
				t.Fatalf("one disagreeing superframe was adopted: %+v", seen)
			}
		case 2:
			if len(seen) != 2 || seen[1] != scr {
				t.Fatalf("two agreeing superframes were not adopted: %+v", seen)
			}
		}
	}
	if tr.VCALL != scr {
		t.Errorf("call's VCALL is %+v, want the confirmed scrambler VCALL", tr.VCALL)
	}
}
