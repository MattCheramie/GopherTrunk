package sdr

import "testing"

// A per-device sample rate (issue #1260): an AirspyHF cannot reach the
// RTL-SDRs' shared 2.4 MS/s, so its hint carries its own rate. The pool must
// program that rate on open AND re-apply it after a USB reacquire — falling
// back to the pool rate there would silently retune a scanner-owned SDR to a
// rate its scanner and voice chain were not built for.
func TestPoolProgramsPerDeviceSampleRate(t *testing.T) {
	drv := &reacquireDriver{name: "fake-dev-rate", infos: []Info{
		{Driver: "fake-dev-rate", Index: 0, Serial: "RTL"},
		{Driver: "fake-dev-rate", Index: 1, Serial: "HF"},
	}}
	registerDriver(t, drv.name, drv)

	p := NewPool(nil)
	if err := p.Open(2_400_000, []Hint{
		{Serial: "RTL", Role: RoleVoice},
		{Serial: "HF", Role: RoleVoice, SampleRateHz: 912_000},
	}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	for serial, want := range map[string]uint32{"RTL": 2_400_000, "HF": 912_000} {
		e := p.FindBySerial(serial)
		if got := e.Device.(*fakeDevice).sampleRate; got != want {
			t.Errorf("%s programmed at %d S/s, want %d", serial, got, want)
		}
		if got := e.SampleRateHz(2_400_000); got != want {
			t.Errorf("%s entry reports %d S/s, want %d", serial, got, want)
		}
	}

	got, err := p.Reacquire("HF", 2_400_000)
	if err != nil {
		t.Fatalf("Reacquire: %v", err)
	}
	if r := got.Device.(*fakeDevice).sampleRate; r != 912_000 {
		t.Errorf("after reacquire HF runs at %d S/s, want its own 912000", r)
	}
	if r := (*PoolEntry)(nil).SampleRateHz(2_400_000); r != 2_400_000 {
		t.Errorf("nil entry rate = %d, want the pool rate", r)
	}
}

// snappingDevice runs only the rates in its table, like an AirspyHF, and
// reports the one it picked through ActualSampleRate.
type snappingDevice struct {
	*fakeDevice
	table []uint32
}

func (d *snappingDevice) SetSampleRate(hz uint32) error {
	best := d.table[0]
	for _, r := range d.table {
		if absDiff(r, hz) < absDiff(best, hz) {
			best = r
		}
	}
	d.sampleRate = best
	return nil
}

func (d *snappingDevice) ActualSampleRate() (uint32, error) { return d.sampleRate, nil }

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

type snappingDriver struct{ reacquireDriver }

func (s *snappingDriver) Open(idx int) (Device, error) {
	dev, err := s.reacquireDriver.Open(idx)
	if err != nil {
		return nil, err
	}
	return &snappingDevice{fakeDevice: dev.(*fakeDevice), table: []uint32{912_000, 768_000}}, nil
}

// A per-device sample_rate the radio cannot run must not leave it running at
// a nearby rate the scanner was not built for: an HF+ asked for 900 kS/s runs
// 912 kS/s, which mis-sizes every filter downstream. The pool refuses the
// device, and accepts one whose rate is in its table.
func TestPoolRefusesDeviceThatCannotRunItsSampleRate(t *testing.T) {
	drv := &snappingDriver{reacquireDriver{name: "fake-snap", infos: []Info{
		{Driver: "fake-snap", Index: 0, Serial: "HF-BAD"},
		{Driver: "fake-snap", Index: 1, Serial: "HF-OK"},
	}}}
	registerDriver(t, drv.name, drv)

	p := NewPool(nil)
	if err := p.Open(2_400_000, []Hint{
		{Serial: "HF-BAD", Role: RoleVoice, SampleRateHz: 900_000},
		{Serial: "HF-OK", Role: RoleVoice, SampleRateHz: 768_000},
	}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	if p.FindBySerial("HF-BAD") != nil {
		t.Error("HF-BAD runs 912 kS/s for a requested 900 kS/s; the pool should refuse it")
	}
	if !drv.openedDevices[0].closed {
		t.Error("the refused device was not closed")
	}
	if p.FindBySerial("HF-OK") == nil {
		t.Error("HF-OK runs exactly its 768 kS/s and should be in the pool")
	}
}
