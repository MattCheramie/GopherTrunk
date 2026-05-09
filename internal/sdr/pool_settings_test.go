package sdr

import (
	"testing"
)

func TestPoolAppliesHintSettings(t *testing.T) {
	drv := &fakeDriver{name: "fake-settings", infos: []Info{
		{Driver: "fake-settings", Index: 0, Serial: "S1"},
	}}
	registryMu.Lock()
	registry["fake-settings"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-settings")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open([]Hint{
		Hint{Serial: "S1", PPM: 7, BiasTee: true}.WithGain(496),
	}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	entries := p.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	dev, ok := entries[0].Device.(*fakeDevice)
	if !ok {
		t.Fatalf("entry device is not *fakeDevice: %T", entries[0].Device)
	}
	if dev.biasTeeSets != 1 {
		t.Errorf("SetBiasTee invocations = %d, want 1", dev.biasTeeSets)
	}
	if !dev.biasTeeOn {
		t.Errorf("biasTeeOn = false, want true")
	}
}

func TestPoolSkipsBiasTeeWhenHintFalse(t *testing.T) {
	drv := &fakeDriver{name: "fake-no-bias", infos: []Info{
		{Driver: "fake-no-bias", Index: 0, Serial: "S2"},
	}}
	registryMu.Lock()
	registry["fake-no-bias"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-no-bias")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open([]Hint{{Serial: "S2"}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dev := p.Entries()[0].Device.(*fakeDevice)
	if dev.biasTeeSets != 0 {
		t.Errorf("SetBiasTee called %d times when bias_tee was false", dev.biasTeeSets)
	}
}
