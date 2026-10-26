package main

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// fixedSDRDriver enumerates a fixed set of serials; its devices accept
// every setting and stream nothing until ctx ends.
type fixedSDRDriver struct {
	name    string
	serials []string
}

func (f *fixedSDRDriver) Name() string { return f.name }
func (f *fixedSDRDriver) Enumerate() ([]sdr.Info, error) {
	infos := make([]sdr.Info, len(f.serials))
	for i, s := range f.serials {
		infos[i] = sdr.Info{Driver: f.name, Index: i, Serial: s}
	}
	return infos, nil
}
func (f *fixedSDRDriver) Open(idx int) (sdr.Device, error) {
	return &fixedSDRDevice{info: sdr.Info{Driver: f.name, Index: idx, Serial: f.serials[idx]}}, nil
}

type fixedSDRDevice struct{ info sdr.Info }

func (d *fixedSDRDevice) Info() sdr.Info             { return d.info }
func (d *fixedSDRDevice) SetCenterFreq(uint32) error { return nil }
func (d *fixedSDRDevice) SetSampleRate(uint32) error { return nil }
func (d *fixedSDRDevice) SetGain(int) error          { return nil }
func (d *fixedSDRDevice) SetPPM(int) error           { return nil }
func (d *fixedSDRDevice) SetBiasTee(bool) error      { return nil }
func (d *fixedSDRDevice) Close() error               { return nil }
func (d *fixedSDRDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	go func() { <-ctx.Done(); close(ch) }()
	return ch, nil
}

// TestDaemonReservesDecoderDonglesFromThePool is the #1316 report-3
// regression: four `auto` RTL-SDRs, APRS pinned to one and ADS-B to
// another, no trunked system. The pool used to make the ADS-B stick THE
// control SDR (rigctld attached to it) and the APRS stick a voice SDR
// (the manual-tune scanner and a live hunt both picked "the last voice
// SDR"). Now both open as role: dedicated, the free sticks take control /
// voice, and no by-role consumer sees the pinned ones.
func TestDaemonReservesDecoderDonglesFromThePool(t *testing.T) {
	drv := &fixedSDRDriver{name: "fake-dedicated-rig", serials: []string{"00000001", "00000002", "54300989", "00000004"}}
	sdr.Register(drv)

	cfg := config.Default()
	cfg.API.HTTPAddr = ""
	cfg.Trunking.Systems = nil
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "00000001", Role: "auto"},
		{Serial: "00000002", Role: "auto"},
		{Serial: "54300989", Role: "auto"},
		{Serial: "00000004", Role: "auto"},
	}
	cfg.APRS.Channels = []config.APRSChannelConfig{{Serial: "54300989", FrequencyHz: 144_390_000}}
	cfg.ADSB.Channels = []config.ADSBChannelConfig{{Serial: "00000001"}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "dedicated-test", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)
	if d.pool == nil {
		t.Fatalf("pool did not open: %v", d.StartupWarnings())
	}

	roles := map[string]sdr.Role{}
	for _, e := range d.pool.Entries() {
		roles[e.Info.Serial] = e.Role
	}
	if roles["00000001"] != sdr.RoleDedicated || roles["54300989"] != sdr.RoleDedicated {
		t.Errorf("pinned sticks: adsb=%v aprs=%v, want dedicated", roles["00000001"], roles["54300989"])
	}
	if roles["00000002"] != sdr.RoleControl || roles["00000004"] != sdr.RoleVoice {
		t.Errorf("free sticks: 00000002=%v 00000004=%v, want control / voice", roles["00000002"], roles["00000004"])
	}
	// rigctld / the CC hunter attach to the first control SDR: never a pinned one.
	if c := d.pool.FirstByRole(sdr.RoleControl); c == nil || c.Info.Serial != "00000002" {
		t.Errorf("FirstByRole(control) = %v, want 00000002", c)
	}
	// The trunking voice pool never lists a pinned stick.
	for _, v := range d.collectVoiceDevices() {
		if v.Serial == "00000001" || v.Serial == "54300989" {
			t.Errorf("voice pool hands out pinned stick %s", v.Serial)
		}
	}
	// One free voice SDR only: the manual-tune scanner's "spare voice SDR"
	// auto-enable (needs two) must not fire, so no conventional scanner.
	if len(d.convScans) != 0 {
		t.Errorf("manual-tune scanner auto-enabled on %d groups; the pinned sticks must not count as spare voice SDRs", len(d.convScans))
	}
	// The decoders themselves are built for the dongles.
	if len(d.aprsReceivers) != 1 || d.aprsReceivers[0] == nil || len(d.adsbPPMReceivers) != 1 || d.adsbPPMReceivers[0] == nil {
		t.Errorf("decoders not built: aprs=%v adsb=%v", d.aprsReceivers, d.adsbPPMReceivers)
	}
}
