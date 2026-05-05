package sdr

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"
)

// MockDriver replays unsigned-8-bit IQ files (.cfile / .iq) from a directory.
// Each file becomes one logical "device". Used for tests and offline replay.
type MockDriver struct {
	Files []string
}

const MockDriverName = "mock"

func (m *MockDriver) Name() string { return MockDriverName }

func (m *MockDriver) Enumerate() ([]Info, error) {
	out := make([]Info, len(m.Files))
	for i, f := range m.Files {
		out[i] = Info{
			Driver:    MockDriverName,
			Index:     i,
			Serial:    fmt.Sprintf("mock-%02d", i),
			Product:   "MockIQ",
			TunerName: "file",
			Gains:     []int{0},
		}
		_ = f
	}
	return out, nil
}

func (m *MockDriver) Open(idx int) (Device, error) {
	if idx < 0 || idx >= len(m.Files) {
		return nil, fmt.Errorf("mock: index %d out of range", idx)
	}
	return &MockDevice{path: m.Files[idx], info: Info{Driver: MockDriverName, Index: idx, Serial: fmt.Sprintf("mock-%02d", idx)}, sampleRate: 2_400_000}, nil
}

// MockDevice replays a single .cfile in real time.
type MockDevice struct {
	path       string
	info       Info
	sampleRate uint32
	mu         sync.Mutex
	closed     bool
}

func (d *MockDevice) Info() Info                 { return d.info }
func (d *MockDevice) SetCenterFreq(uint32) error { return nil }
func (d *MockDevice) SetGain(int) error          { return nil }
func (d *MockDevice) SetPPM(int) error           { return nil }
func (d *MockDevice) SetSampleRate(hz uint32) error {
	d.sampleRate = hz
	return nil
}

func (d *MockDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}

// StreamIQ reads the file in chunks of ~16 KiB and meters delivery to roughly
// match the configured sample rate. Closing the channel signals EOF.
func (d *MockDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	f, err := os.Open(d.path)
	if err != nil {
		return nil, err
	}
	out := make(chan []complex64, 8)
	const chunkSamples = 8192
	chunkBytes := chunkSamples * 2 // u8 IQ pairs

	go func() {
		defer close(out)
		defer f.Close()
		buf := make([]byte, chunkBytes)
		ticker := time.NewTicker(time.Duration(float64(time.Second) * float64(chunkSamples) / float64(d.sampleRate)))
		defer ticker.Stop()
		for {
			n, err := io.ReadFull(f, buf)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				if n == 0 {
					return
				}
			} else if err != nil {
				return
			}
			samples := decodeU8(buf[:n])
			select {
			case <-ctx.Done():
				return
			case out <- samples:
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

func decodeU8(buf []byte) []complex64 {
	n := len(buf) / 2
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		i8 := float32(buf[2*i]) - 127.5
		q8 := float32(buf[2*i+1]) - 127.5
		out[i] = complex(i8/127.5, q8/127.5)
	}
	return out
}

// MockFloat32Driver replays interleaved-float32 IQ files (GNU Radio cfile).
type MockFloat32Driver struct {
	Files []string
}

const MockFloat32DriverName = "mock-f32"

func (m *MockFloat32Driver) Name() string { return MockFloat32DriverName }

func (m *MockFloat32Driver) Enumerate() ([]Info, error) {
	out := make([]Info, len(m.Files))
	for i := range m.Files {
		out[i] = Info{Driver: MockFloat32DriverName, Index: i, Serial: fmt.Sprintf("mockf32-%02d", i), TunerName: "file"}
	}
	return out, nil
}

func (m *MockFloat32Driver) Open(idx int) (Device, error) {
	if idx < 0 || idx >= len(m.Files) {
		return nil, fmt.Errorf("mock-f32: index %d out of range", idx)
	}
	return &mockF32Device{path: m.Files[idx], info: Info{Driver: MockFloat32DriverName, Index: idx, Serial: fmt.Sprintf("mockf32-%02d", idx)}, sampleRate: 2_400_000}, nil
}

type mockF32Device struct {
	path       string
	info       Info
	sampleRate uint32
}

func (d *mockF32Device) Info() Info                    { return d.info }
func (d *mockF32Device) SetCenterFreq(uint32) error    { return nil }
func (d *mockF32Device) SetGain(int) error             { return nil }
func (d *mockF32Device) SetPPM(int) error              { return nil }
func (d *mockF32Device) SetSampleRate(hz uint32) error { d.sampleRate = hz; return nil }
func (d *mockF32Device) Close() error                  { return nil }

func (d *mockF32Device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	f, err := os.Open(d.path)
	if err != nil {
		return nil, err
	}
	out := make(chan []complex64, 8)
	const chunkSamples = 4096
	go func() {
		defer close(out)
		defer f.Close()
		buf := make([]byte, chunkSamples*8) // 2 * float32 per sample
		ticker := time.NewTicker(time.Duration(float64(time.Second) * float64(chunkSamples) / float64(d.sampleRate)))
		defer ticker.Stop()
		for {
			n, err := io.ReadFull(f, buf)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				if n == 0 {
					return
				}
			} else if err != nil {
				return
			}
			samples := decodeF32(buf[:n])
			select {
			case <-ctx.Done():
				return
			case out <- samples:
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

func decodeF32(buf []byte) []complex64 {
	n := len(buf) / 8
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		ri := readF32LE(buf[8*i:])
		qi := readF32LE(buf[8*i+4:])
		out[i] = complex(ri, qi)
	}
	return out
}

func readF32LE(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}
