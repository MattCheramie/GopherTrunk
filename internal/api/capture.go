package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// maxCaptureSeconds bounds a single live capture so an operator can't pin a
// tuner indefinitely with one request. The capture streams encoded IQ straight
// to disk (peak RAM is one chunk, not the whole grab), so duration is bounded by
// disk — see maxCaptureIQBytes — rather than memory; this is a coarse ceiling on
// wall-clock duration. 1200 s (20 min): with FLAC a 25 kHz DMR slice runs
// ~3–4 MB per two minutes, so the long captures operators asked for (10 Sep:
// "extend capture time to 600 or 1200 seconds everywhere") are cheap on disk
// and are exactly what a multi-transmission A/B needs.
const maxCaptureSeconds = 1200

// maxCaptureIQBytes caps the on-disk size a single capture may stage. Because
// the capture now streams encoded IQ to disk one chunk at a time (see
// siglab.CaptureWriter), memory is no longer the constraint — this bounds the
// staged file so one request can't fill the disk. The estimate is the
// UNCOMPRESSED size (bytesPerSample of the format's decoder width), so a flac
// grab is budgeted as if it were cs16 — the compression is a bonus, never
// relied on. 4 GiB is 1200 s of cs16/flac at ~890 kS/s (or 1200 s of f32 at
// ~445 kS/s, ~53 s of f32 at 10 MS/s); a grab that would exceed it is rejected
// up front (before the tuner is pinned) with an actionable error suggesting a
// shorter duration or a narrowband slice.
const maxCaptureIQBytes = 4 << 30

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
//
// CentersHz (optional) records SEVERAL narrowband slices at once — one per
// centre, all BandwidthHz wide — from the same live stream, so the staged
// files are sample-synchronous by construction: every slice is carved from
// the same IQ chunks through an identical down-converter (same rate, same
// bandwidth ⇒ same group delay), so sample N of one file is the same instant
// as sample N of every other. Two IPSC repeaters, or a P25 control channel
// plus its voice channels, land as small time-aligned recordings (16 Sep
// request). Mutually exclusive with CenterHz; needs BandwidthHz.
type captureRequest struct {
	Serial      string   `json:"serial"`
	Seconds     int      `json:"seconds"`
	Format      string   `json:"format"`
	Protocol    string   `json:"protocol,omitempty"`
	Source      string   `json:"source,omitempty"`
	CenterHz    uint32   `json:"center_hz,omitempty"`
	BandwidthHz uint32   `json:"bandwidth_hz,omitempty"`
	CentersHz   []uint32 `json:"centers_hz,omitempty"`
}

// maxCaptureSlices bounds how many synchronous slices one request may carve:
// each is a full polyphase DDC on the live stream, run on the capture
// goroutine, and the broker drops chunks to a consumer that falls behind.
const maxCaptureSlices = 8

// captureResponse is returned by a successful capture: the staged capture
// (runnable/identifiable immediately), the metadata sidecar describing it, and
// a relative URL to download the raw .cfile.
//
// A multi-slice request (centers_hz) returns every slice in Captures, in the
// request's order, and mirrors the first one into the three top-level fields
// so a single-slice client keeps working unchanged.
type captureResponse struct {
	Capture     siglabCaptureDTO       `json:"capture"`
	Metadata    *siglab.Metadata       `json:"metadata"`
	DownloadURL string                 `json:"download_url"`
	Captures    []captureSliceResponse `json:"captures,omitempty"`
}

// captureSliceResponse is one staged slice of a multi-slice capture.
type captureSliceResponse struct {
	Capture     siglabCaptureDTO `json:"capture"`
	Metadata    *siglab.Metadata `json:"metadata"`
	DownloadURL string           `json:"download_url"`
}

// captureSlice is the per-output state of one capture: the down-converter
// that carves it (nil for a full-band grab) and the encoder/container writing
// its staged file.
type captureSlice struct {
	center  uint32
	ddc     *siglab.StreamDownconverter
	id      string
	path    string
	f       *os.File
	bw      *bufio.Writer
	cont    *siglab.IQContainer
	samples func() int64
	write   func([]complex64) error
}

// close finalises the slice's file (container trailer / buffered flush) and
// closes it; the first error wins.
func (sl *captureSlice) close() error {
	var err error
	if sl.cont != nil {
		err = sl.cont.Finalize()
	} else if sl.bw != nil {
		err = sl.bw.Flush()
	}
	if cerr := sl.f.Close(); cerr != nil && err == nil {
		err = cerr
	}
	return err
}

