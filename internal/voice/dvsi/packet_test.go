package dvsi

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncodeDecodePacketRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		typ     PacketType
		payload []byte
	}{
		{"control_empty", PktControl, nil},
		{"channel_data_7byte_ambe", PktChannelData, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}},
		{"speech_data_8samples", PktSpeechData, bytes.Repeat([]byte{0xAA, 0x55}, 8)},
		{"ack", PktAck, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wire := EncodePacket(tc.typ, tc.payload)
			if wire[0] != PacketSyncByte {
				t.Errorf("encoded sync byte = %#x, want %#x", wire[0], PacketSyncByte)
			}
			gotType, gotPayload, err := DecodePacket(wire)
			if err != nil {
				t.Fatalf("DecodePacket: %v", err)
			}
			if gotType != tc.typ {
				t.Errorf("type = %#x, want %#x", gotType, tc.typ)
			}
			if !bytes.Equal(gotPayload, tc.payload) {
				t.Errorf("payload = %x, want %x", gotPayload, tc.payload)
			}
		})
	}
}

func TestDecodePacketRejectsBadSync(t *testing.T) {
	bad := []byte{0x00, 0x00, 0x01, byte(PktControl)}
	_, _, err := DecodePacket(bad)
	if !errors.Is(err, ErrPacketBadSync) {
		t.Errorf("err = %v, want ErrPacketBadSync", err)
	}
}

func TestDecodePacketRejectsShortBuffer(t *testing.T) {
	short := []byte{PacketSyncByte, 0x00}
	_, _, err := DecodePacket(short)
	if !errors.Is(err, ErrPacketTooShort) {
		t.Errorf("err = %v, want ErrPacketTooShort", err)
	}
}

func TestDecodePacketRejectsLengthMismatch(t *testing.T) {
	// Declared length says 5 bytes (type + 4 payload), but the
	// buffer only has type + 1 payload byte.
	bad := []byte{PacketSyncByte, 0x00, 0x05, byte(PktChannelData), 0xAA}
	_, _, err := DecodePacket(bad)
	if !errors.Is(err, ErrPacketLengthMismatch) {
		t.Errorf("err = %v, want ErrPacketLengthMismatch", err)
	}
}

func TestSplitPacketsBackToBack(t *testing.T) {
	// Concatenate three packets and confirm SplitPackets walks them.
	a := EncodePacket(PktChannelData, []byte{1, 2, 3, 4, 5, 6, 7})
	b := EncodePacket(PktAck, nil)
	c := EncodePacket(PktSpeechData, bytes.Repeat([]byte{0x55}, 320))
	stream := append(append(append([]byte{}, a...), b...), c...)

	packets, leftover, err := SplitPackets(stream)
	if err != nil {
		t.Fatalf("SplitPackets: %v", err)
	}
	if len(leftover) != 0 {
		t.Errorf("leftover = %d bytes, want 0", len(leftover))
	}
	if len(packets) != 3 {
		t.Fatalf("packets = %d, want 3", len(packets))
	}
	for i, want := range [][]byte{a, b, c} {
		if !bytes.Equal(packets[i], want) {
			t.Errorf("packet[%d] mismatch", i)
		}
	}
}

func TestSplitPacketsBuffersIncompleteTrailer(t *testing.T) {
	// Two full packets plus a partial third (header + truncated
	// payload) — SplitPackets returns the two complete packets and
	// the partial as leftover.
	a := EncodePacket(PktChannelData, []byte{1, 2, 3, 4, 5, 6, 7})
	b := EncodePacket(PktAck, nil)
	partial := []byte{PacketSyncByte, 0x00, 0x08, byte(PktSpeechData), 0xAA, 0xBB} // declared 7-byte payload, only 2 supplied
	stream := append(append(append([]byte{}, a...), b...), partial...)

	packets, leftover, err := SplitPackets(stream)
	if err != nil {
		t.Fatalf("SplitPackets: %v", err)
	}
	if len(packets) != 2 {
		t.Errorf("packets = %d, want 2 (third should be in leftover)", len(packets))
	}
	if !bytes.Equal(leftover, partial) {
		t.Errorf("leftover length = %d, want %d (the truncated trailer)",
			len(leftover), len(partial))
	}
}

func TestSplitPacketsResyncOnBadSync(t *testing.T) {
	// First packet OK, second packet starts with garbage —
	// SplitPackets returns the first packet plus the bad-sync error
	// so the caller can decide how to resync.
	a := EncodePacket(PktChannelData, []byte{1, 2, 3, 4, 5, 6, 7})
	bad := []byte{0xFF, 0x00, 0x01, byte(PktAck)}
	stream := append(append([]byte{}, a...), bad...)

	packets, leftover, err := SplitPackets(stream)
	if !errors.Is(err, ErrPacketBadSync) {
		t.Errorf("err = %v, want ErrPacketBadSync", err)
	}
	if len(packets) != 1 {
		t.Errorf("packets before bad sync = %d, want 1", len(packets))
	}
	if !bytes.Equal(leftover, bad) {
		t.Error("leftover should preserve the bad-sync bytes for upstream resync")
	}
}
