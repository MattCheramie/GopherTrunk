//go:build dvsi

package dvsi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestVocoderInterfaceConformance is the compile-time + runtime
// check that *Vocoder satisfies voice.Vocoder. The compile-time
// assertion is at the end of dvsi_enabled.go; this test confirms the
// runtime values match too.
func TestVocoderInterfaceConformance(t *testing.T) {
	v, err := Open(Options{LoopbackOnly: true})
	if err != nil {
		t.Fatalf("Open loopback: %v", err)
	}
	defer v.Close()
	var _ voice.Vocoder = v
	if v.Name() != VocoderName {
		t.Errorf("Name() = %q, want %q", v.Name(), VocoderName)
	}
	if v.FrameSize() != FrameBytes {
		t.Errorf("FrameSize() = %d, want %d", v.FrameSize(), FrameBytes)
	}
}

// TestVocoderRegisteredInRegistry confirms the init() side effect:
// after importing this package under -tags dvsi, the
// voice.DefaultRegistry contains the "dvsi" name. The factory call
// itself will hit ErrNoDevice (no real chip) but the registration is
// what matters.
func TestVocoderRegisteredInRegistry(t *testing.T) {
	names := voice.DefaultRegistry.Names()
	found := false
	for _, n := range names {
		if n == VocoderName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("voice.DefaultRegistry doesn't contain %q; names = %v",
			VocoderName, names)
	}

	// The factory call returns ErrNoDevice on this test host (no
	// hardware) — that's the expected fall-through behaviour the
	// recorder's fallback chain handles.
	_, err := voice.DefaultRegistry.New(VocoderName)
	if !errors.Is(err, ErrNoDevice) {
		t.Errorf("New(%q) err = %v, want ErrNoDevice", VocoderName, err)
	}
}

// TestVocoderLoopbackRoundTrip exercises the full
// AMBE+2-frame -> packet -> transport -> packet -> PCM-decode chain
// against the loopback transport. Confirms FrameSize(),
// Decode([]byte), and Reset()/Close() all wire up correctly.
func TestVocoderLoopbackRoundTrip(t *testing.T) {
	v, err := Open(Options{LoopbackOnly: true})
	if err != nil {
		t.Fatalf("Open loopback: %v", err)
	}
	defer v.Close()

	frame := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	pcm, err := v.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(pcm) != SpeechFrameSamples {
		t.Errorf("len(pcm) = %d, want %d", len(pcm), SpeechFrameSamples)
	}
	// Loopback returns silence — every sample should be 0.
	for i, s := range pcm {
		if s != 0 {
			t.Errorf("pcm[%d] = %d, want 0 (loopback silence)", i, s)
			break
		}
	}

	// Reset is a no-op but should never error.
	v.Reset()

	// Close + double-close should be idempotent.
	if err := v.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if err := v.Close(); err != nil {
		t.Errorf("double Close: %v", err)
	}

	// Decode after Close fails cleanly (no panic).
	if _, err := v.Decode(frame); err == nil {
		t.Error("Decode after Close: want error, got nil")
	}
}

