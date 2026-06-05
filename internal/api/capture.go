package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// maxCaptureSeconds bounds a single live capture so an operator can't pin a
// tuner (and pile up hundreds of MB of staged IQ) with one request.
const maxCaptureSeconds = 30

// CaptureProvider taps a live SDR for a fixed-length raw-IQ capture. The
// daemon (cmd/gophertrunk) implements it over its iqtap.Broker map; nil keeps
// the POST /api/v1/siglab/capture route returning 503 so a build without SDRs
// (or the offline `siglab serve` console) doesn't pretend to have a tuner to
// record from. Kept narrow — and reusing the SpectrumDevice DTO — so the api
// package stays free of dependencies on internal/sdr, mirroring SpectrumProvider.
type CaptureProvider interface {
	// Devices returns the SDRs that can be captured from.
	Devices() []SpectrumDevice
	// Capture records seconds worth of raw IQ from the named device and
	// returns the complex samples plus the device's current sample rate and
	// centre frequency. ctx cancels an in-flight capture.
	Capture(ctx context.Context, serial string, seconds int) (iq []complex64, sampleRateHz, centerHz uint32, err error)
}

// captureRequest is the body of POST /api/v1/siglab/capture.
type captureRequest struct {
	Serial   string `json:"serial"`
	Seconds  int    `json:"seconds"`
	Format   string `json:"format"`
	Protocol string `json:"protocol,omitempty"`
	Source   string `json:"source,omitempty"`
}

// captureResponse is returned by a successful capture: the staged capture
// (runnable/identifiable immediately), the metadata sidecar describing it, and
// a relative URL to download the raw .cfile.
type captureResponse struct {
	Capture     siglabCaptureDTO `json:"capture"`
	Metadata    *siglab.Metadata `json:"metadata"`
	DownloadURL string           `json:"download_url"`
}

// handleSiglabCaptureDevices answers GET /api/v1/siglab/capture/devices with
// the SDRs available to record from. 503 when no CaptureProvider is wired.
func (s *Server) handleSiglabCaptureDevices(w http.ResponseWriter, r *http.Request) {
	if s.capture == nil {
		s.writeError(w, http.StatusServiceUnavailable, "siglab: live capture not available (no SDR)")
		return
	}
	writeJSON(w, http.StatusOK, s.capture.Devices())
}

// handleSiglabCapture records a fixed-length raw-IQ capture off a live SDR,
// stages it into the siglab capture store (so it can be run/identified
// immediately), writes a metadata sidecar next to it, and returns the staged
// capture DTO plus a download URL for the raw .cfile. This is the runtime
// equivalent of the `gophertrunk capture` subcommand.
func (s *Server) handleSiglabCapture(w http.ResponseWriter, r *http.Request) {
	if s.capture == nil {
		s.writeError(w, http.StatusServiceUnavailable, "siglab: live capture not available (no SDR)")
		return
	}
	var req captureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	if req.Serial == "" {
		s.writeError(w, http.StatusBadRequest, "siglab: serial is required")
		return
	}
	if req.Seconds <= 0 || req.Seconds > maxCaptureSeconds {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("siglab: seconds must be 1..%d", maxCaptureSeconds))
		return
	}
	format, err := siglab.ParseSampleFormat(req.Format)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}

	iq, sampleRateHz, centerHz, err := s.capture.Capture(r.Context(), req.Serial, req.Seconds)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "siglab: capture: "+err.Error())
		return
	}
	if len(iq) == 0 {
		s.writeError(w, http.StatusBadGateway, "siglab: capture produced no samples")
		return
	}

	id := randomID(16)
	path := s.siglab.newCapturePath(id)
	if err := os.WriteFile(path, siglab.EncodeCapture(iq, format), 0o644); err != nil {
		s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+err.Error())
		return
	}

	meta := &siglab.Metadata{
		Protocol:     req.Protocol,
		Source:       req.Source,
		SampleRateHz: float64(sampleRateHz),
		CenterFreqHz: centerHz,
		Format:       format.String(),
	}
	// Best-effort sidecar at the path siglab.DiscoverMetadata probes
	// (<stem>.metadata.json) so the staged file is a drop-in fixture.
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".metadata.json"
	if err := siglab.WriteMetadata(metaPath, meta); err != nil {
		s.log.Warn("api: siglab capture metadata write failed", "err", err)
	}

	c := &siglabCapture{
		ID:           id,
		Name:         captureName(req.Serial, centerHz),
		Path:         path,
		Format:       format,
		SampleRateHz: float64(sampleRateHz),
		Size:         int64(len(iq)) * int64(bytesPerSample(format)),
		Created:      time.Now(),
	}
	s.siglab.putCapture(c)

	writeJSON(w, http.StatusOK, captureResponse{
		Capture:     captureDTO(c),
		Metadata:    meta,
		DownloadURL: fmt.Sprintf("/api/v1/siglab/captures/%s/download", id),
	})
}

// handleSiglabCaptureDownload streams a staged capture's raw bytes as a file
// download. Works for any staged capture (uploaded, synthesized, or live), but
// is wired primarily so a live capture can be saved to the operator's disk.
func (s *Server) handleSiglabCaptureDownload(w http.ResponseWriter, r *http.Request) {
	c, ok := s.siglab.getCapture(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: capture not found")
		return
	}
	f, err := os.Open(c.Path)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "siglab: capture file missing")
		return
	}
	defer f.Close()

	ext := "cfile"
	if c.Format == siglab.FormatU8 {
		ext = "bin"
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", c.ID, ext))
	if _, err := io.Copy(w, f); err != nil {
		s.log.Warn("api: siglab capture download failed", "err", err)
	}
}

// captureName builds a friendly staged-capture name from the device + tuning.
func captureName(serial string, centerHz uint32) string {
	if centerHz > 0 {
		return fmt.Sprintf("capture-%s-%.3fMHz", serial, float64(centerHz)/1e6)
	}
	return "capture-" + serial
}

// bytesPerSample returns the on-disk size of one IQ sample in the format.
func bytesPerSample(format siglab.SampleFormat) int {
	if format == siglab.FormatF32 {
		return 8
	}
	return 2
}
