package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

func convTestEntries(roles ...sdr.Role) ([]*sdr.PoolEntry, func(string) *sdr.PoolEntry) {
	var all, voices []*sdr.PoolEntry
	for i, r := range roles {
		e := &sdr.PoolEntry{Info: sdr.Info{Serial: string(rune('A' + i))}, Role: r}
		all = append(all, e)
		if r == sdr.RoleVoice {
			voices = append(voices, e)
		}
	}
	find := func(s string) *sdr.PoolEntry {
		for _, e := range all {
			if e.Info.Serial == s {
				return e
			}
		}
		return nil
	}
	return voices, find
}

type convPlanRow struct {
	serial    string
	channels  []int
	dedicated bool
}

func convPlanRows(groups []convScanGroup) []convPlanRow {
	var out []convPlanRow
	for _, g := range groups {
		out = append(out, convPlanRow{g.entry.Info.Serial, g.channels, g.dedicated})
	}
	return out
}

// TestPlanConvScanGroups pins how issue #1260 splits the scan list: one scanner
// per named device, in order of first appearance, and the shared scanner SDR
// (the last voice SDR no channel names — the pre-#1260 choice) LAST, holding
// every channel without a usable device, so manual-tune channels appended to it
// shift nobody's numbering.
func TestPlanConvScanGroups(t *testing.T) {
	voices, find := convTestEntries(sdr.RoleVoice, sdr.RoleVoice, sdr.RoleVoice, sdr.RoleControl)
	chans := []config.ConvChannelConfig{
		{Label: "air1", Device: "C"},
		{Label: "vhf1"},
		{Label: "uhf1", Device: "B"},
		{Label: "air2", Device: " C "},
		{Label: "typo", Device: "Z"}, // not in the pool → shared
		{Label: "ctrl", Device: "D"}, // control SDR → shared
	}
	got := convPlanRows(planConvScanGroups(chans, voices, find, false, nil))
	want := []convPlanRow{
		{"C", []int{0, 3}, true},
		{"B", []int{2}, true},
		{"A", []int{1, 4, 5}, false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groups = %+v, want %+v", got, want)
	}
	if got := convDedicatedSerials(chans, find); !reflect.DeepEqual(got, map[string]bool{"B": true, "C": true}) {
		t.Errorf("dedicated = %v, want B and C", got)
	}
}

// TestPlanConvScanGroupsLegacy pins the no-device config: exactly today's
// single scanner on the last voice SDR, nothing withheld from trunking.
func TestPlanConvScanGroupsLegacy(t *testing.T) {
	voices, find := convTestEntries(sdr.RoleVoice, sdr.RoleVoice)
	chans := []config.ConvChannelConfig{{Label: "a"}, {Label: "b"}}
	got := convPlanRows(planConvScanGroups(chans, voices, find, false, nil))
	if want := []convPlanRow{{"B", []int{0, 1}, false}}; !reflect.DeepEqual(got, want) {
		t.Errorf("groups = %+v, want %+v", got, want)
	}
	if got := convDedicatedSerials(chans, find); len(got) != 0 {
		t.Errorf("dedicated = %v, want none", got)
	}
	// Manual tune with an empty scan list still gets the shared scanner.
	got = convPlanRows(planConvScanGroups(nil, voices, find, true, nil))
	if want := []convPlanRow{{"B", nil, false}}; !reflect.DeepEqual(got, want) {
		t.Errorf("manual-tune groups = %+v, want %+v", got, want)
	}
}

// TestPlanConvScanGroupsAllClaimed: when every voice SDR is named, channels
// without a device join the last voice SDR's list (never dropped), and that
// scanner moves last so it is the manual-tune target.
func TestPlanConvScanGroupsAllClaimed(t *testing.T) {
	voices, find := convTestEntries(sdr.RoleVoice, sdr.RoleVoice)
	chans := []config.ConvChannelConfig{
		{Label: "b1", Device: "B"},
		{Label: "free"},
		{Label: "a1", Device: "A"},
	}
	got := convPlanRows(planConvScanGroups(chans, voices, find, false, nil))
	want := []convPlanRow{
		{"A", []int{2}, true},
		{"B", []int{0, 1}, true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groups = %+v, want %+v", got, want)
	}
}

// TestConventionalScannerPerDeviceDaemon is the daemon-wiring half of #1260:
// channels naming a device get their own scanner on that SDR, the named SDRs are
// withheld from the trunking voice pool (a grant must never retune a scanner's
// radio), and the cockpit presents every scanner's channels as one list, each
// row tagged with its SDR.
func TestConventionalScannerPerDeviceDaemon(t *testing.T) {
	dir := t.TempDir()
	var files []string
	for i := 0; i < 3; i++ {
		p := filepath.Join(dir, "voice"+string(rune('0'+i))+".cfile")
		if err := os.WriteFile(p, make([]byte, 4096), 0o644); err != nil {
			t.Fatalf("write mock IQ: %v", err)
		}
		files = append(files, p)
	}
	sdr.Register(&sdr.MockDriver{Files: files})

	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "voice"},
		{Serial: "mock-01", Role: "voice"},
		{Serial: "mock-02", Role: "voice"},
	}
	cfg.Scanner.Conventional = []config.ConvChannelConfig{
		{Label: "AIR", FrequencyHz: 126_400_000, Mode: "am", Device: "mock-01"},
		{Label: "VHF", FrequencyHz: 146_670_000, Mode: "nfm", SquelchDbFS: -10},
		{Label: "AIR2", FrequencyHz: 118_300_000, Mode: "am", Device: "mock-01"},
	}
	cfg.Recordings.Dir = t.TempDir()
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.API.HTTPAddr = ""

	d, err := NewDaemon(cfg, "test-convscan-multi", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	if len(d.convScans) != 2 {
		t.Fatalf("scanners = %d, want 2 (mock-01 for the air band, the shared mock-02)", len(d.convScans))
	}
	var poolSerials []string
	for _, vd := range d.voicePool.Devices() {
		poolSerials = append(poolSerials, vd.Serial)
	}
	for _, s := range poolSerials {
		if s == "mock-01" {
			t.Fatalf("voice pool %v includes the scanner-owned mock-01; a grant could retune it", poolSerials)
		}
	}
	for _, want := range []string{"mock-00", "mock-02"} {
		found := false
		for _, s := range poolSerials {
			found = found || s == want
		}
		if !found {
			t.Errorf("voice pool %v lost %s, which no channel names", poolSerials, want)
		}
	}
	for _, s := range []string{"mock-01", "mock-02"} {
		if d.scannerBrokers[s] == nil {
			t.Errorf("scanner serial %s not registered in scannerBrokers (#1075)", s)
		}
	}

	st := scannerCockpit{convs: d.convScans}.Status().Conventional
	type row struct {
		idx    int
		label  string
		serial string
	}
	var got []row
	for _, ch := range st.Channels {
		got = append(got, row{ch.Index, ch.Label, ch.DeviceSerial})
	}
	want := []row{{0, "AIR", "mock-01"}, {1, "AIR2", "mock-01"}, {2, "VHF", "mock-02"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cockpit channels = %+v, want %+v", got, want)
	}
	if st.DeviceSerial != "mock-02" || !reflect.DeepEqual(st.DeviceSerials, []string{"mock-01", "mock-02"}) {
		t.Errorf("device_serial = %q, device_serials = %v; want the shared mock-02 and both", st.DeviceSerial, st.DeviceSerials)
	}

	// Cockpit indexes map onto the owning scanner: index 2 is VHF, channel 0
	// of the shared scanner.
	ck := scannerCockpit{convs: d.convScans}
	if !ck.LockoutConventional(2) {
		t.Fatal("LockoutConventional(2) = false")
	}
	if snap := d.convScans[1].Snapshot(); !snap.Channels[0].LockedOut {
		t.Error("lockout of cockpit index 2 did not reach VHF on the shared scanner")
	}
	if snap := d.convScans[0].Snapshot(); snap.Channels[0].LockedOut || snap.Channels[1].LockedOut {
		t.Error("lockout of cockpit index 2 leaked onto the air-band scanner")
	}
	if ck.LockoutConventional(3) {
		t.Error("LockoutConventional(3) past the end = true")
	}
	// Manual tune lands on the shared scanner, after every configured channel.
	idx, ok := ck.ManualTune(api.ManualTuneRequest{FrequencyHz: 155_000_000})
	if !ok || idx != 3 {
		t.Fatalf("ManualTune = %d, %v; want index 3 on the shared scanner", idx, ok)
	}
	if ck.ClearManualTune(1) {
		t.Error("ClearManualTune(1) removed a channel from the air-band scanner")
	}
	if !ck.ClearManualTune(idx) {
		t.Error("ClearManualTune(3) = false")
	}
}
