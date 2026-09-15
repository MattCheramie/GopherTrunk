package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestSiglabCaptureCentreWithoutBandwidthIs400 (15 Sep): an operator asked
// for a 60 s grab "at an arbitrary centre" so both repeaters would land in
// one file, and the route quietly recorded the TUNER centre instead — the
// centre only ever applied to a narrowband slice, and a centre that arrived
// without a bandwidth was dropped on the floor. A centre that differs from
// the tuner centre without a bandwidth is now a 400 that says what to do;
// the tuner centre itself (what a full-band grab is at anyway) stays fine.
func TestSiglabCaptureCentreWithoutBandwidthIs400(t *testing.T) {
	prov := &fakeCaptureProvider{
		devices: []SpectrumDevice{{Serial: "X310", Driver: "soapyremote"}},
		iq:      make([]complex64, 4800),
		rate:    6_250_000,
		center:  441_700_000,
	}
	ts := newCaptureTestServer(t, prov)
	post := func(body string) (*http.Response, []byte) {
		t.Helper()
		resp, err := http.Post(ts.URL+"/api/v1/siglab/capture", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST capture: %v", err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp, b
	}
	resp, b := post(`{"serial":"X310","seconds":1,"format":"cs16","center_hz":442812500}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(b), "bandwidth_hz") {
		t.Fatalf("centre without bandwidth: status %d body %s, want 400 pointing at bandwidth_hz", resp.StatusCode, b)
	}
	resp, b = post(`{"serial":"X310","seconds":1,"format":"cs16","center_hz":441700000}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tuner-centre without bandwidth: status %d body %s, want 200", resp.StatusCode, b)
	}
	var cr captureResponse
	if err := json.Unmarshal(b, &cr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The staged capture now says where it is, so the list can show it.
	if cr.Capture.CenterHz != 441_700_000 {
		t.Errorf("full-band capture centre = %d, want the tuner centre 441700000", cr.Capture.CenterHz)
	}
}
