// Direct-ioctl ALSA backend — drives /dev/snd/pcmC{card}D{device}p
// straight via SNDRV_PCM_IOCTL_* syscalls. No libasound2.so.2 at
// runtime, no purego dlopen, no cgo. Useful for stripped-down
// container images (distroless, scratch, minimal Alpine) that
// don't ship the userspace ALSA library but do have the kernel
// sound subsystem available.
//
// This is the no-runtime-dep alternative to alsa_linux.go's
// purego dlopen path. Selected by setting audio.device to
// "ioctl" (default card 0 device 0) or "ioctl:hw:C,D" (specific
// card / device). The libasound2 dlopen path stays the default
// because it negotiates format / rate / period sizes with the
// hardware automatically; the ioctl path uses fixed values
// (S16_LE mono at the configured sample rate) so it only works
// when the underlying hardware supports them.
//
// References:
//   - Kernel UAPI: include/uapi/sound/asound.h
//   - ALSA library source (alsa-lib/src/pcm/pcm_hw.c) for the
//     reference state-machine ordering: OPEN → HW_PARAMS →
//     SW_PARAMS → PREPARE → WRITEI → DRAIN/DROP → CLOSE.
//
// Notes on the implementation:
//   - The snd_pcm_hw_params struct is 604 bytes laid out as 3
//     used + 5 reserved 256-bit masks, then 12 used + 9 reserved
//     12-byte intervals, then a tail of bitmasks / info fields.
//     Sizes are platform-independent because the kernel UAPI is
//     defined in fixed-width types.
//   - All ioctls use SYS_IOCTL via syscall.Syscall. We don't pull
//     in golang.org/x/sys/unix as a direct dep — the project's
//     other "purego" backends (RTL-SDR, ALSA-via-dlopen) follow
//     the same convention.

//go:build linux

package player

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Linux UAPI constants from <sound/asound.h>. Stable since
// libasound v1.0; these match every kernel that ships modern
// ALSA support.
const (
	// Mask parameter indices into snd_pcm_hw_params.Masks.
	hwParamAccess    = 0
	hwParamFormat    = 1
	hwParamSubformat = 2

	// Interval parameter indices into snd_pcm_hw_params.Intervals.
	hwParamSampleBits  = 0
	hwParamFrameBits   = 1
	hwParamChannels    = 2
	hwParamRate        = 3
	hwParamPeriodTime  = 4
	hwParamPeriodSize  = 5
	hwParamPeriodBytes = 6
	hwParamPeriods     = 7
	hwParamBufferTime  = 8
	hwParamBufferSize  = 9
	hwParamBufferBytes = 10

	// Enum values for the mask params.
	pcmAccessRWInterleaved = 3 // SNDRV_PCM_ACCESS_RW_INTERLEAVED
	pcmFormatS16LE         = 2 // SNDRV_PCM_FORMAT_S16_LE
	pcmSubformatStd        = 0 // SNDRV_PCM_SUBFORMAT_STD

	// snd_pcm_sw_params SNDRV_PCM_HW_PARAMS_NO_PERIOD_WAKEUP is unused.
	// We just set start_threshold + stop_threshold + avail_min and let
	// every other sw_param default.

	// Magic version the kernel checks at HW_PARAMS time. Pinned to
	// SNDRV_PCM_VERSION as of the v6.x UAPI header set; backwards
	// compatibility means older kernels accept newer versions.
	pcmVersion = 0x00020000 + 13 // 2.0.13
)

// sndMask is the kernel's 256-bit mask, packed into 8 uint32s.
type sndMask struct {
	Bits [8]uint32
}

// sndInterval mirrors struct snd_interval. Min/Max are the bounds;
// Flags encodes openmin/openmax/integer/empty (we only use INTEGER
// and the implicit "min == max for a pinned value" case).
type sndInterval struct {
	Min, Max uint32
	// Bit 0: openmin, bit 1: openmax, bit 2: integer, bit 3: empty.
	Flags uint32
}

// sndPCMHwParams mirrors struct snd_pcm_hw_params. 604 bytes total.
// Layout is fixed by the UAPI; do not reorder.
type sndPCMHwParams struct {
	Flags     uint32
	Masks     [3]sndMask // ACCESS, FORMAT, SUBFORMAT
	Mres      [5]sndMask // reserved masks
	Intervals [12]sndInterval
	Ires      [9]sndInterval // reserved intervals
	Rmask     uint32         // requested mask (set by user, kernel may clear)
	Cmask     uint32         // changed mask (kernel sets on REFINE)
	Info      uint32
	MSBits    uint32
	RateNum   uint32
	RateDen   uint32
	FifoSize  uint64
	Reserved  [64]byte
}

