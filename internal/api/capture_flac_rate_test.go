package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

// TestSiglabCaptureFLACSliceAtOddRate is the 15 Sep operator report verbatim:
// an X310 streaming 6.25 MS/s, an 880 kHz slice requested as flac. The
// down-converter's capped L/M lands the slice at 880029 Hz — a rate FLAC's
// frame header cannot carry — and the capture aborted after 3908 samples
// with "unable to encode sample rate 880029". The staged flac must now
// decode back to the slice at its true rate. Fails against the old encoder
// (500 "encode flac frame").
func TestSiglabCaptureFLACSliceAtOddRate(t *testing.T) {
	const rate = 6_250_000
	iq := make([]complex64, rate/10) // 0.1 s is plenty: the fake ignores seconds
	for i := range iq {
		iq[i] = complex(float32(i%5)/5, float32(i%9)/9)
	}
	prov := &fakeCaptureProvider{
		devices: []SpectrumDevice{{Serial: "X310", Driver: "soapyremote"}},
		iq:      iq,
		rate:    rate,
		center:  441_700_000,
		chunks:  16,
	}
	ts := newCaptureTestServer(t, prov)
	body := `{"serial":"X310","seconds":60,"format":"flac","protocol":"dmr","center_hz":442812500,"bandwidth_hz":880000}`
	resp, err := http.Post(ts.URL+"/api/v1/siglab/capture", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST capture: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200 (%s)", resp.StatusCode, b)
	}
	var cr captureResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cr.Capture.SampleRateHz != 880_029 {
		t.Errorf("slice rate = %g Hz, want the capped-L/M 880029 the operator saw", cr.Capture.SampleRateHz)
	}
	if cr.Capture.CenterHz != 442_812_500 {
		t.Errorf("staged capture centre = %d, want the requested 442812500", cr.Capture.CenterHz)
	}
	// The staged file is a real flac at that rate.
	dl, err := http.Get(ts.URL + cr.DownloadURL)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer dl.Body.Close()
	data, _ := io.ReadAll(dl.Body)
	if !bytes.HasPrefix(data, []byte("fLaC")) {
		t.Fatalf("downloaded capture is not a flac stream (starts %q)", data[:4])
	}
	tmp := filepath.Join(t.TempDir(), "slice.flac")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, gotRate, err := baseband.ReadIQFLACSamples(tmp)
	if err != nil {
		t.Fatalf("decode staged flac: %v", err)
	}
	if gotRate != 880_029 || len(got) == 0 {
		t.Fatalf("decoded %d samples at %d Hz, want a non-empty 880029 Hz slice", len(got), gotRate)
	}
}

// TestSiglabCaptureFLACFullBandOverCeilingIs400: a flac full-band grab of a
// 2.4 MS/s tuner cannot exist (20-bit STREAMINFO rate); the route says so
// before pinning the tuner instead of aborting the capture after the wait.
func TestSiglabCaptureFLACFullBandOverCeilingIs400(t *testing.T) {
	prov := &fakeCaptureProvider{
		devices: []SpectrumDevice{{Serial: "SDR1", Driver: "mock"}},
		iq:      make([]complex64, 4800),
		rate:    2_400_000,
		center:  460_000_000,
	}
	ts := newCaptureTestServer(t, prov)
	resp, err := http.Post(ts.URL+"/api/v1/siglab/capture", "application/json",
		bytes.NewBufferString(`{"serial":"SDR1","seconds":1,"format":"flac"}`))
	if err != nil {
		t.Fatalf("POST capture: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(b), "1048575") {
		t.Fatalf("status = %d body %s, want 400 naming the 1048575 Hz ceiling", resp.StatusCode, b)
	}
}
