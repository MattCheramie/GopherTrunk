package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// registerMockDongles registers n mock SDRs (mock-00 … mock-0n-1).
func registerMockDongles(t *testing.T, n int) {
	t.Helper()
	var files []string
	for i := 0; i < n; i++ {
		p := filepath.Join(t.TempDir(), "wb.cfile")
		if err := os.WriteFile(p, make([]byte, 4096), 0o644); err != nil {
			t.Fatalf("write mock IQ: %v", err)
		}
		files = append(files, p)
	}
	sdr.Register(&sdr.MockDriver{Files: files})
}

// groupDaemonConfig is a three-stick group covering 851–857 MHz: more than
// one 2.4 MS/s RTL-SDR's 2.16 MHz, so the P25 control channels at either end
// can only decode on different members.
func groupDaemonConfig() config.Config {
	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "wideband", Group: "south"},
		{Serial: "mock-01", Role: "wideband", Group: "south"},
		{Serial: "mock-02", Role: "wideband", Group: "south"},
	}
	cfg.SDR.Groups = []config.SDRGroupConfig{{
		Name: "south", LowHz: 851_000_000, HighHz: 857_000_000, VoiceTaps: 1,
		Channels: []config.DeviceChannelConfig{
			{FrequencyHz: 851_012_500, System: "P25"},
			{FrequencyHz: 856_987_500, System: "P25"},
		},
	}}
	cfg.Trunking.Systems = []config.SystemConfig{{
		Name: "P25", Protocol: "p25", ControlChannels: []uint32{851_012_500, 856_987_500},
	}}
	cfg.API.HTTPAddr = ""
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = ""
	return cfg
}

// TestDaemonBuildsSDRGroupAsWidebandMembers pins the group wiring end to end
// at construction: every member gets a wideband engine at its planned centre,
// each control channel decodes on exactly one member (the one that covers it),
// the member with no channel still gets an engine to own its stream, and the
// voice pool's virtual taps cover the group's whole span — a grant only the
// top member reaches lands only on the top member's tap.
func TestDaemonBuildsSDRGroupAsWidebandMembers(t *testing.T) {
	registerMockDongles(t, 3)
	cfg := groupDaemonConfig()
	if errs := cfg.ValidateAll(); len(errs) != 0 {
		t.Fatalf("group config does not validate: %v", errs)
	}
	var logBuf bytes.Buffer
	d, err := NewDaemon(cfg, "test-group", slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if len(d.sdrGroups) != 1 || len(d.sdrGroups[0].Members) != 3 {
		t.Fatalf("group plans = %+v", d.sdrGroups)
	}
	plan := d.sdrGroups[0]
	if len(d.widebandT2) != 3 {
		t.Fatalf("%d wideband engines, want one per member", len(d.widebandT2))
	}
	owner := map[uint32]string{}
	for _, eng := range d.widebandT2 {
		if eng.Group() != "south" {
			t.Errorf("engine %s group = %q", eng.Serial(), eng.Group())
		}
		m, _, ok := d.sdrGroupMember(eng.Serial())
		if !ok || m.CenterFreqHz != eng.CenterFreqHz() {
			t.Errorf("engine %s tuned %d, plan says %+v", eng.Serial(), eng.CenterFreqHz(), m)
		}
		for _, f := range eng.Channels() {
			if prev, dup := owner[f]; dup {
				t.Errorf("%d Hz decoded on both %s and %s", f, prev, eng.Serial())
			}
			owner[f] = eng.Serial()
			if !m.Covers(f) {
				t.Errorf("%d Hz assigned to %s, outside its window %+v", f, eng.Serial(), m)
			}
		}
	}
	if len(owner) != 2 || owner[851_012_500] == owner[856_987_500] {
		t.Errorf("control channel owners = %v, want the two CCs on different members", owner)
	}
	// The original config the API serves is untouched by the expansion.
	if got := d.Cfg().SDR.Devices[0]; got.CenterFreqHz != 0 || len(got.Channels) != 0 {
		t.Errorf("daemon's served config was expanded: %+v", got)
	}

	// Voice: one tap per member, and a grant only the top member's window
	// holds is tunable on that member's tap alone.
	if len(d.virtualVoiceTuners) != 3 {
		t.Fatalf("%d virtual voice taps, want 1 per member", len(d.virtualVoiceTuners))
	}
	top := plan.Members[2]
	grant := top.HighHz - 20_000
	for _, vt := range d.virtualVoiceTuners {
		onTop := strings.Contains(vt.Serial(), top.Serial)
		if vt.CanTune(grant) != onTop {
			t.Errorf("tap %s CanTune(%d) = %v, want %v", vt.Serial(), grant, vt.CanTune(grant), onTop)
		}
	}
	// Every frequency of the span is reachable by some tap.
	for f := plan.LowHz; f <= plan.HighHz; f += 50_000 {
		ok := false
		for _, vt := range d.virtualVoiceTuners {
			if vt.CanTune(f) {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("no voice tap reaches %d Hz inside the group span", f)
		}
	}
	if !strings.Contains(logBuf.String(), "sdr group member") {
		t.Error("the group plan was not logged at startup")
	}
	if d.sdrGroupOf("mock-01") != "south" || d.sdrGroupOf("nope") != "" {
		t.Error("sdrGroupOf lookups wrong")
	}

	// A member losing its stream names the slice that goes dark.
	logBuf.Reset()
	d.warnSDRGroupMemberLost(top.Serial, errors.New("usb gone"))
	if out := logBuf.String(); !strings.Contains(out, "group=south") || !strings.Contains(out, "dark=") {
		t.Errorf("member-loss WARN = %q", out)
	}
}

// TestDaemonRejectsUnplannableGroup: a group its members cannot cover fails
// construction with the planner's error rather than starting half-blind.
func TestDaemonRejectsUnplannableGroup(t *testing.T) {
	registerMockDongles(t, 3)
	cfg := groupDaemonConfig()
	cfg.SDR.Devices = cfg.SDR.Devices[:1]
	_, err := NewDaemon(cfg, "test-group", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil || !strings.Contains(err.Error(), "uncovered") {
		t.Fatalf("NewDaemon err = %v, want the planner's uncovered-range error", err)
	}
}
