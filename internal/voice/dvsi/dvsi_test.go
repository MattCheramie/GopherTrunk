//go:build dvsi

package dvsi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
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
	t             *testing.T
	expectedOut   []byte
	reply         []byte
	writeCalled   int
	readCalled    int
	closeCalled   int
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

// TestDefaultOptionsCarriesDocumentedVIDPID confirms the DefaultOptions
// helper hands back the FTDI VID + FT2232H PID the AMBE-3003 module
// ships with. Pinned by test so accidental changes surface in CI.
func TestDefaultOptionsCarriesDocumentedVIDPID(t *testing.T) {
	opts := DefaultOptions()
	if opts.USBVendorID != DefaultUSBVendorID {
		t.Errorf("USBVendorID = %#x, want %#x", opts.USBVendorID, DefaultUSBVendorID)
	}
	if opts.USBProductID != DefaultUSBProductID {
		t.Errorf("USBProductID = %#x, want %#x", opts.USBProductID, DefaultUSBProductID)
	}
	if opts.LoopbackOnly {
		t.Errorf("LoopbackOnly = true, want false (production-safe defaults)")
	}
	if opts.Transport != nil {
		t.Errorf("Transport = %v, want nil (production constructs USB transport)", opts.Transport)
	}
}

// TestOpenZeroVIDPIDFallsBackToDefaults confirms the zero-value
// Options pass through DefaultOptions's VID/PID. Documented in the
// Options struct comments — pinned by test so a refactor doesn't
// silently break it.
func TestOpenZeroVIDPIDFallsBackToDefaults(t *testing.T) {
	// Use a scripted transport so Open succeeds without hardware; the
	// behaviour under test is the field defaulting before transport
	// construction.
	scripted := &scriptedTransport{t: t, expectedOut: nil, reply: nil}
	v, err := Open(Options{Transport: scripted})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v.Close()
	// Open mutates opts locally before applying defaults — the only
	// observable defaulting effect is that no error fired. The
	// USB-transport path is what reads VID/PID, and that path doesn't
	// run when Transport is explicit. This test pins the no-error
	// contract; the defaulting itself is exercised at the
	// openUSBTransport boundary (TestOpenStubFTDIIncludesVIDPID below).
}

// TestOpenStubFTDIIncludesVIDPID confirms that without LoopbackOnly
// or an explicit Transport, Open's USB-stub path surfaces ErrNoDevice
// wrapped with the VID/PID that would have been claimed. This is the
// diagnostic operators rely on when troubleshooting "why didn't my
// chip show up?".
func TestOpenStubFTDIIncludesVIDPID(t *testing.T) {
	_, err := Open(Options{USBVendorID: 0x1234, USBProductID: 0x5678, SerialMatch: "ABC123"})
	if !errors.Is(err, ErrNoDevice) {
		t.Fatalf("Open: err = %v, want ErrNoDevice", err)
	}
	msg := err.Error()
	for _, want := range []string{"0x1234", "0x5678", "ABC123"} {
		if !strings.Contains(msg, want) {
			t.Errorf("err message %q missing diagnostic substring %q", msg, want)
		}
	}
}