// sndPCMSwParams mirrors struct snd_pcm_sw_params. We only set
// a handful of fields; the rest stay at the zero default which
// matches what alsa-lib's defaults produce.
type sndPCMSwParams struct {
	TstampMode       uint32
	PeriodStep       uint32
	SleepMin         uint32
	AvailMin         uint64
	XferAlign        uint64 // obsolete; kept for layout
	StartThreshold   uint64
	StopThreshold    uint64
	SilenceThreshold uint64
	SilenceSize      uint64
	Boundary         uint64
	Proto            uint32
	TstampType       uint32
	Reserved         [56]byte
}

// sndXferi mirrors struct snd_xferi — the WRITEI / READI payload.
type sndXferi struct {
	Result int64
	Buf    uintptr // pointer to the sample buffer
	Frames uint64
}

// ioctl request numbers. Computed once and cached.
var (
	ioctlHwParams = ioctlRequest(_IOC_READ|_IOC_WRITE, 'A', 0x11, unsafe.Sizeof(sndPCMHwParams{}))
	ioctlSwParams = ioctlRequest(_IOC_READ|_IOC_WRITE, 'A', 0x13, unsafe.Sizeof(sndPCMSwParams{}))
	ioctlPrepare  = ioctlRequest(_IOC_NONE, 'A', 0x40, 0)
	ioctlStart    = ioctlRequest(_IOC_NONE, 'A', 0x42, 0)
	ioctlDrop     = ioctlRequest(_IOC_NONE, 'A', 0x43, 0)
	ioctlDrain    = ioctlRequest(_IOC_NONE, 'A', 0x44, 0)
	ioctlWriteI   = ioctlRequest(_IOC_WRITE, 'A', 0x50, unsafe.Sizeof(sndXferi{}))
)

const (
	_IOC_NONE  = 0
	_IOC_WRITE = 1
	_IOC_READ  = 2

	_IOC_NRSHIFT   = 0
	_IOC_TYPESHIFT = 8
	_IOC_SIZESHIFT = 16
	_IOC_DIRSHIFT  = 30
)

func ioctlRequest(dir uintptr, typ uintptr, nr uintptr, size uintptr) uintptr {
	return (dir << _IOC_DIRSHIFT) |
		(typ << _IOC_TYPESHIFT) |
		(nr << _IOC_NRSHIFT) |
		(size << _IOC_SIZESHIFT)
}

// ioctlBackend implements Backend via direct kernel ALSA ioctls.
type ioctlBackend struct {
	f          *os.File
	rate       uint32
	periodSize uint32

	closeOnce sync.Once
	closeErr  error
}

// newIoctlALSABackend opens /dev/snd/pcmC{card}D{device}p and runs
// the HW_PARAMS / SW_PARAMS / PREPARE handshake so the device is
// ready to accept WRITEI calls. Device spec format: "hw:C,D" or
// just "" (defaults to "hw:0,0").
func newIoctlALSABackend(cfg Config, deviceSpec string) (Backend, error) {
	card, device, err := parseIoctlDevice(deviceSpec)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/dev/snd/pcmC%dD%dp", card, device)
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("ioctl-alsa: open %s: %w", path, err)
	}

	rate := cfg.SampleRate
	if rate == 0 {
		rate = 8000
	}
	// Period size in frames — match the buffer-ms config but cap
	// to keep things sane. 8 kHz × 80 ms = 640 frames; we round
	// up to a power-of-two-ish value the kernel prefers.
	bufMs := cfg.BufferMs
	if bufMs <= 0 {
		bufMs = 80
	}
	periodSize := uint32(int(rate) * bufMs / 1000)
	if periodSize < 64 {
		periodSize = 64
	}
	// Buffer holds 2 periods so the kernel can pipeline.
	bufferSize := periodSize * 2

	b := &ioctlBackend{f: f, rate: rate, periodSize: periodSize}

	if err := b.setHWParams(rate, periodSize, bufferSize); err != nil {
		f.Close()
		return nil, err
	}
	if err := b.setSWParams(periodSize, bufferSize); err != nil {
		f.Close()
		return nil, err
	}
	if err := b.ioctl(ioctlPrepare, 0); err != nil {
		f.Close()
		return nil, fmt.Errorf("ioctl-alsa: PREPARE: %w", err)
	}
	return b, nil
}

// parseIoctlDevice accepts "" or "hw:C,D" forms. Returns the
// numeric card / device. "ioctl" prefix is stripped by the caller
// (newALSABackend) before reaching us.
func parseIoctlDevice(spec string) (card, device uint32, err error) {
	if spec == "" || spec == "default" || spec == "hw:0,0" {
		return 0, 0, nil
	}
	if !strings.HasPrefix(spec, "hw:") {
		return 0, 0, fmt.Errorf("ioctl-alsa: device %q must be empty or \"hw:C,D\"", spec)
	}
	parts := strings.Split(spec[3:], ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("ioctl-alsa: device %q malformed", spec)
	}
	c, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("ioctl-alsa: parse card %q: %w", parts[0], err)
	}
	d, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("ioctl-alsa: parse device %q: %w", parts[1], err)
	}
	return uint32(c), uint32(d), nil
}

