package sdr

import (
	"context"
	"errors"
	"io"
	"testing"
)

type fakeDriver struct {
	name  string
	infos []Info
}

func (f *fakeDriver) Name() string                 { return f.name }
func (f *fakeDriver) Enumerate() ([]Info, error)   { return f.infos, nil }
func (f *fakeDriver) Open(idx int) (Device, error) { return &fakeDevice{info: f.infos[idx]}, nil }

type fakeDevice struct {
	info        Info
	closed      bool
	biasTeeOn   bool
	biasTeeSets int
	sampleRate  uint32
	rateErr     error
}

func (d *fakeDevice) Info() Info                 { return d.info }
func (d *fakeDevice) SetCenterFreq(uint32) error { return nil }
func (d *fakeDevice) SetSampleRate(hz uint32) error {
	if d.rateErr != nil {
		return d.rateErr
	}
	d.sampleRate = hz
	return nil
}
func (d *fakeDevice) SetGain(int) error                                    { return nil }
func (d *fakeDevice) SetPPM(int) error                                     { return nil }
func (d *fakeDevice) SetBiasTee(on bool) error                             { d.biasTeeOn = on; d.biasTeeSets++; return nil }
func (d *fakeDevice) StreamIQ(context.Context) (<-chan []complex64, error) { return nil, io.EOF }
func (d *fakeDevice) Close() error {
	if d.closed {
		return errors.New("already closed")
	}
	d.closed = true
	return nil
}

func TestPoolAssignsRoles(t *testing.T) {
	drv := &fakeDriver{name: "fake-pool", infos: []Info{
		{Driver: "fake-pool", Index: 0, Serial: "AAA"},
		{Driver: "fake-pool", Index: 1, Serial: "BBB"},
		{Driver: "fake-pool", Index: 2, Serial: "CCC"},
	}}
	registryMu.Lock()
	registry["fake-pool"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-pool")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open(0, []Hint{{Serial: "BBB", Role: RoleControl}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	entries := p.Entries()
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}

	roles := map[string]Role{}
	for _, e := range entries {
		roles[e.Info.Serial] = e.Role
	}
	if roles["BBB"] != RoleControl {
		t.Errorf("BBB role = %v, want control", roles["BBB"])
	}
	// Non-hinted devices get auto assignment, with first device taking
	// the still-unassigned control slot if no other hint claimed it.
	// Here BBB is hinted control, so AAA and CCC should be voice.
	if roles["AAA"] != RoleVoice || roles["CCC"] != RoleVoice {
		t.Errorf("AAA=%v CCC=%v, want both voice", roles["AAA"], roles["CCC"])
	}
}

// TestPoolProgramsSampleRate guards against the bug behind issue #275:
// without a SetSampleRate call at pool-open time the chip streams at
// whatever rate its resampler powered up at, the decoder pipeline runs
// its symbol-timing math against the configured rate, and the result
// is a silent failure to lock.
func TestPoolProgramsSampleRate(t *testing.T) {
	drv := &fakeDriver{name: "fake-rate", infos: []Info{
		{Driver: "fake-rate", Index: 0, Serial: "R1"},
		{Driver: "fake-rate", Index: 1, Serial: "R2"},
	}}
	registryMu.Lock()
	registry["fake-rate"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-rate")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open(2_400_000, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	for _, e := range p.Entries() {
		fd, ok := e.Device.(*fakeDevice)
		if !ok {
			t.Fatalf("device %s not *fakeDevice", e.Info.Serial)
		}
		if fd.sampleRate != 2_400_000 {
			t.Errorf("%s sample rate = %d, want 2400000", e.Info.Serial, fd.sampleRate)
		}
	}
}

// TestPoolDefaultsZeroSampleRate verifies the librtlsdr-parity fallback
// when the daemon hasn't been configured with an sdr.sample_rate.
func TestPoolDefaultsZeroSampleRate(t *testing.T) {
	drv := &fakeDriver{name: "fake-default-rate", infos: []Info{
		{Driver: "fake-default-rate", Index: 0, Serial: "D1"},
	}}
	registryMu.Lock()
	registry["fake-default-rate"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-default-rate")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open(0, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	fd := p.Entries()[0].Device.(*fakeDevice)
	if fd.sampleRate != DefaultSampleRateHz {
		t.Errorf("sample rate = %d, want %d", fd.sampleRate, DefaultSampleRateHz)
	}
}

func TestPoolFirstByRole(t *testing.T) {
	drv := &fakeDriver{name: "fake-first", infos: []Info{
		{Driver: "fake-first", Index: 0, Serial: "X"},
		{Driver: "fake-first", Index: 1, Serial: "Y"},
	}}
	registryMu.Lock()
	registry["fake-first"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-first")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open(0, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if e := p.FirstByRole(RoleControl); e == nil || e.Info.Serial != "X" {
		t.Errorf("control = %+v, want X", e)
	}
	if e := p.FirstByRole(RoleVoice); e == nil || e.Info.Serial != "Y" {
		t.Errorf("voice = %+v, want Y", e)
	}
}
