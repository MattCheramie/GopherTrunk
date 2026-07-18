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
// tuner indefinitely with one request. The tighter real bound on memory is
// maxCaptureIQBytes below (peak RAM scales with seconds × sample rate, which
// varies ~5x across SDRs); this is a coarse ceiling on wall-clock duration.
const maxCaptureSeconds = 120

// maxCaptureIQBytes caps the raw wideband IQ a single capture may buffer. The
// whole capture is collected into RAM as complex64 (8 bytes/sample) and then
// EncodeCapture allocates a second copy, so peak resident memory is roughly
// twice this. 1 GiB of complex64 (~134 M samples) keeps that peak to a couple
// of GB even on a modest host. A high-rate grab that would exceed it is
// rejected up front (before the tuner is pinned) with an actionable error
// instead of swapping/OOM-ing the daemon. The guard is on the raw collection,
// so it also bounds a narrowband slice (which buffers the full band first).
const maxCaptureIQBytes = 1 << 30

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
//
// CenterHz + BandwidthHz (both optional) request a narrowband slice carved from
// the tuner's current wideband stream: the channel at CenterHz is shifted to DC
// and decimated to ~BandwidthHz, so the staged file is a small baseband
// recording instead of a full-rate wideband grab. The tuner is NOT retuned (the
// slice is extracted from what it is already streaming), so CenterHz must fall
// inside the tuned span. BandwidthHz == 0 keeps the legacy full-band behaviour.
type captureRequest struct {
	Serial      string `json:"serial"`
	Seconds     int    `json:"seconds"`
	Format      string `json:"format"`
	Protocol    string `json:"protocol,omitempty"`
	Source      string `json:"source,omitempty"`
	CenterHz    uint32 `json:"center_hz,omitempty"`
	BandwidthHz uint32 `json:"bandwidth_hz,omitempty"`
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
	// A live capture spends `seconds` of real time collecting IQ before the
	// handler writes anything, then stages a large file — both can exceed the
	// server-level 30s WriteTimeout (server.go), which would silently tear down
	// the 200 mid-write and leave the UI stuck on "Capturing…". Disable the
	// write deadline per-request, exactly as the SSE and audio-stream handlers
	// do (see sse.go / handlers_audio_stream.go).
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
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

	// Reject an over-budget grab before pinning the tuner. The device's current
	// sample rate is known from the picker, so the raw-IQ footprint
	// (seconds × rate × 8 bytes for complex64) can be estimated up front. When
	// the rate is unknown (device not streaming yet) the estimate is skipped and
	// maxCaptureSeconds is the only bound.
	if rate := captureDeviceRate(s.capture.Devices(), req.Serial); rate > 0 {
		estIQBytes := int64(req.Seconds) * int64(rate) * 8
		if estIQBytes > maxCaptureIQBytes {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf(
				"siglab: a %ds capture at %.3f MS/s needs ~%d MiB of IQ, over the %d MiB budget — "+
					"reduce seconds or request a narrowband slice (center_hz + bandwidth_hz)",
				req.Seconds, float64(rate)/1e6, estIQBytes>>20, int64(maxCaptureIQBytes)>>20))
			return
		}
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

	// Optional narrowband slice: carve the channel at CenterHz down to
	// ~BandwidthHz from the tuner's current wideband stream (no retune), so the
	// staged file is small enough to hand to an analysis tool.
	if req.BandwidthHz > 0 {
		iq, sampleRateHz, centerHz, err = narrowband(iq, sampleRateHz, centerHz, req.CenterHz, req.BandwidthHz)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
			return
		}
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
	switch c.Format {
	case siglab.FormatU8:
		ext = "bin"
	case siglab.FormatS16:
		ext = "raw"
	case siglab.FormatWAV:
		ext = "wav"
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", c.ID, ext))
	if _, err := io.Copy(w, f); err != nil {
		s.log.Warn("api: siglab capture download failed", "err", err)
	}
}

// narrowband carves the channel at wantCenterHz (±wantBWHz/2) out of the
// wideband IQ the tuner is currently streaming, decimating it to a baseband
// stream and returning the slice, its new sample rate, and its new centre
// (== wantCenterHz, or the tuner centre when wantCenterHz is 0). The tuner is
// not retuned — the slice is extracted from the live stream — so the requested
// channel must fit wholly inside the tuned span [tunerCentre ± rate/2].
func narrowband(iq []complex64, rateHz, tunerCenterHz, wantCenterHz, wantBWHz uint32) ([]complex64, uint32, uint32, error) {
	center := wantCenterHz
	if center == 0 {
		center = tunerCenterHz
	}
	offsetHz := int64(center) - int64(tunerCenterHz)
	half := int64(rateHz) / 2
	if abs64(offsetHz)+int64(wantBWHz)/2 > half {
		return nil, 0, 0, fmt.Errorf(
			"center_hz %d + bandwidth_hz %d falls outside the tuner's current span %.3f–%.3f MHz "+
				"(centre %.3f MHz, rate %.3f MS/s); a capture extracts a channel from the live stream "+
				"without retuning, so pick a centre/bandwidth inside the span",
			center, wantBWHz,
			float64(int64(tunerCenterHz)-half)/1e6, float64(int64(tunerCenterHz)+half)/1e6,
			float64(tunerCenterHz)/1e6, float64(rateHz)/1e6)
	}
	nb, outRate := siglab.Downconvert(iq, float64(rateHz), float64(offsetHz), float64(wantBWHz))
	if len(nb) == 0 {
		return nil, 0, 0, fmt.Errorf("bandwidth_hz %d is too small for a %d-sample capture (no output samples)", wantBWHz, len(iq))
	}
	return nb, uint32(outRate + 0.5), center, nil
}

// abs64 is the absolute value of a signed 64-bit integer.
func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// captureDeviceRate returns the current sample rate of the device with the
// given serial from the capture picker's device list, or 0 when the device is
// unknown or has no rate yet (not streaming).
func captureDeviceRate(devices []SpectrumDevice, serial string) uint32 {
	for _, d := range devices {
		if d.Serial == serial {
			return d.SampleRateHz
		}
	}
	return 0
}

// captureName builds a friendly staged-capture name from the device + tuning.
func captureName(serial string, centerHz uint32) string {
	if centerHz > 0 {
		return fmt.Sprintf("capture-%s-%.3fMHz", serial, float64(centerHz)/1e6)
	}
	return "capture-" + serial
}

// bytesPerSample returns the on-disk size of one IQ sample in the format. It
// defers to the format's own decoder width (f32 → 8, cs16/wav → 4, u8 → 2) so
// staged-capture sizes stay correct as formats are added.
func bytesPerSample(format siglab.SampleFormat) int {
	_, n := format.Decoder()
	return n
}
