package sdr

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestPoolDedicatedHintIsNeverControlOrVoice is the #1316 regression: a
// dongle pinned to a single-channel decoder (Hint.Dedicated) must never be
// handed out by role. With three `auto` dongles the first used to become
// the control SDR and the rest voice SDRs — including the APRS and ADS-B
// sticks, which the manual-tune scanner, a live hunt and rigctld then
// retuned.
func TestPoolDedicatedHintIsNeverControlOrVoice(t *testing.T) {
	drv := &fakeDriver{name: "fake-dedicated", infos: []Info{
		{Driver: "fake-dedicated", Index: 0, Serial: "00000001"}, // adsb
		{Driver: "fake-dedicated", Index: 1, Serial: "00000002"},
		{Driver: "fake-dedicated", Index: 2, Serial: "54300989"}, // aprs
		{Driver: "fake-dedicated", Index: 3, Serial: "00000004"},
	}}
	registryMu.Lock()
	registry["fake-dedicated"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-dedicated")
		registryMu.Unlock()
	})

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := NewPool(log)
	if err := p.OpenWith(PoolOpenOptions{
		Hints: []Hint{
			{Serial: "00000001", Role: RoleControl, Dedicated: true}, // the reporter's workaround, now overridden
			{Serial: "00000002", Role: RoleAuto},
			{Serial: "54300989", Role: RoleAuto, Dedicated: true},
			{Serial: "00000004", Role: RoleAuto},
		},
		Strict: true,
	}); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	roles := map[string]Role{}
	for _, e := range p.Entries() {
		roles[e.Info.Serial] = e.Role
	}
	if roles["00000001"] != RoleDedicated || roles["54300989"] != RoleDedicated {
		t.Errorf("pinned dongles: %v / %v, want dedicated", roles["00000001"], roles["54300989"])
	}
	// The control slot goes to the first NON-dedicated auto dongle.
	if roles["00000002"] != RoleControl || roles["00000004"] != RoleVoice {
		t.Errorf("free dongles: 00000002=%v 00000004=%v, want control / voice", roles["00000002"], roles["00000004"])
	}
	if c := p.FirstByRole(RoleControl); c == nil || c.Info.Serial != "00000002" {
		t.Errorf("FirstByRole(control) = %v, want 00000002", c)
	}
	for _, v := range p.AllByRole(RoleVoice) {
		if v.Info.Serial == "00000001" || v.Info.Serial == "54300989" {
			t.Errorf("AllByRole(voice) hands out the pinned dongle %s", v.Info.Serial)
		}
	}
	if len(p.AllByRole(RoleDedicated)) != 2 {
		t.Errorf("AllByRole(dedicated) = %d, want 2", len(p.AllByRole(RoleDedicated)))
	}
	if !strings.Contains(buf.String(), "configured role is ignored") {
		t.Errorf("an explicit role on a pinned dongle must be logged as ignored; log: %s", buf.String())
	}
	if RoleDedicated.String() != "dedicated" || ParseRole("dedicated") != RoleDedicated {
		t.Errorf("role round trip: %q / %v", RoleDedicated.String(), ParseRole("dedicated"))
	}
}