// setHWParams pins access mode, sample format, channel count, and
// rate to fixed values (S16_LE / 1 ch / configured rate). The
// kernel sets the rest based on hardware capability; we just
// have to leave a valid wildcard for each.
func (b *ioctlBackend) setHWParams(rate, periodSize, bufferSize uint32) error {
	var p sndPCMHwParams
	// Mask params — start with all bits set so the kernel can
	// pick the one we narrow to.
	for i := range p.Masks {
		for j := range p.Masks[i].Bits {
			p.Masks[i].Bits[j] = 0xFFFFFFFF
		}
	}
	for i := range p.Intervals {
		p.Intervals[i].Max = 0xFFFFFFFF
	}
	// Narrow ACCESS to RW_INTERLEAVED.
	p.Masks[hwParamAccess].Bits = [8]uint32{}
	p.Masks[hwParamAccess].Bits[pcmAccessRWInterleaved/32] = 1 << (pcmAccessRWInterleaved % 32)
	// Narrow FORMAT to S16_LE.
	p.Masks[hwParamFormat].Bits = [8]uint32{}
	p.Masks[hwParamFormat].Bits[pcmFormatS16LE/32] = 1 << (pcmFormatS16LE % 32)
	// Narrow SUBFORMAT to STD.
	p.Masks[hwParamSubformat].Bits = [8]uint32{}
	p.Masks[hwParamSubformat].Bits[pcmSubformatStd/32] = 1 << (pcmSubformatStd % 32)

	pinInterval := func(idx, value uint32) {
		p.Intervals[idx].Min = value
		p.Intervals[idx].Max = value
		p.Intervals[idx].Flags = 0
	}
	pinInterval(hwParamChannels, 1)
	pinInterval(hwParamRate, rate)
	pinInterval(hwParamPeriodSize, periodSize)
	pinInterval(hwParamBufferSize, bufferSize)
	// SAMPLE_BITS = 16, FRAME_BITS = 16 * channels = 16. The
	// kernel derives these from FORMAT + CHANNELS but pinning
	// them defensively avoids hardware-side surprises.
	pinInterval(hwParamSampleBits, 16)
	pinInterval(hwParamFrameBits, 16)

	// Rmask = all params we set (the kernel checks which ones we
	// actually constrained).
	p.Rmask = 0xFFFFFFFF
	if err := b.ioctl(ioctlHwParams, uintptr(unsafe.Pointer(&p))); err != nil {
		return fmt.Errorf("ioctl-alsa: HW_PARAMS: %w", err)
	}
	return nil
}

// setSWParams configures the minimum start / stop threshold so
// WRITEI starts playback automatically once we've buffered a
// period and stops gracefully on underrun.
func (b *ioctlBackend) setSWParams(periodSize, bufferSize uint32) error {
	var p sndPCMSwParams
	p.AvailMin = uint64(periodSize)
	p.StartThreshold = uint64(periodSize) // start after the first full period
	p.StopThreshold = uint64(bufferSize)  // stop when buffer drains entirely
	p.Boundary = 1 << 30                  // arbitrary large wrap-around
	if err := b.ioctl(ioctlSwParams, uintptr(unsafe.Pointer(&p))); err != nil {
		return fmt.Errorf("ioctl-alsa: SW_PARAMS: %w", err)
	}
	return nil
}

// Write feeds int16 PCM via SNDRV_PCM_IOCTL_WRITEI_FRAMES. On
// underrun (EPIPE) the kernel returns the device to XRUN state;
// we recover by re-PREPARE'ing and re-issuing the write rather
// than losing the chunk.
func (b *ioctlBackend) Write(samples []int16) error {
	if len(samples) == 0 {
		return nil
	}
	frames := uint64(len(samples))
	written := uint64(0)
	for written < frames {
		var xferi sndXferi
		xferi.Buf = uintptr(unsafe.Pointer(&samples[written]))
		xferi.Frames = frames - written
		err := b.ioctl(ioctlWriteI, uintptr(unsafe.Pointer(&xferi)))
		if err != nil {
			if errors.Is(err, syscall.EPIPE) {
				// Underrun. Re-arm the device and try the
				// remaining frames again.
				if perr := b.ioctl(ioctlPrepare, 0); perr != nil {
					return fmt.Errorf("ioctl-alsa: PREPARE after underrun: %w", perr)
				}
				continue
			}
			return fmt.Errorf("ioctl-alsa: WRITEI: %w", err)
		}
		if xferi.Result < 0 {
			return fmt.Errorf("ioctl-alsa: WRITEI result %d", xferi.Result)
		}
		written += uint64(xferi.Result)
	}
	return nil
}

// Close drains the buffer and releases the fd.
func (b *ioctlBackend) Close() error {
	b.closeOnce.Do(func() {
		_ = b.ioctl(ioctlDrain, 0)
		b.closeErr = b.f.Close()
	})
	return b.closeErr
}

// ioctl is a thin wrapper around syscall.Syscall(SYS_IOCTL, ...).
func (b *ioctlBackend) ioctl(req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, b.f.Fd(), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}
