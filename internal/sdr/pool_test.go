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
	info   Info
	closed bool
}

func (d *fakeDevice) Info() Info                                            { return d.info }
func (d *fakeDevice) SetCenterFreq(uint32) error                            { return nil }
func (d *fakeDevice) SetSampleRate(uint32) error                            { return nil }
func (d *fakeDevice) SetGain(int) error                                     { return nil }
func (d *fakeDevice) SetPPM(int) error                                      { return nil }
func (d *fakeDevice) StreamIQ(context.Context) (<-chan []complex64, error)  { return nil, io.EOF }
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
	if err := p.Open([]Hint{{Serial: "BBB", Role: RoleControl}}); err != nil {
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
	if err := p.Open(nil); err != nil {
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
