package ambe2

import (
	"fmt"
	"math/rand"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// Frame parameters per AMBE+2 2400 bps. Every frame carries 49
// information bits over 20 ms of audio at 8 kHz mono.
const (
	// InfoBits is the per-frame information-bit count after the
	// upstream protocol layer (P25 P2 / DMR / NXDN) has applied
	// its FEC and produced the bare AMBE+2 information bits.
	InfoBits = 49

	// FrameBytes is the byte count callers pass to Decode: 49
	// bits round up to 7 bytes with 7 unused trailing bits.
	// Matches what the libmbe wrapper accepted so upstream callers
	// don't need to repack.
	FrameBytes = 7

	// VocoderName is the registry key the daemon resolves at
	// startup. Same name libmbe registered under so existing
	// configs work without change.
	VocoderName = "ambe2"
)

// Decoder is the pure-Go AMBE+2 2400 decoder. Mirrors the imbe
// Decoder shape: one mbe.SynthState (cross-frame log2(Ml)
// prediction + voiced phase + amp memory + §6.4 OA tail), one
// math/rand source for the unvoiced excitation noise, one
// *mbe.AGC, and a one-frame cache of the last-good params for
// the frame-repeat path — all per-call so concurrent calls on
// different decoders don't share state.
//
// Decode currently emits silence; PR-D plugs in parameter
// unpacking and PR-E wires the shared mbe synthesis pipeline.
// The recorder + call pipeline can connect to this decoder
// today and start receiving real audio for free as the later
// pieces land.
type Decoder struct {
	state mbe.SynthState
	rng   *rand.Rand
	agc   *mbe.AGC
}

// New returns a fresh Decoder. The unvoiced-excitation noise
// source is seeded from a fixed default so two decoders
// constructed via New() produce byte-identical output for the
// same frame stream (useful for tests + reproducibility).
// Production callers wanting genuinely-random noise across runs
// should use NewWithSeed with a time-derived seed.
func New() *Decoder {
	return NewWithSeed(0)
}

// NewWithSeed constructs a Decoder with an explicit seed for the
// internal noise source. Lets tests pin output across runs and
// lets production callers spread noise across decoders so two
// parallel calls don't share the same noise stream. AGC parameters
// use mbe.DefaultAGCConfig.
func NewWithSeed(seed int64) *Decoder {
	return NewWithConfig(seed, mbe.DefaultAGCConfig())
}

// NewWithConfig constructs a Decoder with an explicit noise seed +
// AGC configuration. Zero-value fields in cfg fall back to
// mbe.DefaultAGCConfig values, so callers can override only the
// parameters they care about. Mirrors imbe.NewWithConfig.
func NewWithConfig(seed int64, cfg mbe.AGCConfig) *Decoder {
	return &Decoder{
		rng: rand.New(rand.NewSource(seed)),
		agc: mbe.NewAGC(cfg),
	}
}

// Name returns the registry key. Matches VocoderName.
func (d *Decoder) Name() string { return VocoderName }

// FrameSize returns the per-frame input byte count (7 bytes / 49
// information bits with 7 trailing padding bits).
func (d *Decoder) FrameSize() int { return FrameBytes }

// Decode reads 49 information bits from frame and returns 160
// int16 PCM samples at 8 kHz. Currently emits silence: the
// parameter unpack (PR-D) and synthesis wire-up (PR-E) land in
// follow-up PRs. Validates the frame length so callers wiring up
// today get a clear error on a malformed frame and the contract
// stays stable when synthesis lands.
func (d *Decoder) Decode(frame []byte) ([]int16, error) {
	if len(frame) != FrameBytes {
		return nil, fmt.Errorf("ambe2: frame must be %d bytes (49 bits + 7 padding), got %d", FrameBytes, len(frame))
	}
	return make([]int16, mbe.SamplesPerFrame), nil
}

// Reset clears all per-call synthesis state — the cross-frame
// log-amplitude prediction history, the voiced harmonic phase +
// amplitude memory, the §6.4 overlap-add tail, and the AGC
// envelope. Callers invoke it on stream re-sync (e.g., a
// frame-loss event from the upstream P25 P2 / DMR / NXDN
// decoder) so the next frame starts from a clean baseline.
func (d *Decoder) Reset() {
	d.state.Reset()
	d.agc.Reset()
}

// Close releases any resources held by the decoder. The pure-Go
// implementation holds none, so this is always a no-op.
func (d *Decoder) Close() error { return nil }

// Compile-time check that Decoder satisfies voice.Vocoder.
var _ voice.Vocoder = (*Decoder)(nil)

func init() {
	voice.DefaultRegistry.Register(VocoderName, func() (voice.Vocoder, error) {
		return New(), nil
	})
}
