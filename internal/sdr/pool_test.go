package sdr

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
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
	ppm         int
	ppmSets     int
	blogV4Sets  int
	blogV4Lite  bool
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
func (d *fakeDevice) SetPPM(ppm int) error                                 { d.ppm = ppm; d.ppmSets++; return nil }
func (d *fakeDevice) SetBiasTee(on bool) error                             { d.biasTeeOn = on; d.biasTeeSets++; return nil }
func (d *fakeDevice) SetBlogV4(lite bool) error                            { d.blogV4Sets++; d.blogV4Lite = lite; return nil }
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

func TestPoolMatchesAirspySerialAliases(t *testing.T) {
	drv := &fakeDriver{name: "fake-airspy-alias", infos: []Info{
		{Driver: "fake-airspy-alias", Index: 0, Serial: "35AC63DC2D701C4F"},
	}}
	registryMu.Lock()
	registry["fake-airspy-alias"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-airspy-alias")
		registryMu.Unlock()
	})

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := NewPool(log)
	if err := p.OpenWith(PoolOpenOptions{
		Hints:  []Hint{{Serial: "AIRSPY SN:35ac63dc2d701c4f", Role: RoleVoice}},
		Strict: true,
	}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	entries := p.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1; log was: %q", len(entries), buf.String())
	}
	if entries[0].Info.Serial != "35AC63DC2D701C4F" {
		t.Errorf("opened serial = %q, want 35AC63DC2D701C4F", entries[0].Info.Serial)
	}
	if got := p.FindBySerial("airspy_sn:35ac63dc2d701c4f"); got == nil {
		t.Fatalf("FindBySerial(alias) returned nil")
	} else if got.Info.Serial != "35AC63DC2D701C4F" {
		t.Errorf("FindBySerial(alias) = %q, want 35AC63DC2D701C4F", got.Info.Serial)
	}
	if got := p.FindBySerial("35ac63dc2d701c4f"); got == nil {
		t.Fatalf("FindBySerial(raw) returned nil")
	}
	out := buf.String()
	if strings.Contains(out, "configured SDR not present on the bus") {
		t.Errorf("unexpected missing-serial warn; log was: %q", out)
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

// TestPoolLogsOnHintWithoutGain guards the no-gain path: a hint that
// reaches the pool without a gain set (e.g. baseband replay / direct
// callers — the daemon now resolves an empty `gain:` to auto upstream)
// must still surface a startup line so the operator knows no explicit
// gain was applied. It is INFO, not WARN: the RTL-SDR driver establishes
// AGC at open (runBringup), so "no explicit gain hint" is no longer the
// deafness hazard it was in issue #356 — it just falls back to AGC. The
// line must include the serial and point at setting a specific gain for
// weak-signal sites, on every role.
func TestPoolLogsOnHintWithoutGain(t *testing.T) {
	drv := &fakeDriver{name: "fake-nogain", infos: []Info{
		{Driver: "fake-nogain", Index: 0, Serial: "NOGAIN-1"},
	}}
	registryMu.Lock()
	registry["fake-nogain"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-nogain")
		registryMu.Unlock()
	})

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := NewPool(log)
	// Hint with serial only — gain is unset.
	if err := p.Open(0, []Hint{{Serial: "NOGAIN-1", Role: RoleVoice}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	out := buf.String()
	if !strings.Contains(out, "no explicit gain hint") {
		t.Errorf("expected no-gain info line for NOGAIN-1; log was: %q", out)
	}
	if !strings.Contains(out, "NOGAIN-1") {
		t.Errorf("line should include the device serial; log was: %q", out)
	}
	if !strings.Contains(out, "AGC default") {
		t.Errorf("line should mention the AGC default; log was: %q", out)
	}
}

// TestPoolSilentOnHintWithGain confirms the warn does NOT fire when
// gain is explicitly configured (auto or numeric). A noisy warn here
// would train operators to ignore the no-gain warn above.
func TestPoolSilentOnHintWithGain(t *testing.T) {
	drv := &fakeDriver{name: "fake-withgain", infos: []Info{
		{Driver: "fake-withgain", Index: 0, Serial: "G-AUTO"},
		{Driver: "fake-withgain", Index: 1, Serial: "G-FIXED"},
	}}
	registryMu.Lock()
	registry["fake-withgain"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-withgain")
		registryMu.Unlock()
	})

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	p := NewPool(log)
	autoHint := Hint{Serial: "G-AUTO", Role: RoleVoice}.WithGain(-1)
	fixedHint := Hint{Serial: "G-FIXED", Role: RoleVoice}.WithGain(496)
	if err := p.Open(0, []Hint{autoHint, fixedHint}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if out := buf.String(); strings.Contains(out, "no explicit gain hint") {
		t.Errorf("hints with gain set should not log the no-gain line; log was: %q", out)
	}
}

// reacquireDriver lets a test swap Enumerate's return between calls
// (simulating a USB unplug → replug) and inject Open / Enumerate
// failures.
type reacquireDriver struct {
	name         string
	infos        []Info
	openErr      error
	enumErr      error
	enumerateCnt int
	opens        []int // record of Open(idx) calls in order
	// openedDevices records every fakeDevice the driver hands out so
	// tests can inspect Close calls on stale handles.
	openedDevices []*fakeDevice
}

func (r *reacquireDriver) Name() string { return r.name }
func (r *reacquireDriver) Enumerate() ([]Info, error) {
	r.enumerateCnt++
	if r.enumErr != nil {
		return nil, r.enumErr
	}
	out := make([]Info, len(r.infos))
	copy(out, r.infos)
	return out, nil
}
func (r *reacquireDriver) Open(idx int) (Device, error) {
	r.opens = append(r.opens, idx)
	if r.openErr != nil {
		return nil, r.openErr
	}
	for _, info := range r.infos {
		if info.Index == idx {
			d := &fakeDevice{info: info}
			r.openedDevices = append(r.openedDevices, d)
			return d, nil
		}
	}
	return nil, errors.New("no device at that index")
}

func registerDriver(t *testing.T, name string, d Driver) {
	t.Helper()
	registryMu.Lock()
	registry[name] = d
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, name)
		registryMu.Unlock()
	})
}

