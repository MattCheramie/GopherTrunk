package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// groupSpectrumProvider adds an sdr.groups entry to the fake provider.
type groupSpectrumProvider struct {
	fakeSpectrumProvider
	groups []SpectrumDevice
}

func (g *groupSpectrumProvider) Groups() []SpectrumDevice { return g.groups }

// TestSpectrumGroupDevice pins the group's API surface: listed only when the
// client asks (?groups=1, so the per-device panels never open a symbol
// receiver on it), streamable, never tunable, and refused by the
// single-stream diag views before the WS upgrade.
func TestSpectrumGroupDevice(t *testing.T) {
	prov := &groupSpectrumProvider{
		fakeSpectrumProvider: fakeSpectrumProvider{
			devices: []SpectrumDevice{{Serial: "M1", Driver: "rtlsdr", Role: "wideband"}},
			frames:  []SpectrumFrame{{CenterHz: 854_000_000, SampleRateHz: 6_000_000, Bins: []float32{-100, -90}}},
		},
		groups: []SpectrumDevice{{Serial: "group:south", Driver: "group", Role: "group", Group: true, Members: []string{"M1", "M2"}}},
	}
	ts := newSpectrumTestServer(t, prov)

	list := func(q string) []SpectrumDevice {
		t.Helper()
		resp, err := http.Get(ts.URL + "/api/v1/spectrum/devices" + q)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var got []SpectrumDevice
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := list(""); len(got) != 1 {
		t.Errorf("default list = %+v, want members only", got)
	}
	if got := list("?groups=1"); len(got) != 2 || !got[1].Group {
		t.Errorf("?groups=1 list = %+v, want the group appended", got)
	}

	resp, err := http.Post(ts.URL+"/api/v1/spectrum/devices/group:south/tune", "application/json", bytes.NewBufferString(`{"center_hz":852000000}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || len(prov.tuned) != 0 {
		t.Errorf("tune group: status %d, tuned %v", resp.StatusCode, prov.tuned)
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/spectrum/stream?device=group:south"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("group stream dial: %v", err)
	}
	var f SpectrumFrame
	if err := conn.ReadJSON(&f); err != nil || f.CenterHz != 854_000_000 {
		t.Errorf("group stream frame = %+v, %v", f, err)
	}
	conn.Close()

	// The single-stream views (symbols / diag / mixer) gate on
	// rejectUnknownDevice, which knows only real devices: a group id 404s
	// there before any upgrade.
	srv := &Server{spectrum: prov}
	rec := httptest.NewRecorder()
	if !srv.rejectUnknownDevice(rec, "group:south") || rec.Code != http.StatusNotFound {
		t.Errorf("rejectUnknownDevice(group) = %d, want a 404", rec.Code)
	}
}
