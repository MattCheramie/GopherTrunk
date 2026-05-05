package voice

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Vocoder is implemented by every voice decoder GopherTrunk can use:
// pure-Go IMBE, the build-tagged mbelib AMBE+2 wrapper, the future DVSI
// hardware backend, and the NullVocoder used when no decoder is
// available.
//
// Decode consumes one compressed frame and returns 16-bit PCM samples at
// 8 kHz mono (one frame is 20 ms = 160 samples for IMBE/AMBE+2). Decoders
// that need internal state across frames (most vocoders do) keep it on
// the implementation; they are NOT safe for concurrent calls on the same
// instance.
type Vocoder interface {
	Name() string
	FrameSize() int // input bytes per frame
	Decode(frame []byte) ([]int16, error)
	Reset()
	Close() error
}

// Registry holds the set of vocoders the running daemon has linked in.
// Drivers register from init(); callers fetch by name from config.
type Registry struct {
	mu sync.RWMutex
	v  map[string]VocoderFactory
}

// VocoderFactory builds a fresh vocoder instance per call. We allocate
// one per call so vocoders with internal state don't bleed between calls.
type VocoderFactory func() (Vocoder, error)

// DefaultRegistry is process-global; init() in subpackages registers
// here.
var DefaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{v: make(map[string]VocoderFactory)}
}

// Register adds (or replaces) a factory by name.
func (r *Registry) Register(name string, f VocoderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.v[name] = f
}

// Names returns every registered vocoder name, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.v))
	for k := range r.v {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// New returns a fresh instance of the named vocoder.
func (r *Registry) New(name string) (Vocoder, error) {
	r.mu.RLock()
	f, ok := r.v[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("voice: unknown vocoder %q", name)
	}
	return f()
}

// ErrNoVocoder is returned by recorders when a CallStart references a
// vocoder name that isn't registered in the build.
var ErrNoVocoder = errors.New("voice: no vocoder registered for that name")

// NullVocoder produces silence. It's the default when no IMBE / AMBE+2
// decoder is available, and it is always safe to use because it doesn't
// touch any patented algorithm.
type NullVocoder struct {
	frameSize  int
	samplesOut int
}

const (
	// IMBE / AMBE+2 frame parameters at 8 kHz output.
	frameDurationMs = 20
	pcmHzDefault    = 8000
)

// NewNullVocoder returns a silent vocoder with the supplied frame size
// (in bytes). Output is 8 kHz / 20 ms / 160 samples per frame regardless.
func NewNullVocoder(frameSize int) *NullVocoder {
	return &NullVocoder{frameSize: frameSize, samplesOut: pcmHzDefault * frameDurationMs / 1000}
}

func (n *NullVocoder) Name() string                          { return "null" }
func (n *NullVocoder) FrameSize() int                        { return n.frameSize }
func (n *NullVocoder) Decode(_ []byte) ([]int16, error)      { return make([]int16, n.samplesOut), nil }
func (n *NullVocoder) Reset()                                {}
func (n *NullVocoder) Close() error                          { return nil }

func init() {
	DefaultRegistry.Register("null", func() (Vocoder, error) { return NewNullVocoder(11), nil })
}
