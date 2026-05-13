package dvsi

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// AMBE-3003 packet wire format (per DVSI's public datasheet):
//
//	[0x61] [len_hi] [len_lo] [type] [payload bytes...]
//
//	- 0x61 is the fixed sync byte at the start of every packet.
//	- length is big-endian and covers the type byte + payload (so a
//	  zero-payload packet has length=1).
//	- type identifies the packet's purpose; see PacketType constants.
//
// The framing primitives are unconditionally compiled (no build tag)
// because they're pure data — there's no patent surface in describing
// the chip's serial wire protocol. The Vocoder that actually talks to
// the chip lives under //go:build dvsi.

// PacketSyncByte is the first byte of every AMBE-3003 packet.
const PacketSyncByte byte = 0x61

// PacketHeaderBytes is the byte count of the fixed packet header
// (sync + 2-byte length + 1-byte type) before the payload begins.
const PacketHeaderBytes = 4

// PacketType is the 1-byte type field of an AMBE-3003 packet.
type PacketType byte

const (
	// PktControl carries configuration / status exchanges with the
	// chip (sample-rate setup, channel-format selection, etc.).
	PktControl PacketType = 0x00

	// PktChannelData carries a 49-bit AMBE+2 frame (packed in 7
	// bytes) from the host to the chip for speech synthesis, or
	// from the chip to the host for encode operations.
	PktChannelData PacketType = 0x01

	// PktSpeechData carries 160 samples of 16-bit signed PCM
	// (320 bytes, little-endian) at 8 kHz mono — one 20 ms voice
	// frame.
	PktSpeechData PacketType = 0x02

	// PktAck is the chip's response to a control request.
	PktAck PacketType = 0x06
)

// ErrPacketTooShort is returned by DecodePacket when the input buffer
// is shorter than the fixed header length.
var ErrPacketTooShort = errors.New("dvsi: packet shorter than header")

// ErrPacketBadSync is returned by DecodePacket when the first byte
// isn't PacketSyncByte.
var ErrPacketBadSync = errors.New("dvsi: packet missing sync byte")

// ErrPacketLengthMismatch is returned by DecodePacket when the
// declared length doesn't match the supplied buffer.
var ErrPacketLengthMismatch = errors.New("dvsi: packet length mismatch")

// EncodePacket serialises a single AMBE-3003 packet. The caller writes
// the result directly to the FTDI bulk-OUT endpoint. payload may be
// nil for zero-length control acks.
func EncodePacket(typ PacketType, payload []byte) []byte {
	out := make([]byte, PacketHeaderBytes+len(payload))
	out[0] = PacketSyncByte
	// length covers the type byte + payload (per the datasheet).
	binary.BigEndian.PutUint16(out[1:3], uint16(1+len(payload)))
	out[3] = byte(typ)
	copy(out[PacketHeaderBytes:], payload)
	return out
}

// DecodePacket parses one AMBE-3003 packet from b and returns the
// type + payload slice (which aliases into b — copy before mutating).
// The slice b must contain exactly one packet; trailing bytes are
// rejected with ErrPacketLengthMismatch.
func DecodePacket(b []byte) (PacketType, []byte, error) {
	if len(b) < PacketHeaderBytes {
		return 0, nil, fmt.Errorf("%w: got %d bytes, want >= %d",
			ErrPacketTooShort, len(b), PacketHeaderBytes)
	}
	if b[0] != PacketSyncByte {
		return 0, nil, fmt.Errorf("%w: first byte = %#x, want %#x",
			ErrPacketBadSync, b[0], PacketSyncByte)
	}
	declared := int(binary.BigEndian.Uint16(b[1:3]))
	// declared length covers type + payload, so the full packet is
	// 3 header bytes + declared.
	wantTotal := 3 + declared
	if len(b) != wantTotal {
		return 0, nil, fmt.Errorf("%w: declared %d, buffer %d",
			ErrPacketLengthMismatch, wantTotal, len(b))
	}
	return PacketType(b[3]), b[PacketHeaderBytes:], nil
}

// SplitPackets walks a stream that may contain multiple back-to-back
// AMBE-3003 packets and returns the packets it could parse plus the
// trailing bytes that form an incomplete packet (for buffering across
// the next read). Stops on the first ErrPacketBadSync — the caller is
// expected to resync the stream upstream.
func SplitPackets(stream []byte) (packets [][]byte, leftover []byte, err error) {
	for len(stream) >= PacketHeaderBytes {
		if stream[0] != PacketSyncByte {
			return packets, stream, fmt.Errorf("%w at offset %d",
				ErrPacketBadSync, 0)
		}
		declared := int(binary.BigEndian.Uint16(stream[1:3]))
		total := 3 + declared
		if total > len(stream) {
			// Incomplete trailing packet — caller buffers it.
			return packets, stream, nil
		}
		// Aliased slice into stream; caller copies before mutating.
		packets = append(packets, stream[:total])
		stream = stream[total:]
	}
	return packets, stream, nil
}
