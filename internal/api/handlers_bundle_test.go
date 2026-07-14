package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
)

// writeTestBundle packs a minimal bundle on disk and returns its path.
func writeTestBundle(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "case.gtb.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	w, err := gtbundle.NewWriter(f, gtbundle.WriterOptions{Name: "case", CaptureIntent: gtbundle.IntentCCMap})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	sys := &hunt.DiscoveredSystem{Name: "Web Case", Protocol: "p25"}
	if _, err := w.AddJSON(gtbundle.RoleMappingSystem, "system.json", sys, "hunt"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return path
}

func TestHandleBundleInfo(t *testing.T) {
	dir := t.TempDir()
	path := writeTestBundle(t, dir)
	s := &Server{bundleRoot: dir}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/info?path="+path, nil)
	rec := httptest.NewRecorder()
	s.handleBundleInfo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var dto bundleInfoDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dto.Name != "case" || dto.Format != gtbundle.FormatID {
		t.Errorf("dto = %+v", dto)
	}
	if dto.Counts["mapping-system"] != 1 {
		t.Errorf("role counts = %v", dto.Counts)
	}
}

func TestHandleBundleVerify(t *testing.T) {
	dir := t.TempDir()
	path := writeTestBundle(t, dir)
	s := &Server{bundleRoot: dir}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundle/verify",
		strings.NewReader(`{"path":"`+path+`"}`))
	rec := httptest.NewRecorder()
	s.handleBundleVerify(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var out bundleVerifyDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.OK || out.Failures != 0 {
		t.Errorf("verify dto = %+v", out)
	}
}

// TestHandleBundleInfoRejectsOutsideRoot confirms the bundleRoot confinement.
func TestHandleBundleInfoRejectsOutsideRoot(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	path := writeTestBundle(t, other) // bundle lives outside the configured root
	s := &Server{bundleRoot: dir}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/info?path="+path, nil)
	rec := httptest.NewRecorder()
	s.handleBundleInfo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a path outside bundleRoot", rec.Code)
	}
}

// TestHandleBundleInfoRejectsNonBundleSuffix confirms arbitrary host files
// cannot be read through the endpoint.
func TestHandleBundleInfoRejectsNonBundleSuffix(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	os.WriteFile(secret, []byte("nope"), 0o644)
	s := &Server{bundleRoot: dir}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundle/info?path="+secret, nil)
	rec := httptest.NewRecorder()
	s.handleBundleInfo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a non-.gtb path", rec.Code)
	}
}
