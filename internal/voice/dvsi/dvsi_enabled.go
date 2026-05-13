//go:build dvsi

package dvsi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// FrameBytes is the input byte count per AMBE+2 frame, matching
// internal/voice/ambe2.FrameBytes (49 bits packed into 7 bytes with
// 7 padding bits). The DVSI Vocoder.FrameSize() returns this so
// recorders can stride raw-frame inputs uniformly across the pure-Go
// and hardware backends.
const FrameBytes = 7

// SpeechFrameSamples is the per-frame PCM sample count at 8 kHz mono
// (20 ms × 8 kHz). The AMBE-3003 chip's PktSpeechData payload is
// 2 × SpeechFrameSamples bytes (320 bytes, little-endian int16).
const SpeechFrameSamples = 160

// SpeechFrameBytes is the on-wire byte count of a PktSpeechData
// payload.
const SpeechFrameBytes = 2 * SpeechFrameSamples

// Default USB identifiers for the DVSI USB-3000 (FTDI FT2232H-based).
// Operators with custom-VID modules override via Options.
const (
	DefaultUSBVendorID  uint16 = 0x0403 // FTDI
	DefaultUSBProductID uint16 = 0x6010 // FT2232H Dual UART/FIFO
)

// ErrNoDevice is returned by Open when no FTDI device matching the
// configured VID/PID is enumerated. Recorders fall back to the
// operator's configured non-DVSI vocoder when this surfaces.
var ErrNoDevice = errors.New("dvsi: no FTDI device matching VID/PID")

// ErrUnexpectedReply is returned by Decode when the chip's response
// isn't a PktSpeechData of the expected length.
var ErrUnexpectedReply = errors.New("dvsi: unexpected reply from chip")

// Transport is the byte-stream abstraction the Vocoder uses to talk
// to the AMBE-3003 chip. The real implementation (usbTransport)
// wraps an FTDI bulk-IN/bulk-OUT pair; tests use mockTransport for
// scripted exchanges and loopbackTransport for software-only CI.
//
// Write writes one fully-framed packet (PacketSyncByte + length +
// type + payload, as EncodePacket produces). Read returns the next
// fully-framed packet, allocating fresh.
type Transport interface {
	Write(packet []byte) error
	Read() ([]byte, error)
	Close() error
}

// Options configures a Vocoder. Zero-valued fields fall back to the
// Default* constants. Most operators leave LoopbackOnly false and set
// nothing else; integration tests with hardware override the USB
// identifiers; CI runs with LoopbackOnly = true.
type Options struct {
	// USBVendorID / USBProductID select the FTDI device to claim.
	// 0 falls back to DefaultUSBVendorID / DefaultUSBProductID.
	USBVendorID  uint16
	USBProductID uint16

	// SerialMatch, when non-empty, restricts enumeration to FTDI
	// devices whose iSerialNumber descriptor matches exactly.
	// Useful on multi-chip hosts.
	SerialMatch string

	// Transport overrides the default USB transport. Intended for
	// tests; production code leaves this nil and Open constructs a
	// real usbTransport (or returns ErrNoDevice). When set, the
	// supplied Transport's lifetime is owned by the returned
	// Vocoder — Close shuts it down.
	Transport Transport

	// LoopbackOnly switches Open to construct a software-loopback
	// Transport that synthesises silent PktSpeechData responses to
	// every PktChannelData request. Exercises packet framing + the
	// Vocoder state machine + the voice.Vocoder interface contract
	// in CI without a real chip. NEVER set this in production —
	// every call decodes to silence.
	LoopbackOnly bool
}

// DefaultOptions returns Options with VID/PID set to the
// DVSI USB-3000 defaults and LoopbackOnly off.
func DefaultOptions() Options {
	return Options{
		USBVendorID:  DefaultUSBVendorID,
		USBProductID: DefaultUSBProductID,
	}
}

// Vocoder is the DVSI hardware backend. Implements voice.Vocoder.
type Vocoder struct {
	t      Transport
	closed bool
}

// Open constructs a Vocoder ready to serve Decode calls. Returns
// ErrNoDevice (wrapped) when no FTDI device matching opts is
// enumerated, unless LoopbackOnly is true or opts.Transport is
// non-nil.
func Open(opts Options) (*Vocoder, error) {
	if opts.USBVendorID == 0 {
		opts.USBVendorID = DefaultUSBVendorID
	}
	if opts.USBProductID == 0 {
		opts.USBProductID = DefaultUSBProductID
	}
	var t Transport
	switch {
	case opts.Transport != nil:
		t = opts.Transport
	case opts.LoopbackOnly:
		t = newLoopbackTransport()
	default:
		usb, err := openUSBTransport(opts)
		if err != nil {
			return nil, err
		}
		t = usb
	}
	return &Vocoder{t: t}, nil
}

