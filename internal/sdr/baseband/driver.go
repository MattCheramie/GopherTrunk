package baseband

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// FileDriverName is the sdr.Driver name for baseband replay devices.
const FileDriverName = "baseband-replay"

// ReplaySpec names one baseband WAV recording to mount as a virtual
// tuner. Path is required; Serial defaults to a generated value and
// Loop (default true) restarts the recording on EOF so an offline
// tuner behaves like a continuous source.
type ReplaySpec struct {
	Path   string
	Serial string
	Loop   bool
}

// FileDriver mounts baseband WAV recordings as virtual sdr.Devices so
// they can be opened by the pool exactly like real tuners. Register an
// instance with sdr.Register before Pool.Open.
type FileDriver struct {
	specs []ReplaySpec
}

// NewFileDriver builds a replay driver over the given recordings.
func NewFileDriver(specs []ReplaySpec) *FileDriver {
	return &FileDriver{specs: specs}
}

// Name implements sdr.Driver.
func (d *FileDriver) Name() string { return FileDriverName }

// Enumerate validates each recording's header and returns one Info per
// playable file. A recording that fails to parse is skipped.
func (d *FileDriver) Enumerate() ([]sdr.Info, error) {
	var out []sdr.Info
	for i, spec := range d.specs {
		if _, err := ReadIQWavInfo(spec.Path); err != nil {
			continue
		}
		out = append(out, sdr.Info{
			Driver:    FileDriverName,
			Index:     i,
			Serial:    replaySerial(spec, i),
			Product:   "BasebandReplay",
			TunerName: "file",
			Gains:     []int{0},
		})
	}
	return out, nil
}

// Open returns a replay device for the spec at idx.
func (d *FileDriver) Open(idx int) (sdr.Device, error) {
	if idx < 0 || idx >= len(d.specs) {
		return nil, fmt.Errorf("baseband: replay index %d out of range", idx)
	}
	spec := d.specs[idx]
	info, err := ReadIQWavInfo(spec.Path)
	if err != nil {
		return nil, fmt.Errorf("baseband: %s: %w", spec.Path, err)
	}
	return &replayDevice{
		path: spec.Path,
		loop: spec.Loop,
		info: sdr.Info{
			Driver:    FileDriverName,
			Index:     idx,
			Serial:    replaySerial(spec, idx),
			Product:   "BasebandReplay",
			TunerName: "file",
		},
		wavRate: info.SampleRate,
		rate:    info.SampleRate,
	}, nil
}

func replaySerial(spec ReplaySpec, idx int) string {
	if spec.Serial != "" {
		return spec.Serial
	}
	return fmt.Sprintf("replay-%02d", idx)
}

// replayDevice streams a baseband WAV recording as IQ. SetSampleRate
// overrides the metering rate; otherwise the WAV's native rate is used.
type replayDevice struct {
	path    string
	loop    bool
	info    sdr.Info
	wavRate uint32

	mu   sync.Mutex
	rate uint32
}

func (d *replayDevice) Info() sdr.Info             { return d.info }
func (d *replayDevice) SetCenterFreq(uint32) error { return nil }
func (d *replayDevice) SetGain(int) error          { return nil }
func (d *replayDevice) SetPPM(int) error           { return nil }
func (d *replayDevice) SetBiasTee(bool) error      { return nil }
func (d *replayDevice) Close() error               { return nil }

// SetSampleRate overrides the metering rate. A real-time replay needs
// this to match the recording's rate; the daemon documents that
// sdr.sample_rate should equal the recording rate.
func (d *replayDevice) SetSampleRate(hz uint32) error {
	d.mu.Lock()
	if hz != 0 {
		d.rate = hz
	}
	d.mu.Unlock()
	return nil
}

// StreamIQ replays the recording, metered to the configured rate, and
// loops on EOF when Loop is set.
func (d *replayDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	f, err := os.Open(d.path)
	if err != nil {
		return nil, err
	}
	dataBytes, _, err := parseIQWavHeader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	dataStart, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		f.Close()
		return nil, err
	}
	d.mu.Lock()
	rate := d.rate
	d.mu.Unlock()
	if rate == 0 {
		rate = d.wavRate
	}
	if rate == 0 {
		rate = sdr.DefaultSampleRateHz
	}

	const chunkSamples = 8192
	out := make(chan []complex64, 8)
	go func() {
		defer close(out)
		defer f.Close()
		buf := make([]byte, chunkSamples*iqWavBlockAlign)
		interval := time.Duration(float64(time.Second) * float64(chunkSamples) / float64(rate))
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		remaining := int64(dataBytes)
		for {
			if remaining <= 0 {
				if !d.loop {
					return
				}
				if _, err := f.Seek(dataStart, io.SeekStart); err != nil {
					return
				}
				remaining = int64(dataBytes)
			}
			want := int64(len(buf))
			if want > remaining {
				want = remaining
			}
			n, err := io.ReadFull(f, buf[:want])
			if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			remaining -= int64(n)
			if n > 0 {
				select {
				case out <- decodeIQ16(buf[:n]):
				case <-ctx.Done():
					return
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return out, nil
}

// decodeIQ16 converts interleaved 16-bit I/Q PCM into complex64.
func decodeIQ16(buf []byte) []complex64 {
	n := len(buf) / iqWavBlockAlign
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := int16(binary.LittleEndian.Uint16(buf[4*i:]))
		qv := int16(binary.LittleEndian.Uint16(buf[4*i+2:]))
		out[i] = complex(float32(iv)/32768, float32(qv)/32768)
	}
	return out
}
