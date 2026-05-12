// ALSA backend for the live-audio Player on Linux. Talks to
// libasound2.so.2 directly via github.com/ebitengine/purego —
// no cgo, no libasound2-dev at build time, no pkg-config. The
// runtime dependency (libasound2 itself) ships on every Linux
// desktop / server image that has audio.
//
// We use the high-level alsa-pcm-min API (five functions) so the
// surface area stays small: snd_pcm_open / snd_pcm_set_params /
// snd_pcm_writei / snd_pcm_drain / snd_pcm_close, plus
// snd_pcm_recover for underrun handling and snd_strerror for
// human-readable error messages. ALSA's hw_params machinery is
// powerful but vastly more verbose; set_params covers everything
// we need (channels, rate, format, latency) in one call.
//
// When libasound2.so.2 fails to load (e.g. on a headless Alpine
// container with no audio libs at all), defaultBackendFactory
// returns an error and the Player upstream falls back to a null
// backend — the same code path that runs when audio.enabled is
// false. The daemon keeps recording WAVs and serving the TUI.

//go:build linux

package player

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ALSA stream / format / access constants from <alsa/pcm.h>. These
// are part of the ALSA ABI and have been stable since libasound 1.0.
const (
	sndPCMStreamPlayback    = 0
	sndPCMFormatS16LE       = 2
	sndPCMAccessRWInterleav = 3
	// errno value alsa returns on underrun (-EPIPE). snd_pcm_recover
	// hides the platform-specific number under "silent=1" mode.
	defaultLatencyUs uint32 = 80_000
)

// Function pointers bound at first construction via purego.
// Signatures match libasound's C declarations; uintptr stands in
// for snd_pcm_t* (an opaque handle), unsafe.Pointer for raw buffers,
// and int32 for C's int (ALSA never uses long for return codes
// outside of the snd_pcm_*_t typedefs we don't need here).
var (
	alsaOnce sync.Once
	alsaLib  uintptr
	alsaErr  error

	sndPCMOpen      func(pcm *uintptr, name *byte, stream int32, mode int32) int32
	sndPCMSetParams func(pcm uintptr, format int32, access int32, channels uint32, rate uint32, softResample int32, latency uint32) int32
	sndPCMWritei    func(pcm uintptr, buffer unsafe.Pointer, size uintptr) int64
	sndPCMDrain     func(pcm uintptr) int32
	sndPCMClose     func(pcm uintptr) int32
	sndPCMRecover   func(pcm uintptr, err int32, silent int32) int32
)

// loadALSA dlopens libasound2.so.2 once. Subsequent calls return
// the cached error (if any). Idempotent.
func loadALSA() error {
	alsaOnce.Do(func() {
		// libasound.so.2 is the SONAME — the ABI-stable symlink to
		// the actual shared object. libasound.so (without the .2)
		// is the dev-package symlink and shouldn't be relied on at
		// runtime.
		h, err := purego.Dlopen("libasound.so.2", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			alsaErr = fmt.Errorf("dlopen libasound.so.2: %w", err)
			return
		}
		alsaLib = h
		// RegisterLibFunc panics on a missing symbol — wrap each
		// binding in a recover so a stripped-down libasound (rare
		// but possible) degrades gracefully instead of bringing
		// the whole daemon down.
		defer func() {
			if r := recover(); r != nil {
				alsaErr = fmt.Errorf("symbol resolution failed: %v", r)
			}
		}()
		purego.RegisterLibFunc(&sndPCMOpen, h, "snd_pcm_open")
		purego.RegisterLibFunc(&sndPCMSetParams, h, "snd_pcm_set_params")
		purego.RegisterLibFunc(&sndPCMWritei, h, "snd_pcm_writei")
		purego.RegisterLibFunc(&sndPCMDrain, h, "snd_pcm_drain")
		purego.RegisterLibFunc(&sndPCMClose, h, "snd_pcm_close")
		purego.RegisterLibFunc(&sndPCMRecover, h, "snd_pcm_recover")
	})
	return alsaErr
}

func init() {
	defaultBackendFactory = newALSABackend
}

