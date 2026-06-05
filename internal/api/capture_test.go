package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// fakeCaptureProvider is an in-memory CaptureProvider for handler tests.
type fakeCaptureProvider struct {
	devices []SpectrumDevice
	iq      []complex64
	rate    uint32
	center  uint32
	err     error
}

func (f *fakeCaptureProvider) Devices() []SpectrumDevice { return f.devices }

func (f *fakeCaptureProvider) Capture(_ context.Context, _ string, _ int) ([]complex64, uint32, uint32, error) {
	if f.err != nil {
		return nil, 0, 0, f.err
	}
	return f.iq, f.rate, f.center, nil
}

func newCaptureTestServer(t *testing.T, prov CaptureProvider) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr:           "127.0.0.1:0",
		Bus:            bus,
		AllowMutations: true,
		Siglab:         SiglabOptions{Enabled: true, TempDir: t.TempDir()},
		Capture:        prov,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestSiglabCaptureUnavailableWhenNotWired(t *testing.T) {
	ts := newSiglabTestServer(t) // no Capture provider
	resp, err := http.Post(ts.URL+"/api/v1/siglab/capture", "application/json",
		bytes.NewBufferString(`{"serial":"x","seconds":1,"format":"f32"}`))
	if err != nil {
		t.Fatalf("POST capture: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func TestSiglabCaptureStagesAndDownloads(t *testing.T) {
	iq := make([]complex64, 4800)
	for i := range iq {
		iq[i] = complex(float32(i%7)/7, float32(i%3)/3)
	}
	prov := &fakeCaptureProvider{
		devices: []SpectrumDevice{{Serial: "SDR1", Driver: "mock"}},
		iq:      iq,
		rate:    2_400_000,
		center:  460_000_000,
	}
	ts := newCaptureTestServer(t, prov)

	// Device picker lists the fake device.
	dResp, err := http.Get(ts.URL + "/api/v1/siglab/capture/devices")
	if err != nil {
		t.Fatalf("GET capture/devices: %v", err)
	}
	defer dResp.Body.Close()
	var devices []SpectrumDevice
	if err := json.NewDecoder(dResp.Body).Decode(&devices); err != nil {
		t.Fatalf("decode devices: %v", err)
	}
	if len(devices) != 1 || devices[0].Serial != "SDR1" {
		t.Fatalf("devices = %+v, want one SDR1", devices)
	}

	// Capture → staged DTO + metadata + download URL.
	cResp, err := http.Post(ts.URL+"/api/v1/siglab/capture", "application/json",
		bytes.NewBufferString(`{"serial":"SDR1","seconds":2,"format":"f32","protocol":"p25"}`))
	if err != nil {
		t.Fatalf("POST capture: %v", err)
	}
	defer cResp.Body.Close()
	if cResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(cResp.Body)
		t.Fatalf("status = %d, want 200 (%s)", cResp.StatusCode, b)
	}
	var cr captureResponse
	if err := json.NewDecoder(cResp.Body).Decode(&cr); err != nil {
		t.Fatalf("decode capture response: %v", err)
	}
	if cr.Capture.ID == "" || cr.Capture.SampleRateHz != 2_400_000 {
		t.Fatalf("capture DTO = %+v, want id + 2.4M rate", cr.Capture)
	}
	if cr.Metadata == nil || cr.Metadata.Protocol != "p25" || cr.Metadata.CenterFreqHz != 460_000_000 {
		t.Fatalf("metadata = %+v, want p25 @ 460 MHz", cr.Metadata)
	}
	wantBytes := int64(len(iq)) * 8 // f32 = 8 bytes/sample
	if cr.Capture.Size != wantBytes {
		t.Fatalf("capture size = %d, want %d", cr.Capture.Size, wantBytes)
	}

	// Download streams the raw bytes as an attachment of the right length.
	dlResp, err := http.Get(ts.URL + cr.DownloadURL)
	if err != nil {
		t.Fatalf("GET download: %v", err)
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d, want 200", dlResp.StatusCode)
	}
	if cd := dlResp.Header.Get("Content-Disposition"); cd == "" {
		t.Errorf("missing Content-Disposition header")
	}
	got, _ := io.ReadAll(dlResp.Body)
	if int64(len(got)) != wantBytes {
		t.Errorf("downloaded %d bytes, want %d", len(got), wantBytes)
	}
}

func TestSiglabCaptureRejectsBadSeconds(t *testing.T) {
	prov := &fakeCaptureProvider{iq: []complex64{complex(1, 0)}, rate: 48000}
	ts := newCaptureTestServer(t, prov)
	for _, body := range []string{
		`{"serial":"SDR1","seconds":0,"format":"f32"}`,
		`{"serial":"SDR1","seconds":9999,"format":"f32"}`,
		`{"seconds":1,"format":"f32"}`, // missing serial
	} {
		resp, err := http.Post(ts.URL+"/api/v1/siglab/capture", "application/json",
			bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST capture: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("body %s → status %d, want 400", body, resp.StatusCode)
		}
	}
}
