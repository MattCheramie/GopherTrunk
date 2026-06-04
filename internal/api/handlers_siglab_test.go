package api

import (
	"bytes"
	"encoding/json"
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
