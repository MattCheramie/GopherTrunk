package webserver

import (
	"bytes"
	"encoding/json"
	"math"
	"math/cmplx"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	s, err := New(Options{TempDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s.Handler()
}

func TestHandleAnalyzers(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	newTestServer(t).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/rfscope/analyzers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Analyzers []analyzerDTO `json:"analyzers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	have := map[string]bool{}
	for _, a := range body.Analyzers {
		have[a.Name] = true
	}
	for _, want := range []string{"hierarchy", "timeline", "timing", "topology", "entropy", "expert"} {
		if !have[want] {
			t.Fatalf("analyzers missing %q: %+v", want, body.Analyzers)
		}
	}
}

// gatedToneCapture builds an f32 capture with a tone at +offsetHz over noise.
func gatedToneCapture(rate, dur, onSec, offSec, offsetHz float64) []byte {
	n := int(rate * dur)
	iq := make([]complex64, n)
	rng := rand.New(rand.NewSource(1))
	onN, offN := int(onSec*rate), int(offSec*rate)
	w := 2 * math.Pi * offsetHz / rate
	for i := 0; i < n; i++ {
		s := complex(0.01*(rng.Float64()-0.5), 0.01*(rng.Float64()-0.5))
		if i >= onN && i < offN {
			s += cmplx.Exp(complex(0, w*float64(i)))
		}
		iq[i] = complex64(s)
	}
	return siglab.EncodeCapture(iq, siglab.FormatF32)
}

func TestHandleAnalyze(t *testing.T) {
	t.Parallel()
	capture := gatedToneCapture(240_000, 0.5, 0.1, 0.3, 30_000)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "scene.cfile")
	fw.Write(capture)
	mw.WriteField("format", "f32")
	mw.WriteField("sample_rate_hz", "240000")
	mw.WriteField("freq", "450000000")
	mw.WriteField("fft", "1024")
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/rfscope/analyze", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	newTestServer(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp analyzeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Scene == nil || len(resp.Scene.Bursts) < 1 {
		t.Fatalf("expected at least one burst, got %+v", resp.Scene)
	}
	if resp.Scene.CenterHz != 450_000_000 {
		t.Fatalf("scene center = %d", resp.Scene.CenterHz)
	}
}

func TestHandleAnalyzeRejectsMissingRate(t *testing.T) {
	t.Parallel()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "x.cfile")
	fw.Write(siglab.EncodeCapture(make([]complex64, 4096), siglab.FormatF32))
	mw.WriteField("format", "f32")
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/rfscope/analyze", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	newTestServer(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for missing sample rate, got %d", rec.Code)
	}
}

func TestSPAPlaceholderWhenNoAssets(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	newTestServer(t).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("SPA not bundled")) {
		t.Fatalf("want placeholder, got %q", rec.Body.String())
	}
}