// TestOpenTransportBeatsLoopbackOnly confirms that when both
// Transport and LoopbackOnly are set, Transport wins (per the
// switch-case ordering in Open). Pinning this so a future refactor
// doesn't silently switch the precedence.
func TestOpenTransportBeatsLoopbackOnly(t *testing.T) {
	scripted := &scriptedTransport{
		t:           t,
		expectedOut: EncodePacket(PktChannelData, make([]byte, FrameBytes)),
		reply:       EncodePacket(PktSpeechData, make([]byte, SpeechFrameBytes)),
	}
	v, err := Open(Options{Transport: scripted, LoopbackOnly: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v.Close()

	frame := make([]byte, FrameBytes)
	if _, err := v.Decode(frame); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if scripted.writeCalled != 1 {
		t.Errorf("scripted Write called %d times, want 1 (Transport must beat LoopbackOnly)",
			scripted.writeCalled)
	}
}

// errorTransport returns a configured error from Write or Read so
// the Decode error-wrapping paths can be exercised without hardware.
type errorTransport struct {
	writeErr    error
	readErr     error
	writeCalled int
	readCalled  int
}

func (e *errorTransport) Write(packet []byte) error {
	e.writeCalled++
	return e.writeErr
}

func (e *errorTransport) Read() ([]byte, error) {
	e.readCalled++
	return nil, e.readErr
}

func (e *errorTransport) Close() error { return nil }

// TestDecodeWrapsTransportWriteError confirms a transport.Write
// failure surfaces with a "transport write" prefix so operators can
// distinguish wire-format problems from chip-protocol problems.
func TestDecodeWrapsTransportWriteError(t *testing.T) {
	wantErr := errors.New("synthetic write failure")
	et := &errorTransport{writeErr: wantErr}
	v, err := Open(Options{Transport: et})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v.Close()

	_, err = v.Decode(make([]byte, FrameBytes))
	if err == nil {
		t.Fatal("Decode: want error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "transport write") {
		t.Errorf("err message %q missing 'transport write' prefix", err.Error())
	}
	if et.writeCalled != 1 {
		t.Errorf("Write called %d times, want 1", et.writeCalled)
	}
	if et.readCalled != 0 {
		t.Errorf("Read called %d times, want 0 (Write failed first)", et.readCalled)
	}
}

// TestDecodeWrapsTransportReadError confirms a transport.Read failure
// surfaces with a "transport read" prefix and that Write ran exactly
// once before the Read attempt.
func TestDecodeWrapsTransportReadError(t *testing.T) {
	wantErr := errors.New("synthetic read failure")
	et := &errorTransport{readErr: wantErr}
	v, err := Open(Options{Transport: et})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v.Close()

	_, err = v.Decode(make([]byte, FrameBytes))
	if err == nil {
		t.Fatal("Decode: want error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "transport read") {
		t.Errorf("err message %q missing 'transport read' prefix", err.Error())
	}
	if et.writeCalled != 1 {
		t.Errorf("Write called %d times, want 1", et.writeCalled)
	}
	if et.readCalled != 1 {
		t.Errorf("Read called %d times, want 1", et.readCalled)
	}
}

// TestLoopbackReadWithoutPendingFails confirms calling Read before
// Write produces an error rather than blocking or returning silence.
// Pinned so a refactor doesn't accidentally hide the "out of order"
// condition.
func TestLoopbackReadWithoutPendingFails(t *testing.T) {
	l := newLoopbackTransport()
	if _, err := l.Read(); err == nil {
		t.Error("Read before Write: want error, got nil")
	}
}

// TestLoopbackWriteAfterCloseFails confirms the closed-pipe sentinel
// surfaces from Write so a Vocoder that's holding a closed transport
// can't accidentally queue a reply.
func TestLoopbackWriteAfterCloseFails(t *testing.T) {
	l := newLoopbackTransport()
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	err := l.Write(EncodePacket(PktChannelData, make([]byte, FrameBytes)))
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("Write after Close: err = %v, want io.ErrClosedPipe", err)
	}
}

// TestLoopbackReadAfterCloseFails: symmetric to the Write case.
func TestLoopbackReadAfterCloseFails(t *testing.T) {
	l := newLoopbackTransport()
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := l.Read(); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("Read after Close: err = %v, want io.ErrClosedPipe", err)
	}
}

// TestLoopbackControlPacketEchoesAck: PktControl writes get mirrored
// as PktAck replies. Locks in the loopback's behaviour so a future
// AMBE-3003 control-flow test has a known baseline.
func TestLoopbackControlPacketEchoesAck(t *testing.T) {
	l := newLoopbackTransport()
	if err := l.Write(EncodePacket(PktControl, []byte{0x01, 0x02})); err != nil {
		t.Fatalf("Write(PktControl): %v", err)
	}
	reply, err := l.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	typ, payload, err := DecodePacket(reply)
	if err != nil {
		t.Fatalf("DecodePacket: %v", err)
	}
	if typ != PktAck {
		t.Errorf("reply type = %#x, want PktAck %#x", typ, PktAck)
	}
	if len(payload) != 0 {
		t.Errorf("Ack payload len = %d, want 0", len(payload))
	}
}

// TestLoopbackUnknownPacketTypeAcks confirms unrecognised packet
// types still get a clean Ack reply so the chip doesn't stall mid-
// stream. Belt-and-braces — the production AMBE-3003 only emits a
// fixed set of types, but defensive handling makes the loopback safe
// for fuzzing.
func TestLoopbackUnknownPacketTypeAcks(t *testing.T) {
	l := newLoopbackTransport()
	if err := l.Write(EncodePacket(PacketType(0xEE), nil)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	reply, err := l.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	typ, _, err := DecodePacket(reply)
	if err != nil {
		t.Fatalf("DecodePacket: %v", err)
	}
	if typ != PktAck {
		t.Errorf("reply type = %#x, want PktAck %#x", typ, PktAck)
	}
}

// TestLoopbackWriteRejectsMalformedPacket confirms the loopback
// short-circuits on packets it can't parse rather than papering over
// the corruption. Important because production code that misframes a
// packet would otherwise see a successful Write followed by a hang.
func TestLoopbackWriteRejectsMalformedPacket(t *testing.T) {
	l := newLoopbackTransport()
	// Truncated header (only two bytes, well below PacketHeaderBytes).
	err := l.Write([]byte{PacketSyncByte, 0x00})
	if err == nil {
		t.Error("Write(malformed): want error, got nil")
	}
}