// Name returns "dvsi", matching the registry key.
func (v *Vocoder) Name() string { return VocoderName }

// FrameSize returns the per-frame input byte count (7).
func (v *Vocoder) FrameSize() int { return FrameBytes }

// Decode sends frame as a PktChannelData packet, reads the chip's
// PktSpeechData response, and returns SpeechFrameSamples of 16-bit
// little-endian PCM.
func (v *Vocoder) Decode(frame []byte) ([]int16, error) {
	if v.closed {
		return nil, errors.New("dvsi: Decode after Close")
	}
	if len(frame) != FrameBytes {
		return nil, fmt.Errorf("dvsi: frame must be %d bytes, got %d", FrameBytes, len(frame))
	}
	if err := v.t.Write(EncodePacket(PktChannelData, frame)); err != nil {
		return nil, fmt.Errorf("dvsi: transport write: %w", err)
	}
	reply, err := v.t.Read()
	if err != nil {
		return nil, fmt.Errorf("dvsi: transport read: %w", err)
	}
	typ, payload, err := DecodePacket(reply)
	if err != nil {
		return nil, fmt.Errorf("dvsi: decode reply: %w", err)
	}
	if typ != PktSpeechData {
		return nil, fmt.Errorf("%w: got type %#x, want PktSpeechData %#x",
			ErrUnexpectedReply, typ, PktSpeechData)
	}
	if len(payload) != SpeechFrameBytes {
		return nil, fmt.Errorf("%w: payload %d bytes, want %d",
			ErrUnexpectedReply, len(payload), SpeechFrameBytes)
	}
	samples := make([]int16, SpeechFrameSamples)
	for i := 0; i < SpeechFrameSamples; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(payload[i*2 : i*2+2]))
	}
	return samples, nil
}

// Reset is a no-op for the DVSI backend — the chip carries no
// across-call state once a PktSpeechData has been returned.
func (v *Vocoder) Reset() {}

// Close releases the underlying transport. Idempotent.
func (v *Vocoder) Close() error {
	if v.closed {
		return nil
	}
	v.closed = true
	if v.t == nil {
		return nil
	}
	return v.t.Close()
}

// Compile-time check that Vocoder satisfies voice.Vocoder.
var _ voice.Vocoder = (*Vocoder)(nil)

func init() {
	voice.DefaultRegistry.Register(VocoderName, func() (voice.Vocoder, error) {
		return Open(DefaultOptions())
	})
}

// loopbackTransport synthesises silent PktSpeechData responses to
// every PktChannelData request. Exercises the full packet roundtrip
// in CI without a real chip. NEVER use outside tests.
type loopbackTransport struct {
	pending []byte // staged reply for the next Read call
	closed  bool
}

func newLoopbackTransport() *loopbackTransport { return &loopbackTransport{} }

func (l *loopbackTransport) Write(packet []byte) error {
	if l.closed {
		return io.ErrClosedPipe
	}
	typ, _, err := DecodePacket(packet)
	if err != nil {
		return err
	}
	switch typ {
	case PktChannelData:
		// Stage a 320-byte zero-PCM SpeechData reply.
		silence := make([]byte, SpeechFrameBytes)
		l.pending = EncodePacket(PktSpeechData, silence)
	case PktControl:
		// Mirror as Ack.
		l.pending = EncodePacket(PktAck, nil)
	default:
		// Unknown packet types: ack and move on.
		l.pending = EncodePacket(PktAck, nil)
	}
	return nil
}

func (l *loopbackTransport) Read() ([]byte, error) {
	if l.closed {
		return nil, io.ErrClosedPipe
	}
	if l.pending == nil {
		return nil, errors.New("dvsi loopback: Read with no pending reply")
	}
	out := l.pending
	l.pending = nil
	return out, nil
}

func (l *loopbackTransport) Close() error {
	l.closed = true
	return nil
}

// openUSBTransport constructs a real USB transport against an FTDI
// device matching opts.USBVendorID / USBProductID. The actual FTDI
// MPSSE / bulk-endpoint plumbing lands in a follow-up — the public
// USB transport interface is in place, and operators with hardware
// can replace this stub. Today this returns ErrNoDevice
// unconditionally so the recorder's fallback chain takes over and
// the build still links cleanly under -tags dvsi.
func openUSBTransport(opts Options) (Transport, error) {
	return nil, fmt.Errorf("%w: USB transport stub — hardware integration follows in a separate PR (VID=%#04x PID=%#04x serial=%q)",
		ErrNoDevice, opts.USBVendorID, opts.USBProductID, opts.SerialMatch)
}
