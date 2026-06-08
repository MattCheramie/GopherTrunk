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
	if err := p.Open(0, []Hint{
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

// TestPoolForcesBlogV4FromHint pins the issue #264 override: a device
// whose USB strings don't auto-identify it as a V4 still gets SetBlogV4
// driven from config (ForceBlogV4), with the Lite flag propagated.
func TestPoolForcesBlogV4FromHint(t *testing.T) {
	drv := &fakeDriver{name: "fake-v4", infos: []Info{
		{Driver: "fake-v4", Index: 0, Serial: "V4"},
	}}
	registryMu.Lock()
	registry["fake-v4"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-v4")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open(0, []Hint{{Serial: "V4", ForceBlogV4: true, ForceBlogV4Lite: true}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	dev := p.Entries()[0].Device.(*fakeDevice)
	if dev.blogV4Sets != 1 {
		t.Errorf("SetBlogV4 invocations = %d, want 1", dev.blogV4Sets)
	}
	if !dev.blogV4Lite {
		t.Errorf("blogV4Lite = false, want true (ForceBlogV4Lite not propagated)")
	}
}

// TestPoolSkipsBlogV4WhenHintFalse confirms the override is opt-in: no
// ForceBlogV4 means SetBlogV4 is never called, so non-V4 dongles and
// auto-detected V4s are untouched by config.
func TestPoolSkipsBlogV4WhenHintFalse(t *testing.T) {
	drv := &fakeDriver{name: "fake-no-v4", infos: []Info{
		{Driver: "fake-no-v4", Index: 0, Serial: "S3"},
	}}
	registryMu.Lock()
	registry["fake-no-v4"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-no-v4")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open(0, []Hint{{Serial: "S3"}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dev := p.Entries()[0].Device.(*fakeDevice)
	if dev.blogV4Sets != 0 {
		t.Errorf("SetBlogV4 called %d times when blog_v4 was false", dev.blogV4Sets)
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
	if err := p.Open(0, []Hint{{Serial: "S2"}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dev := p.Entries()[0].Device.(*fakeDevice)
	if dev.biasTeeSets != 0 {
		t.Errorf("SetBiasTee called %d times when bias_tee was false", dev.biasTeeSets)
	}
}