// resolveCaptureCenters turns the request's centre fields into the list of
// slice centres (empty ⇒ a full-band grab): center_hz alone is one slice,
// centers_hz is several; both together, a duplicate, or more than
// maxCaptureSlices is an error, and any list needs bandwidth_hz (a capture is
// carved from the live stream without retuning, so a centre only means
// something for a slice).
func resolveCaptureCenters(req captureRequest, tunerCenterHz uint32) ([]uint32, error) {
	if len(req.CentersHz) == 0 {
		if req.CenterHz == 0 && req.BandwidthHz == 0 {
			return nil, nil
		}
		if req.BandwidthHz == 0 {
			// Refuse rather than quietly record the tuner centre under the
			// name of the one the operator asked for (15 Sep: a
			// "442.8125 MHz" grab that neither repeater was in).
			if req.CenterHz != tunerCenterHz {
				return nil, fmt.Errorf(
					"center_hz %d without bandwidth_hz: a capture is carved from the tuner's live stream "+
						"without retuning (tuner centre %.4f MHz), so a centre only applies to a narrowband slice — "+
						"add bandwidth_hz to slice around %.4f MHz, or omit center_hz for a full-band grab",
					req.CenterHz, float64(tunerCenterHz)/1e6, float64(req.CenterHz)/1e6)
			}
			return nil, nil
		}
		return []uint32{req.CenterHz}, nil
	}
	if req.CenterHz != 0 {
		return nil, errors.New("center_hz and centers_hz are mutually exclusive — put every slice centre in centers_hz")
	}
	if req.BandwidthHz == 0 {
		return nil, errors.New("centers_hz needs bandwidth_hz: each centre is a narrowband slice carved from the tuner's live stream")
	}
	if len(req.CentersHz) > maxCaptureSlices {
		return nil, fmt.Errorf("centers_hz lists %d slices, at most %d can be recorded together", len(req.CentersHz), maxCaptureSlices)
	}
	seen := map[uint32]bool{}
	for _, c := range req.CentersHz {
		if c == 0 {
			return nil, errors.New("centers_hz entries must be non-zero frequencies in Hz")
		}
		if seen[c] {
			return nil, fmt.Errorf("centers_hz lists %.4f MHz twice", float64(c)/1e6)
		}
		seen[c] = true
	}
	return append([]uint32(nil), req.CentersHz...), nil
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

	centers, err := resolveCaptureCenters(req, centerHz)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}

	// Optional narrowband slice(s): validate that every requested channel fits
	// the tuner's current span and build one streaming down-converter per
	// slice up front, so an out-of-span request 400s before the tuner is
	// pinned. Each slice is carved from the SAME live chunks during capture
	// (no wideband buffer), through an identical DDC, so multi-slice files are
	// sample-synchronous. Done before the budget check so a slice is sized by
	// its decimated output rate, not the full band.
	slices := []*captureSlice{{center: centerHz}}
	outRate := rate
	if len(centers) > 0 {
		if rate == 0 {
			s.writeError(w, http.StatusBadGateway,
				"siglab: capture: device has no sample rate yet (start or resume a scan on this SDR)")
			return
		}
		slices = slices[:0]
		for _, want := range centers {
			offsetHz, center, err := narrowbandParams(rate, centerHz, want, req.BandwidthHz)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
				return
			}
			ddc := siglab.NewStreamDownconverter(float64(rate), float64(offsetHz), float64(req.BandwidthHz))
			slices = append(slices, &captureSlice{center: center, ddc: ddc})
			outRate = uint32(ddc.OutRateHz() + 0.5)
		}
	}
	multi := len(slices) > 1

	// Reject an over-budget grab before pinning the tuner. Streaming keeps RAM
	// bounded to one chunk, so this bounds the on-disk size (seconds ×
	// effective rate × bytes-per-sample × slices), not memory. outRate is the
	// full band for a plain grab and the decimated slice rate for a narrowband
	// request, so a legitimate slice is not rejected on its full-band
	// footprint. Skipped when the rate is unknown (device not streaming yet) —
	// then maxCaptureSeconds is the only bound.
	if rate > 0 {
		estBytes := int64(req.Seconds) * int64(outRate) * int64(bytesPerSample(format)) * int64(len(slices))
		if estBytes > maxCaptureIQBytes {
			hint := "reduce seconds or request a narrowband slice (center_hz + bandwidth_hz)"
			if len(centers) > 0 {
				hint = "reduce seconds, bandwidth, or the number of slices"
			}
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf(
				"siglab: a %ds capture at %.3f MS/s × %d file(s) would stage ~%d MiB, over the %d MiB budget — %s",
				req.Seconds, float64(outRate)/1e6, len(slices), estBytes>>20, int64(maxCaptureIQBytes)>>20, hint))
			return
		}
	}

	// The container formats (wav/flac) need the on-disk rate for their header
	// before the first sample lands, so they require a streaming device.
	isContainer := format == siglab.FormatWAV || format == siglab.FormatFLAC
	if isContainer && outRate == 0 {
		s.writeError(w, http.StatusBadGateway,
			"siglab: capture: a wav/flac capture needs the device sample rate up front (start or resume a scan on this SDR)")
		return
	}
	// FLAC's STREAMINFO rate field is 20 bits: a full-band grab of a
	// multi-MS/s tuner cannot be a flac. Say so before pinning the tuner (the
	// encoder would otherwise fail its first block and the capture abort
	// after the operator waited on it — the 15 Sep report's first attempt).
	if format == siglab.FormatFLAC && outRate > baseband.FLACMaxSampleRateHz {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"siglab: flac cannot carry a %.3f MS/s stream (FLAC's STREAMINFO ceiling is %d Hz) — "+
				"request a narrowband slice (center_hz + bandwidth_hz) under that rate, or use cs16/wav for the full band",
			float64(outRate)/1e6, baseband.FLACMaxSampleRateHz))
		return
	}

	// Stream encoded IQ straight to the staged file(s): peak memory is one
	// chunk (plus each DDC's FIR state), independent of duration. Headerless
	// formats stream through the byte encoder; wav/flac go through the
	// IQContainer, which owns the header + finalize (RIFF length patch / FLAC
	// STREAMINFO) — a staged "wav"/"flac" capture used to get a headerless
	// (or, for flac, mis-encoded) body under a container label.
	var group string
	if multi {
		group = randomID(8)
	}
	removeAll := func() {
		for _, sl := range slices {
			if sl.f != nil {
				_ = os.Remove(sl.path)
			}
		}
	}
	for _, sl := range slices {
		sl.id = randomID(16)
		sl.path = s.siglab.newCapturePath(sl.id)
		f, err := os.Create(sl.path)
		if err != nil {
			removeAll()
			s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+err.Error())
			return
		}
		sl.f = f
		if isContainer {
			cont, err := siglab.NewIQContainer(f, format, int(outRate))
			if err != nil {
				removeAll()
				s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+err.Error())
				return
			}
			sl.cont = cont
			sl.samples, sl.write = cont.Samples, cont.Write
		} else {
			sl.bw = bufio.NewWriterSize(f, 1<<20)
			enc := siglab.NewCaptureWriter(sl.bw, format)
			sl.samples, sl.write = enc.Samples, enc.Write
		}
	}
	sink := func(chunk []complex64) error {
		for _, sl := range slices {
			out := chunk
			if sl.ddc != nil {
				out = sl.ddc.Process(chunk)
			}
			if err := sl.write(out); err != nil {
				return err
			}
		}
		return nil
	}

	// Bracket the grab in debug.log so an operator can line a capture up with
	// the daemon's decode log without guessing (13 Sep request: "capture
	// started, freq= bandwidth= format … capture ended"). The start line
	// carries everything the metadata sidecar will, plus the tuner's own
	// centre/rate for a narrowband slice; the end line carries what was
	// actually recorded.
	started := time.Now()
	startAttrs := []any{
		"serial", req.Serial, "center_hz", slices[0].center, "sample_rate_hz", outRate,
		"bandwidth_hz", req.BandwidthHz, "requested_center_hz", req.CenterHz,
		"tuner_center_hz", centerHz, "tuner_rate_hz", rate,
		"format", format.String(), "seconds", req.Seconds, "protocol", req.Protocol, "path", slices[0].path,
	}
	if multi {
		startAttrs = append(startAttrs, "capture_group", group, "slices", len(slices), "centers_hz", centers)
	}
	s.log.Info("siglab: capture started", startAttrs...)
	// A capture runs for up to req.Seconds and Shutdown would wait it out, so
	// a daemon stop cancels it the same way a client hangup does.
	capCtx, capCancel := s.streamCtx(r.Context())
	defer capCancel()
	gotRate, gotCenter, capErr := s.capture.CaptureStream(capCtx, req.Serial, req.Seconds, sink)
	samples := slices[0].samples
	logEnd := func(outcome string, err error) {
		args := []any{
			"serial", req.Serial, "center_hz", slices[0].center, "sample_rate_hz", outRate,
			"format", format.String(), "samples", samples(), "elapsed", time.Since(started).Round(time.Millisecond),
			"path", slices[0].path,
		}
		if outRate > 0 {
			args = append(args, "recorded_seconds", math.Round(float64(samples())/float64(outRate)*100)/100)
		}
		if multi {
			args = append(args, "capture_group", group, "slices", len(slices))
		}
		if err != nil {
			s.log.Warn("siglab: capture "+outcome, append(args, "err", err)...)
			return
		}
		s.log.Info("siglab: capture "+outcome, args...)
	}
	var flushErr error
	for _, sl := range slices {
		if err := sl.close(); err != nil && flushErr == nil {
			flushErr = err
		}
	}
	if capErr != nil {
		logEnd("aborted", capErr)
		removeAll()
		s.writeError(w, http.StatusBadGateway, "siglab: capture: "+capErr.Error())
		return
	}
	if flushErr != nil {
		logEnd("aborted", flushErr)
		removeAll()
		s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+flushErr.Error())
		return
	}
	if samples() == 0 {
		logEnd("aborted", errors.New("capture produced no samples"))
		removeAll()
		s.writeError(w, http.StatusBadGateway, "siglab: capture produced no samples")
		return
	}
	logEnd("ended", nil)

	// A full-band grab keeps the tuner's authoritative rate/centre from the
	// stream; a narrowband slice keeps its decimated rate + requested centre.
	// A container grab keeps the pre-capture rate its header was written with.
	if len(centers) == 0 {
		if !isContainer {
			outRate = gotRate
		}
		slices[0].center = gotCenter
	}

	resp := captureResponse{}
	for i, sl := range slices {
		meta := &siglab.Metadata{
			Protocol:     req.Protocol,
			Source:       req.Source,
			SampleRateHz: float64(outRate),
			CenterFreqHz: sl.center,
			Format:       format.String(),
		}
		if multi {
			// The group ties the sample-synchronous slices together for
			// whoever lines them up later; the start stamp ties them to
			// the daemon log.
			meta.CaptureGroup = group
			meta.CaptureGroupIndex = i
			meta.CaptureGroupSize = len(slices)
			meta.CaptureStartedAt = started.UTC().Format(time.RFC3339Nano)
		}
		// Best-effort sidecar at the path siglab.DiscoverMetadata probes
		// (<stem>.metadata.json) so the staged file is a drop-in fixture.
		metaPath := strings.TrimSuffix(sl.path, filepath.Ext(sl.path)) + ".metadata.json"
		if err := siglab.WriteMetadata(metaPath, meta); err != nil {
			s.log.Warn("api: siglab capture metadata write failed", "err", err)
		}

		c := &siglabCapture{
			ID:           sl.id,
			Name:         captureName(req.Serial, sl.center),
			Path:         sl.path,
			Format:       format,
			SampleRateHz: float64(outRate),
			CenterHz:     sl.center,
			Group:        group,
			Size:         stagedCaptureSize(sl.path, sl.samples(), format),
			Created:      started,
		}
		s.siglab.putCapture(c)
		one := captureSliceResponse{
			Capture:     captureDTO(c),
			Metadata:    meta,
			DownloadURL: fmt.Sprintf("/api/v1/siglab/captures/%s/download", sl.id),
		}
		if i == 0 {
			resp.Capture, resp.Metadata, resp.DownloadURL = one.Capture, one.Metadata, one.DownloadURL
		}
		if multi {
			resp.Captures = append(resp.Captures, one)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// stagedCaptureSize reports a staged capture's size: the real on-disk size
// when it can be statted (the only truthful number for a compressed flac),
// falling back to the uncompressed samples × bytes-per-sample estimate.
func stagedCaptureSize(path string, samples int64, format siglab.SampleFormat) int64 {
	if fi, err := os.Stat(path); err == nil {
		return fi.Size()
	}
	return samples * int64(bytesPerSample(format))
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
	case siglab.FormatFLAC:
		ext = "flac"
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
			"center_hz %d + bandwidth_hz %d falls outside the tuner's current span %.4f–%.4f MHz "+
				"(centre %.4f MHz, rate %.3f MS/s); a capture extracts a channel from the live stream "+
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
// Four decimals: the standard 6.25 / 12.5 kHz channel steps land on quarter-
// kilohertz centres (442.3875, 443.2375 MHz), and three decimals silently
// renamed a 442.3875 MHz slice "442.387MHz" (10 Sep report) — every other
// frequency-bearing name in the tree (survey/hunt captures, rfscope labels)
// already uses %.4f.
func captureName(serial string, centerHz uint32) string {
	if centerHz > 0 {
		return fmt.Sprintf("capture-%s-%.4fMHz", serial, float64(centerHz)/1e6)
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