func TestPoolReacquireSwapsDeviceHandleInPlace(t *testing.T) {
	drv := &reacquireDriver{name: "fake-reacq-ok", infos: []Info{
		{Driver: "fake-reacq-ok", Index: 0, Serial: "S1"},
	}}
	registerDriver(t, drv.name, drv)

	p := NewPool(nil)
	if err := p.Open(2_400_000, []Hint{{Serial: "S1", Role: RoleControl, BiasTee: true}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	original := p.FindBySerial("S1")
	if original == nil {
		t.Fatal("FindBySerial(S1) returned nil")
	}
	oldDev := original.Device.(*fakeDevice)
	// Simulate kernel re-enumeration: same serial, new device number.
	drv.infos = []Info{{Driver: drv.name, Index: 7, Serial: "S1"}}

	got, err := p.Reacquire("S1", 2_400_000)
	if err != nil {
		t.Fatalf("Reacquire: %v", err)
	}
	if got != original {
		t.Errorf("Reacquire returned %p, want same PoolEntry %p (in-place swap)", got, original)
	}
	if !oldDev.closed {
		t.Error("expected stale device handle to be Closed during Reacquire")
	}
	newDev, ok := got.Device.(*fakeDevice)
	if !ok || newDev == oldDev {
		t.Errorf("device handle was not swapped: got %p, old %p", got.Device, oldDev)
	}
	if newDev.sampleRate != 2_400_000 {
		t.Errorf("new device sample rate = %d, want 2400000", newDev.sampleRate)
	}
	if !newDev.biasTeeOn {
		t.Error("hint bias-tee not re-applied to fresh handle")
	}
	if got.Info.Index != 7 {
		t.Errorf("Info.Index = %d, want 7 (refreshed from re-enumerate)", got.Info.Index)
	}
	if got.Info.Serial != "S1" || got.Role != RoleControl {
		t.Errorf("identity drifted: serial=%q role=%v", got.Info.Serial, got.Role)
	}
}

func TestPoolReacquireErrorsWhenSerialUnknown(t *testing.T) {
	drv := &reacquireDriver{name: "fake-reacq-unknown", infos: []Info{
		{Driver: "fake-reacq-unknown", Index: 0, Serial: "S1"},
	}}
	registerDriver(t, drv.name, drv)
	p := NewPool(nil)
	if err := p.Open(0, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if _, err := p.Reacquire("does-not-exist", 0); err == nil {
		t.Error("Reacquire with unknown serial: want error, got nil")
	}
}

func TestPoolReacquireErrorsWhenSerialMissingAfterReenumerate(t *testing.T) {
	drv := &reacquireDriver{name: "fake-reacq-gone", infos: []Info{
		{Driver: "fake-reacq-gone", Index: 0, Serial: "S1"},
	}}
	registerDriver(t, drv.name, drv)
	p := NewPool(nil)
	if err := p.Open(0, []Hint{{Serial: "S1", Role: RoleControl}}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Simulate the device staying disconnected — enumerate returns
	// nothing for that serial.
	drv.infos = nil
	_, err := p.Reacquire("S1", 0)
	if err == nil {
		t.Fatal("Reacquire with missing serial: want error, got nil")
	}
	// The stale handle must still have been closed before we tried
	// the re-enumerate — that's the whole point of best-effort
	// cleanup ahead of recovery.
	entry := p.FindBySerial("S1")
	if entry == nil {
		t.Fatal("entry should still be in pool after failed reacquire")
	}
	if !entry.Device.(*fakeDevice).closed {
		t.Error("stale device handle should be Closed even on failed reacquire")
	}
}

// TestPoolStrictOpensOnlyHintedSerial guards the fix for issue #264:
// when the operator names devices in config.yaml, the pool must treat
// that list as an allowlist. Previously every enumerated device was
// opened, which on a multi-stick host meant unrelated dongles got
// taken over (and worse — a non-hinted dongle could win RoleControl,
// so the cchunt path bound to a device that hadn't received the
// operator's PPM correction).
func TestPoolStrictOpensOnlyHintedSerial(t *testing.T) {
	drv := &fakeDriver{name: "fake-strict", infos: []Info{
		{Driver: "fake-strict", Index: 0, Serial: "NOOELEC-1"},
		{Driver: "fake-strict", Index: 1, Serial: "V4"},
		{Driver: "fake-strict", Index: 2, Serial: "NOOELEC-2"},
	}}
	registerDriver(t, drv.name, drv)

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := NewPool(log)
	err := p.OpenWith(PoolOpenOptions{
		SampleRateHz: 2_400_000,
		Hints:        []Hint{{Serial: "V4", PPM: -4}},
		Strict:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	entries := p.Entries()
	if len(entries) != 1 {
		t.Fatalf("strict mode opened %d devices, want 1; log was: %q", len(entries), buf.String())
	}
	if entries[0].Info.Serial != "V4" {
		t.Errorf("opened serial = %q, want V4", entries[0].Info.Serial)
	}
	// The lone opened device should claim RoleControl via the
	// unclaimed-control rule, so FirstByRole(RoleControl) hands the
	// PPM-corrected V4 to cchunt / ccdecoder.
	if entries[0].Role != RoleControl {
		t.Errorf("role = %v, want control", entries[0].Role)
	}
	dev := entries[0].Device.(*fakeDevice)
	if dev.ppmSets != 1 || dev.ppm != -4 {
		t.Errorf("SetPPM not applied: sets=%d ppm=%d, want sets=1 ppm=-4", dev.ppmSets, dev.ppm)
	}
	out := buf.String()
	if !strings.Contains(out, "NOOELEC-1") || !strings.Contains(out, "NOOELEC-2") {
		t.Errorf("expected skip log for both NooElecs; log was: %q", out)
	}
	if !strings.Contains(out, "skipping non-configured SDR") {
		t.Errorf("expected skip-reason log; log was: %q", out)
	}
	if !strings.Contains(out, "ppm=-4") {
		t.Errorf("device-opened log should surface ppm; log was: %q", out)
	}
}

// TestPoolStrictWarnsOnMissingHintSerial covers the operator-error
// path: a serial in config.yaml that no connected dongle reports.
// The pool should warn about the missing hint and leave the other
// physically-present devices closed (strict mode is an allowlist, not
// a "use these or fall back" preference).
func TestPoolStrictWarnsOnMissingHintSerial(t *testing.T) {
	drv := &fakeDriver{name: "fake-strict-missing", infos: []Info{
		{Driver: "fake-strict-missing", Index: 0, Serial: "PRESENT-1"},
		{Driver: "fake-strict-missing", Index: 1, Serial: "PRESENT-2"},
	}}
	registerDriver(t, drv.name, drv)

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := NewPool(log)
	err := p.OpenWith(PoolOpenOptions{
		Hints:  []Hint{{Serial: "MISSING-ZZZ", Role: RoleControl}},
		Strict: true,
	})
	// All discovered devices are non-hinted, so strict mode skips them
	// and OpenWith returns "no SDR devices opened".
	if err == nil {
		t.Fatal("expected error when no hinted device is present, got nil")
	}
	if !strings.Contains(err.Error(), "no SDR devices opened") {
		t.Errorf("err = %v, want 'no SDR devices opened'", err)
	}
	if entries := p.Entries(); len(entries) != 0 {
		t.Errorf("pool has %d entries, want 0 (the present-but-non-hinted devices must not be opened)", len(entries))
	}
	out := buf.String()
	if !strings.Contains(out, "configured SDR not present") || !strings.Contains(out, "MISSING-ZZZ") {
		t.Errorf("expected missing-serial warn; log was: %q", out)
	}
}

// TestPoolStrictIgnoresEmptySerialHint covers the ambiguous case: an
// allowlist entry that doesn't name a device. The hint is dropped at
// ingest with a warn — left in, it would silently match nothing and
// trigger the missing-serial warn with an empty value, which is the
// less actionable failure mode.
func TestPoolStrictIgnoresEmptySerialHint(t *testing.T) {
	drv := &fakeDriver{name: "fake-strict-empty", infos: []Info{
		{Driver: "fake-strict-empty", Index: 0, Serial: "S1"},
	}}
	registerDriver(t, drv.name, drv)

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := NewPool(log)
	err := p.OpenWith(PoolOpenOptions{
		Hints: []Hint{
			{Serial: "", Role: RoleControl, PPM: -4},
			{Serial: "S1"},
		},
		Strict: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	entries := p.Entries()
	if len(entries) != 1 || entries[0].Info.Serial != "S1" {
		t.Fatalf("entries = %+v, want exactly S1", entries)
	}
	// PPM from the empty-serial hint must NOT have leaked onto S1.
	if dev := entries[0].Device.(*fakeDevice); dev.ppmSets != 0 {
		t.Errorf("empty-serial hint leaked SetPPM onto S1: sets=%d ppm=%d", dev.ppmSets, dev.ppm)
	}
	out := buf.String()
	if !strings.Contains(out, "ignoring hint with empty serial in strict mode") {
		t.Errorf("expected empty-serial warn; log was: %q", out)
	}
}

// TestPoolOpenWithNonStrictPreservesLegacyBehavior exercises the new
// OpenWith entry point with Strict: false to confirm the shim path
// matches the historical Open(rate, hints) behaviour: every
// enumerated device opens and non-hinted devices get auto-assigned
// roles.
func TestPoolOpenWithNonStrictPreservesLegacyBehavior(t *testing.T) {
	drv := &fakeDriver{name: "fake-nonstrict", infos: []Info{
		{Driver: "fake-nonstrict", Index: 0, Serial: "AAA"},
		{Driver: "fake-nonstrict", Index: 1, Serial: "BBB"},
		{Driver: "fake-nonstrict", Index: 2, Serial: "CCC"},
	}}
	registerDriver(t, drv.name, drv)

	p := NewPool(nil)
	err := p.OpenWith(PoolOpenOptions{
		Hints:  []Hint{{Serial: "BBB", Role: RoleControl}},
		Strict: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	if entries := p.Entries(); len(entries) != 3 {
		t.Fatalf("non-strict OpenWith opened %d, want 3", len(entries))
	}
}

func TestPoolReacquireErrorsWhenOpenFails(t *testing.T) {
	openErr := errors.New("usb open boom")
	drv := &reacquireDriver{
		name:    "fake-reacq-open-fail",
		infos:   []Info{{Driver: "fake-reacq-open-fail", Index: 0, Serial: "S1"}},
		openErr: nil, // initial Open succeeds during pool.Open
	}
	registerDriver(t, drv.name, drv)
	p := NewPool(nil)
	if err := p.Open(0, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Trip Open() during Reacquire.
	drv.openErr = openErr
	if _, err := p.Reacquire("S1", 0); err == nil || !errors.Is(err, openErr) {
		t.Errorf("Reacquire = %v, want underlying %v", err, openErr)
	}
}
