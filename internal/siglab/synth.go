package siglab

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// SynthOptions configures a synthesis run: which protocol to build, the
// front-end impairments to overlay on the ideal modulator output, and the
// on-disk capture format.
type SynthOptions struct {
	Protocol trunking.Protocol
	Format   SampleFormat
	// Impairments applied to the ideal IQ (SNR, carrier offset, DC spike,
	// I/Q imbalance, multipath). The zero value is a clean capture.
	Impairments demod.Impairments
	// Modulation optionally overrides the fixture's default modulator. Empty
	// uses the per-protocol default. For P25 Phase 1, "lsm" (or "cqpsk")
	// selects the linear π/4-DQPSK (LSM) waveform instead of C4FM, so a
	// capture can exercise the linear CQPSK demod path (issue #492).
	Modulation string
}

// Synthesize builds a known-good (optionally impaired) capture for a
// protocol and returns the IQ plus the Metadata describing how to decode and
// grade it. Returns an error when no synthesis fixture is registered for the
// protocol (see Fixtures for the supported set).
func Synthesize(opts SynthOptions) ([]complex64, *Metadata, error) {
	fx, ok := fixtures[opts.Protocol]
	if !ok {
		return nil, nil, fmt.Errorf("siglab: no synthesis fixture for protocol %s (have: %v)", opts.Protocol, Fixtures())
	}
	modulate := fx.modulate
	if opts.Modulation != "" {
		m, err := modulatorFor(opts.Protocol, opts.Modulation)
		if err != nil {
			return nil, nil, err
		}
		modulate = m
	}
	symbols := fx.build()
	iq := modulate(symbols, fx.sampleRate)
	iq = demod.ApplyImpairments(iq, fx.sampleRate, opts.Impairments)

	meta := &Metadata{
		Protocol:     opts.Protocol.String(),
		Source:       "synthesized",
		SampleRateHz: fx.sampleRate,
		Format:       opts.Format.String(),
		System:       fx.systemKnobs,
		Expected:     fx.expected,
	}
	return iq, meta, nil
}

// modulatorFor resolves a SynthOptions.Modulation override to a modulator.
// Only the combinations a fixture can sensibly re-shape are supported; an
// unknown protocol/modulation pair is an error rather than a silent default.
func modulatorFor(proto trunking.Protocol, mod string) (func([]uint8, float64) []complex64, error) {
	switch proto {
	case trunking.ProtocolP25:
		switch mod {
		case "c4fm":
			return func(d []uint8, sr float64) []complex64 { return demod.ModulateP25C4FM(d, sr, 1800.0) }, nil
		case "lsm", "cqpsk":
			// Linear simulcast modulation: π/4-DQPSK at 10 sps / 48 kHz with the
			// P25 RRC (span 8, α=0.20) and the LSM π/4 constellation rotation —
			// the waveform the linear CQPSK demod is designed for (issue #492).
			return func(d []uint8, _ float64) []complex64 {
				return demod.ModulatePiOver4DQPSK(d, 10, 8, 0.20, math.Pi/4)
			}, nil
		}
	}
	return nil, fmt.Errorf("siglab: no %q modulation for protocol %s", mod, proto)
}

// WriteCapture writes IQ to path in the given format (u8 or f32).
func WriteCapture(path string, iq []complex64, format SampleFormat) error {
	switch format {
	case FormatF32:
		return writeCaptureF32(path, iq)
	default:
		return writeCaptureU8(path, iq)
	}
}

// WriteMetadata writes m to path as JSON (the sidecar `test` auto-discovers).
func WriteMetadata(path string, m *Metadata) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// writeCaptureF32 writes interleaved little-endian float32 IQ (GNU Radio
// cfile). Inverse of decodeF32.
func writeCaptureF32(path string, iq []complex64) error {
	buf := make([]byte, len(iq)*8)
	for i, s := range iq {
		binary.LittleEndian.PutUint32(buf[8*i:], math.Float32bits(real(s)))
		binary.LittleEndian.PutUint32(buf[8*i+4:], math.Float32bits(imag(s)))
	}
	return os.WriteFile(path, buf, 0o644)
}

// writeCaptureU8 writes rtl_sdr 8-bit unsigned interleaved IQ. Inverse of
// decodeU8: maps [-1,+1] back to [0,255] centred on 127.5, clamped.
func writeCaptureU8(path string, iq []complex64) error {
	buf := make([]byte, len(iq)*2)
	for i, s := range iq {
		buf[2*i] = floatToU8(real(s))
		buf[2*i+1] = floatToU8(imag(s))
	}
	return os.WriteFile(path, buf, 0o644)
}

func floatToU8(v float32) byte {
	x := float64(v)*127.5 + 127.5
	if x < 0 {
		x = 0
	}
	if x > 255 {
		x = 255
	}
	return byte(math.Round(x))
}