// alsaBackend implements Backend over a single snd_pcm_t handle in
// blocking write mode. One backend per Player; the Player serialises
// Write calls so we don't need a mutex around the handle.
type alsaBackend struct {
	pcm       uintptr
	closeOnce sync.Once
	closeErr  error
}

func newALSABackend(cfg Config) (Backend, error) {
	if err := loadALSA(); err != nil {
		return nil, err
	}

	deviceName := cfg.Device
	if deviceName == "" || deviceName == "default" {
		deviceName = "default"
	}
	cname := append([]byte(deviceName), 0)

	var pcm uintptr
	if r := sndPCMOpen(&pcm, &cname[0], sndPCMStreamPlayback, 0); r < 0 {
		return nil, fmt.Errorf("alsa: snd_pcm_open(%q): %s", deviceName, alsaErrorString(r))
	}

	latency := defaultLatencyUs
	if cfg.BufferMs > 0 {
		latency = uint32(cfg.BufferMs) * 1000
	}
	rate := cfg.SampleRate
	if rate == 0 {
		rate = 8000
	}
	// soft_resample=1 lets ALSA resample if the hardware can't natively
	// produce our rate — we'd rather have correct-pitch audio at a
	// slight CPU cost than fail outright when the on-board codec
	// only supports 44.1/48 kHz.
	if r := sndPCMSetParams(pcm, sndPCMFormatS16LE, sndPCMAccessRWInterleav, 1, rate, 1, latency); r < 0 {
		sndPCMClose(pcm)
		return nil, fmt.Errorf("alsa: snd_pcm_set_params(rate=%d): %s", rate, alsaErrorString(r))
	}
	return &alsaBackend{pcm: pcm}, nil
}

// Write feeds int16 PCM to ALSA. Blocks until every sample has been
// consumed (PCM buffer flushed). Underruns trigger snd_pcm_recover;
// any other negative return code is returned as an error.
func (b *alsaBackend) Write(samples []int16) error {
	if len(samples) == 0 {
		return nil
	}
	// snd_pcm_writei wants a frame count, not a byte count. We're
	// mono int16 so frames == samples.
	frames := uintptr(len(samples))
	written := uintptr(0)
	for written < frames {
		ptr := unsafe.Pointer(&samples[written])
		r := sndPCMWritei(b.pcm, ptr, frames-written)
		if r < 0 {
			// EPIPE = underrun; EAGAIN / EBUSY also recoverable.
			// snd_pcm_recover with silent=1 covers all three.
			if rr := sndPCMRecover(b.pcm, int32(r), 1); rr < 0 {
				return fmt.Errorf("alsa: snd_pcm_writei: %s", alsaErrorString(int32(r)))
			}
			continue
		}
		written += uintptr(r)
	}
	return nil
}

// Close drains the playback ring and releases the handle. Idempotent.
func (b *alsaBackend) Close() error {
	b.closeOnce.Do(func() {
		// Drain is best-effort — if it fails the close still runs.
		_ = sndPCMDrain(b.pcm)
		if r := sndPCMClose(b.pcm); r < 0 {
			b.closeErr = fmt.Errorf("alsa: snd_pcm_close: %s", alsaErrorString(r))
		}
	})
	return b.closeErr
}

// alsaErrorString turns a negative ALSA return code into a log-
// friendly string. We deliberately don't dereference snd_strerror's
// returned C pointer here — converting a uintptr (which is how
// purego surfaces C pointers) to unsafe.Pointer trips go vet's
// "possible misuse of unsafe.Pointer" check, and the human-readable
// string isn't load-bearing for the daemon's behaviour. Operators
// debugging an audio failure can run `errno` or
// `python3 -c "import errno; print(errno.errorcode[N])"` on the
// numeric code to translate; libasound's strings would have added
// "Broken pipe" / "No such file or directory" style context which
// the bare errno already implies.
func alsaErrorString(code int32) string {
	return fmt.Sprintf("errno=%d", code)
}

// errUnused keeps the linter quiet about `errors` while the package
// matures — alsaErr above flows through fmt.Errorf so a direct
// errors. import isn't strictly required, but it's idiomatic to
// keep it around for future error wrappers.
var _ = errors.New
