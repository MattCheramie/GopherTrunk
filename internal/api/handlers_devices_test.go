package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

type fakeDevices struct{ snap []sdr.SDRStatus }

func (f *fakeDevices) Snapshot() []sdr.SDRStatus { return f.snap }

func TestListDevicesEmptyWhenNoProvider(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/devices")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Devices []sdr.SDRStatus `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Devices) != 0 {
		t.Errorf("devices len = %d, want 0", len(body.Devices))
	}
}

func TestListDevicesReturnsSnapshot(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	devs := &fakeDevices{snap: []sdr.SDRStatus{
		{Driver: "rtlsdr", Serial: "AAA", TunerName: "R820T2", Role: "control",
			Attached: true, GainTenthDB: 496, PPM: 1, Gains: []int{0, 49, 100}},
		{Driver: "rtlsdr", Serial: "BBB", TunerName: "R820T2", Role: "voice",
			Attached: true, GainAuto: true},
	}}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Devices: devs})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/devices")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Devices []sdr.SDRStatus `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Devices) != 2 {
		t.Fatalf("devices len = %d, want 2", len(body.Devices))
	}
	if body.Devices[0].Serial != "AAA" || body.Devices[0].Role != "control" {
		t.Errorf("first device = %+v", body.Devices[0])
	}
	if body.Devices[1].Serial != "BBB" || !body.Devices[1].GainAuto {
		t.Errorf("second device = %+v", body.Devices[1])
	}
}
