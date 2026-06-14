package api

import (
	"bytes"
	"encoding/json"
	"math"
	"math/cmplx"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

func newSiglabTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr:           "127.0.0.1:0",
		Bus:            bus,
		AllowMutations: true,
		Siglab:         SiglabOptions{Enabled: true, TempDir: t.TempDir()},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestSiglabProtocols(t *testing.T) {
	ts := newSiglabTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/siglab/protocols")
	if err != nil {
		t.Fatalf("GET protocols: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Protocols []string `json:"protocols"`
		Fixtures  []string `json:"fixtures"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Protocols) == 0 || len(body.Fixtures) == 0 {
		t.Errorf("expected non-empty protocols/fixtures, got %+v", body)
	}
}

// TestSiglabSynthesizeRunExport drives the full happy path: synthesize a
// clean P25 capture (staged), run the engine over it with IQ capture, poll the
// job to completion, then assert lock + export + IQ retrieval.
func TestSiglabSynthesizeRunExport(t *testing.T) {
	ts := newSiglabTestServer(t)

	// Synthesize → stage a capture.
	synthBody := `{"protocol":"p25","format":"f32"}`
	resp, err := http.Post(ts.URL+"/api/v1/siglab/synthesize", "application/json", bytes.NewBufferString(synthBody))
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("synthesize status = %d, want 200", resp.StatusCode)
	}
	var synth struct {
		Capture siglabCaptureDTO `json:"capture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&synth); err != nil {
		t.Fatalf("decode synth: %v", err)
	}
	if synth.Capture.ID == "" {
		t.Fatal("expected a staged capture id")
	}

	// Run the engine over the staged capture.
	runBody := `{"capture_id":"` + synth.Capture.ID + `","config":{"protocol":"p25","collect_iq_diag":true,"capture_iq":true}}`
	runResp, err := http.Post(ts.URL+"/api/v1/siglab/run", "application/json", bytes.NewBufferString(runBody))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer runResp.Body.Close()
	if runResp.StatusCode != http.StatusAccepted {
		t.Fatalf("run status = %d, want 202", runResp.StatusCode)
	}
	var run struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(runResp.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}

	// Poll the job to completion.
	var job siglabJobDTO
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		r, err := http.Get(ts.URL + "/api/v1/siglab/jobs/" + run.JobID)
		if err != nil {
			t.Fatalf("poll: %v", err)
		}
		_ = json.NewDecoder(r.Body).Decode(&job)
		r.Body.Close()
		if job.State == "done" || job.State == "error" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if job.State != "done" {
		t.Fatalf("job state = %q (err=%q), want done", job.State, job.Error)
	}
	if job.Result == nil || !job.Result.Locked {
		t.Fatalf("expected a locking result; got %+v", job.Result)
	}
	if !job.HasIQ {
		t.Error("expected captured IQ to be available")
	}

	// Export JSON.
	expResp, err := http.Get(ts.URL + "/api/v1/siglab/jobs/" + run.JobID + "/export?format=json")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	defer expResp.Body.Close()
	var exported siglab.Result
	if err := json.NewDecoder(expResp.Body).Decode(&exported); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if exported.Protocol != "p25" {
		t.Errorf("exported protocol = %q, want p25", exported.Protocol)
	}

	// Retrieve decimated IQ.
	iqResp, err := http.Get(ts.URL + "/api/v1/siglab/jobs/" + run.JobID + "/iq?kind=iq")
	if err != nil {
		t.Fatalf("iq: %v", err)
	}
	defer iqResp.Body.Close()
	if iqResp.StatusCode != http.StatusOK {
		t.Fatalf("iq status = %d, want 200", iqResp.StatusCode)
	}
	var iq struct {
		DecimatedIQ []siglab.IQPoint `json:"decimated_iq"`
	}
	if err := json.NewDecoder(iqResp.Body).Decode(&iq); err != nil {
		t.Fatalf("decode iq: %v", err)
	}
	if len(iq.DecimatedIQ) == 0 {
		t.Error("expected decimated IQ points")
	}
}

func TestSiglabRunMissingCapture(t *testing.T) {
	ts := newSiglabTestServer(t)
	body := `{"capture_id":"nope","config":{"protocol":"p25","sample_rate_hz":48000}}`
	resp, err := http.Post(ts.URL+"/api/v1/siglab/run", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestSiglabWideband uploads a synthetic two-carrier wideband capture and
// surveys it, asserting the endpoint returns the averaged band spectrum (for
// plotting) and detects both planted carriers.
func TestSiglabWideband(t *testing.T) {
	ts := newSiglabTestServer(t)

	// Two complex tones over a noise floor in a 2 MS/s band, ~400 Hz off the
	// 12.5 kHz raster (mirrors the siglab wideband detector test).
	const rate = 2_000_000.0
	const center = 441_000_000
	n := 1 << 16
	iq := make([]complex64, n)
	seed := uint64(1)
	rnd := func() float64 { // tiny LCG → ~uniform in [-0.5,0.5)
		seed = seed*6364136223846793005 + 1442695040888963407
		return float64(seed>>11)/float64(1<<53) - 0.5
	}
	for i := range iq {
		iq[i] = complex(float32(rnd()*0.04), float32(rnd()*0.04))
	}
	addTone := func(freqHz, amp float64) {
		step := cmplx.Exp(complex(0, 2*math.Pi*freqHz/rate))
		ph := complex(1, 0)
		for i := range iq {
			iq[i] += complex64(complex(amp, 0) * ph)
			ph *= step
		}
	}
	addTone(300_000+400, 1.0)
	addTone(-250_000+400, 0.7)

	// Upload the capture as raw f32 via the multipart endpoint.
	raw := siglab.EncodeCapture(iq, siglab.FormatF32)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "wide.cf32")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(raw); err != nil {
		t.Fatalf("write part: %v", err)
	}
	_ = mw.WriteField("format", "f32")
	_ = mw.WriteField("sample_rate_hz", "2000000")
	_ = mw.Close()

	upResp, err := http.Post(ts.URL+"/api/v1/siglab/captures", mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer upResp.Body.Close()
	if upResp.StatusCode != http.StatusOK {
		t.Fatalf("upload status = %d, want 200", upResp.StatusCode)
	}
	var cap siglabCaptureDTO
	if err := json.NewDecoder(upResp.Body).Decode(&cap); err != nil {
		t.Fatalf("decode upload: %v", err)
	}

	// Survey the staged capture.
	body := `{"capture_id":"` + cap.ID + `","center_hz":441000000}`
	wbResp, err := http.Post(ts.URL+"/api/v1/siglab/wideband", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("wideband: %v", err)
	}
	defer wbResp.Body.Close()
	if wbResp.StatusCode != http.StatusOK {
		t.Fatalf("wideband status = %d, want 200", wbResp.StatusCode)
	}
	var got siglab.WidebandResult
	if err := json.NewDecoder(wbResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode wideband: %v", err)
	}
	if got.Spectrum == nil || len(got.Spectrum.Bins) == 0 {
		t.Fatal("expected a spectrum with bins for plotting")
	}
	if got.Spectrum.CenterHz != center {
		t.Errorf("spectrum center = %d, want %d", got.Spectrum.CenterHz, center)
	}
	if got.DetectedCount < 2 {
		t.Errorf("detected_count = %d, want >= 2", got.DetectedCount)
	}
}