// TestVocoderRejectsWrongFrameSize confirms the Vocoder validates
// frame length before serialising the packet, so callers can't
// accidentally send malformed input to the chip.
func TestVocoderRejectsWrongFrameSize(t *testing.T) {
	v, err := Open(Options{LoopbackOnly: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v.Close()

	if _, err := v.Decode(make([]byte, FrameBytes-1)); err == nil {
		t.Error("Decode with short frame: want error, got nil")
	}
	if _, err := v.Decode(make([]byte, FrameBytes+1)); err == nil {
		t.Error("Decode with long frame: want error, got nil")
	}
}

// scriptedTransport is a deterministic mock that asserts the
// outgoing packet shape and returns a scripted reply. Used to verify
// the wire protocol byte-by-byte without a chip or the loopback's
// silence shortcut.
type scriptedTransport struct {
	t            *testing.T
	expectedOut  []byte
	reply        []byte
	writeCalled  int
	readCalled   int
	closeCalled  int
	readNoPending bool
}

func (s *scriptedTransport) Write(packet []byte) error {
	s.writeCalled++
	if !bytes.Equal(packet, s.expectedOut) {
		s.t.Errorf("Write packet mismatch:\n got: %x\nwant: %x", packet, s.expectedOut)
	}
	return nil
}

func (s *scriptedTransport) Read() ([]byte, error) {
	s.readCalled++
	if s.readNoPending {
		return nil, io.EOF
	}
	return s.reply, nil
}

func (s *scriptedTransport) Close() error {
	s.closeCalled++
	return nil
}

// TestVocoderScriptedExchange asserts the wire-format bytes a Vocoder
// produces match the AMBE-3003 datasheet exactly: a
// PktChannelData packet wrapping the 7-byte AMBE+2 frame, followed by
// reading a PktSpeechData reply and unpacking it as little-endian
// int16 PCM samples.
func TestVocoderScriptedExchange(t *testing.T) {
	frame := []byte{0xAA, 0x55, 0xCA, 0xFE, 0xBA, 0xBE, 0xDE}
	// Build the expected outgoing packet by hand: sync, big-endian
	// length covering type + payload, PktChannelData byte, frame.
	wantOut := []byte{
		PacketSyncByte,
		0x00, 0x08, // length = 1 (type) + 7 (frame) = 8
		byte(PktChannelData),
	}
	wantOut = append(wantOut, frame...)

	// Build the chip's "reply" packet — 320 bytes of nonzero PCM so
	// we know the Vocoder unpacks rather than just zeros.
	pcm := make([]int16, SpeechFrameSamples)
	for i := range pcm {
		pcm[i] = int16(i - 80) // -80, -79, ..., +79
	}
	replyPayload := make([]byte, SpeechFrameBytes)
	for i, s := range pcm {
		binary.LittleEndian.PutUint16(replyPayload[i*2:i*2+2], uint16(s))
	}
	reply := EncodePacket(PktSpeechData, replyPayload)

	scripted := &scriptedTransport{t: t, expectedOut: wantOut, reply: reply}
	v, err := Open(Options{Transport: scripted})
	if err != nil {
		t.Fatalf("Open with scripted transport: %v", err)
	}

	got, err := v.Decode(frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if scripted.writeCalled != 1 {
		t.Errorf("Write called %d times, want 1", scripted.writeCalled)
	}
	if scripted.readCalled != 1 {
		t.Errorf("Read called %d times, want 1", scripted.readCalled)
	}
	for i := range pcm {
		if got[i] != pcm[i] {
			t.Errorf("pcm[%d] = %d, want %d", i, got[i], pcm[i])
			break
		}
	}

	if err := v.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if scripted.closeCalled != 1 {
		t.Errorf("transport Close called %d times, want 1", scripted.closeCalled)
	}
}

// TestVocoderRejectsUnexpectedReply confirms a chip reply that's not
// a PktSpeechData (or has the wrong payload length) surfaces as
// ErrUnexpectedReply rather than corrupting the PCM stream.
func TestVocoderRejectsUnexpectedReply(t *testing.T) {
	frame := make([]byte, FrameBytes)
	cases := []struct {
		name  string
		reply []byte
	}{
		{"ack_instead_of_speech", EncodePacket(PktAck, nil)},
		{"short_speech_payload", EncodePacket(PktSpeechData, make([]byte, SpeechFrameBytes/2))},
		{"control_packet", EncodePacket(PktControl, []byte{0x01, 0x02})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scripted := &scriptedTransport{
				t:           t,
				expectedOut: EncodePacket(PktChannelData, frame),
				reply:       tc.reply,
			}
			v, err := Open(Options{Transport: scripted})
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer v.Close()
			_, err = v.Decode(frame)
			if !errors.Is(err, ErrUnexpectedReply) {
				t.Errorf("err = %v, want ErrUnexpectedReply", err)
			}
		})
	}
}

// TestOpenWithoutHardwareReturnsErrNoDevice confirms the
// no-Transport, no-LoopbackOnly path returns the sentinel error so
// the recorder fallback chain works as documented.
func TestOpenWithoutHardwareReturnsErrNoDevice(t *testing.T) {
	_, err := Open(DefaultOptions())
	if !errors.Is(err, ErrNoDevice) {
		t.Errorf("Open(DefaultOptions()) err = %v, want ErrNoDevice", err)
	}
}
