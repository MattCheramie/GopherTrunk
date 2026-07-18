package api

import (
	"bufio"
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
// tuner indefinitely with one request. The capture streams encoded IQ straight
// to disk (peak RAM is one chunk, not the whole grab), so duration is bounded by
// disk — see maxCaptureIQBytes — rather than memory; this is a coarse ceiling on
// wall-clock duration.
const maxCaptureSeconds = 120

// maxCaptureIQBytes caps the on-disk size a single capture may stage. Because
// the capture now streams encoded IQ to disk one chunk at a time (see
// siglab.CaptureWriter), memory is no longer the constraint — this bounds the
// staged file so one request can't fill the disk. 1 GiB is ~13 s of f32 at
// 10 MS/s or ~55 s at 2.4 MS/s; a grab that would exceed it is rejected up front
// (before the tuner is pinned) with an actionable error suggesting a shorter
// duration or a narrowband slice.
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
	// CaptureStream records seconds worth of raw IQ from the named device,
	// invoking sink for each chunk of complex samples as it arrives (never
	// buffering the whole capture), and returns the device's current sample
	// rate and centre frequency. A sink error aborts the capture and is
	// returned. ctx cancels an in-flight capture.
	CaptureStream(ctx context.Context, serial string, seconds int, sink func(iq []complex64) error) (sampleRateHz, centerHz uint32, err error)
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

	rate, centerHz := captureDeviceRateCenter(s.capture.Devices(), req.Serial)

	// Reject an over-budget grab before pinning the tuner. Streaming keeps RAM
	// bounded to one chunk, so this bounds the on-disk file size (seconds × rate ×
	// bytes-per-sample), not memory. Skipped when the rate is unknown (device not
	// streaming yet) — then maxCaptureSeconds is the only bound.
	if rate > 0 {
		estBytes := int64(req.Seconds) * int64(rate) * int64(bytesPerSample(format))
		if estBytes > maxCaptureIQBytes {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf(
				"siglab: a %ds capture at %.3f MS/s would stage ~%d MiB, over the %d MiB budget — "+
					"reduce seconds or request a narrowband slice (center_hz + bandwidth_hz)",
				req.Seconds, float64(rate)/1e6, estBytes>>20, int64(maxCaptureIQBytes)>>20))
			return
		}
	}

	// Optional narrowband slice: validate the requested channel fits the tuner's
	// current span and build a streaming down-converter up front, so an
	// out-of-span request 400s before the tuner is pinned. The slice is carved
	// from the live stream chunk-by-chunk during capture — no wideband buffer.
	var ddc *siglab.StreamDownconverter
	outRate, outCenter := rate, centerHz
	if req.BandwidthHz > 0 {
		if rate == 0 {
			s.writeError(w, http.StatusBadGateway,
				"siglab: capture: device has no sample rate yet (start or resume a scan on this SDR)")
			return
		}
		offsetHz, center, err := narrowbandParams(rate, centerHz, req.CenterHz, req.BandwidthHz)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
			return
		}
		ddc = siglab.NewStreamDownconverter(float64(rate), float64(offsetHz), float64(req.BandwidthHz))
		outRate = uint32(ddc.OutRateHz() + 0.5)
		outCenter = center
	}

	// Stream encoded IQ straight to the staged file: peak memory is one chunk
	// (plus the DDC's FIR state for a narrowband slice), independent of duration.
	id := randomID(16)
	path := s.siglab.newCapturePath(id)
	f, err := os.Create(path)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+err.Error())
		return
	}
	bw := bufio.NewWriterSize(f, 1<<20)
	enc := siglab.NewCaptureWriter(bw, format)
	sink := func(chunk []complex64) error {
		if ddc != nil {
			chunk = ddc.Process(chunk)
		}
		return enc.Write(chunk)
	}

	gotRate, gotCenter, capErr := s.capture.CaptureStream(r.Context(), req.Serial, req.Seconds, sink)
	flushErr := bw.Flush()
	if cerr := f.Close(); cerr != nil && flushErr == nil {
		flushErr = cerr
	}
	if capErr != nil {
		_ = os.Remove(path)
		s.writeError(w, http.StatusBadGateway, "siglab: capture: "+capErr.Error())
		return
	}
	if flushErr != nil {
		_ = os.Remove(path)
		s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+flushErr.Error())
		return
	}
	if enc.Samples() == 0 {
		_ = os.Remove(path)
		s.writeError(w, http.StatusBadGateway, "siglab: capture produced no samples")
		return
	}

	// A full-band grab keeps the tuner's authoritative rate/centre from the
	// stream; a narrowband slice keeps its decimated rate + requested centre.
	if req.BandwidthHz == 0 {
		outRate, outCenter = gotRate, gotCenter
	}

	meta := &siglab.Metadata{
		Protocol:     req.Protocol,
		Source:       req.Source,
		SampleRateHz: float64(outRate),
		CenterFreqHz: outCenter,
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
		Name:         captureName(req.Serial, outCenter),
		Path:         path,
		Format:       format,
		SampleRateHz: float64(outRate),
		Size:         enc.Samples() * int64(bytesPerSample(format)),
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

// narrowbandParams validates that the channel at wantCenterHz (±wantBWHz/2) fits
// wholly inside the tuner's current span [tunerCentre ± rate/2] — a capture is
// carved from the live stream without retuning — and returns the channel's
// offset from the tuner centre and its resolved centre (== wantCenterHz, or the
// tuner centre when wantCenterHz is 0). The offset drives the streaming DDC.
func narrowbandParams(rateHz, tunerCenterHz, wantCenterHz, wantBWHz uint32) (offsetHz int64, center uint32, err error) {
	center = wantCenterHz
	if center == 0 {
		center = tunerCenterHz
	}
	offsetHz = int64(center) - int64(tunerCenterHz)
	half := int64(rateHz) / 2
	if abs64(offsetHz)+int64(wantBWHz)/2 > half {
		return 0, 0, fmt.Errorf(
			"center_hz %d + bandwidth_hz %d falls outside the tuner's current span %.3f–%.3f MHz "+
				"(centre %.3f MHz, rate %.3f MS/s); a capture extracts a channel from the live stream "+
				"without retuning, so pick a centre/bandwidth inside the span",
			center, wantBWHz,
			float64(int64(tunerCenterHz)-half)/1e6, float64(int64(tunerCenterHz)+half)/1e6,
			float64(tunerCenterHz)/1e6, float64(rateHz)/1e6)
	}
	return offsetHz, center, nil
}

// abs64 is the absolute value of a signed 64-bit integer.
func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// captureDeviceRateCenter returns the current sample rate and centre frequency
// of the device with the given serial from the capture picker's device list, or
// (0, 0) when the device is unknown or has no rate yet (not streaming).
func captureDeviceRateCenter(devices []SpectrumDevice, serial string) (rate, center uint32) {
	for _, d := range devices {
		if d.Serial == serial {
			return d.SampleRateHz, d.CenterHz
		}
	}
	return 0, 0
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
