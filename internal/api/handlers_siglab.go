package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// siglabRunConfig is the JSON shape of the engine knobs a run/identify
// request carries. It mirrors siglab.Config's operator-facing fields; unset
// fields fall back to the engine defaults.
type siglabRunConfig struct {
	Protocol           string  `json:"protocol"`
	SampleRateHz       float64 `json:"sample_rate_hz"`
	Format             string  `json:"format"`
	FrequencyHz        uint32  `json:"frequency_hz"`
	TuneHz             float64 `json:"tune_hz"`
	AutoTune           bool    `json:"auto_tune"`
	Conjugate          bool    `json:"conjugate"`
	IQCorrect          bool    `json:"iq_correct"`
	CollectIQDiag      bool    `json:"collect_iq_diag"`
	CaptureIQ          bool    `json:"capture_iq"`
	CaptureIQMaxPoints int     `json:"capture_iq_max_points"`

	// P25 Phase 1 deep knobs.
	DemodMode            string `json:"demod_mode"`
	NIDSearchSpan        int    `json:"nid_search_span"`
	EnableDDA            bool   `json:"enable_dda"`
	EnableAdaptiveSlicer bool   `json:"enable_adaptive_slicer"`
	CollectReceiverState bool   `json:"collect_receiver_state"`
}

// siglabRunRequest is the body of POST /api/v1/siglab/run.
type siglabRunRequest struct {
	CaptureID string          `json:"capture_id"`
	Config    siglabRunConfig `json:"config"`
}

// config builds a siglab.Config from the request, defaulting the sample rate
// and format from the staged capture when the request omits them.
func (rc siglabRunConfig) config(cap *siglabCapture) (siglab.Config, error) {
	proto, err := siglab.ParseProtocolCLI(rc.Protocol)
	if err != nil {
		return siglab.Config{}, err
	}
	formatStr := rc.Format
	if formatStr == "" {
		formatStr = cap.Format.String()
	}
	format, err := siglab.ParseSampleFormat(formatStr)
	if err != nil {
		return siglab.Config{}, err
	}
	rate := rc.SampleRateHz
	if rate <= 0 {
		rate = cap.SampleRateHz
	}
	return siglab.Config{
		Protocol:             proto,
		SystemName:           "siglab",
		FrequencyHz:          rc.FrequencyHz,
		SampleRateHz:         rate,
		Format:               format,
		TuneHz:               rc.TuneHz,
		AutoTune:             rc.AutoTune,
		Conjugate:            rc.Conjugate,
		IQCorrect:            rc.IQCorrect,
		CollectIQDiag:        rc.CollectIQDiag,
		CaptureIQ:            rc.CaptureIQ,
		CaptureIQMaxPoints:   rc.CaptureIQMaxPoints,
		DemodMode:            rc.DemodMode,
		NIDSearchSpan:        rc.NIDSearchSpan,
		EnableDDA:            rc.EnableDDA,
		EnableAdaptiveSlicer: rc.EnableAdaptiveSlicer,
		CollectReceiverState: rc.CollectReceiverState,
	}, nil
}

// handleSiglabProtocols answers GET /api/v1/siglab/protocols with the set of
// decodable protocols and the subset that has a synthesis fixture.
func (s *Server) handleSiglabProtocols(w http.ResponseWriter, r *http.Request) {
	decode := make([]string, 0)
	for _, p := range ccdecoder.RegisteredProtocols() {
		decode = append(decode, p.String())
	}
	synth := make([]string, 0)
	for _, p := range siglab.Fixtures() {
		synth = append(synth, p.String())
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"protocols": decode,
		"fixtures":  synth,
		"formats":   []string{"u8", "f32"},
	})
}

// siglabCaptureDTO is the JSON projection of a staged capture.
type siglabCaptureDTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Format       string    `json:"format"`
	SampleRateHz float64   `json:"sample_rate_hz"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
}

func captureDTO(c *siglabCapture) siglabCaptureDTO {
	return siglabCaptureDTO{
		ID:           c.ID,
		Name:         c.Name,
		Format:       c.Format.String(),
		SampleRateHz: c.SampleRateHz,
		Size:         c.Size,
		CreatedAt:    c.Created,
	}
}

// handleSiglabUpload stages an uploaded raw-IQ capture under the service temp
// dir. multipart field "file"; form fields "sample_rate_hz" and "format"
// describe the encoding (a raw cfile/u8 carries neither). The body is
// streamed to disk and bounded by the configured max-upload size.
func (s *Server) handleSiglabUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.siglab.maxUploadBytes)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: a `file` part is required")
		return
	}
	defer file.Close()

	format, err := siglab.ParseSampleFormat(r.FormValue("format"))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	rate, _ := strconv.ParseFloat(r.FormValue("sample_rate_hz"), 64)

	id := randomID(16)
	path := s.siglab.newCapturePath(id)
	dst, err := os.Create(path)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+err.Error())
		return
	}
	n, err := io.Copy(dst, file)
	closeErr := dst.Close()
	if err != nil {
		_ = os.Remove(path)
		s.writeError(w, http.StatusBadRequest, "siglab: upload: "+err.Error())
		return
	}
	if closeErr != nil {
		_ = os.Remove(path)
		s.writeError(w, http.StatusInternalServerError, "siglab: stage capture: "+closeErr.Error())
		return
	}
	c := &siglabCapture{
		ID:           id,
		Name:         hdr.Filename,
		Path:         path,
		Format:       format,
		SampleRateHz: rate,
		Size:         n,
		Created:      time.Now(),
	}
	s.siglab.putCapture(c)
	writeJSON(w, http.StatusOK, captureDTO(c))
}

// handleSiglabDeleteCapture drops a staged capture and its temp file.
func (s *Server) handleSiglabDeleteCapture(w http.ResponseWriter, r *http.Request) {
	if !s.siglab.deleteCapture(r.PathValue("id")) {
		s.writeError(w, http.StatusNotFound, "siglab: capture not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSiglabRun starts an async engine run over a staged capture.
func (s *Server) handleSiglabRun(w http.ResponseWriter, r *http.Request) {
	var req siglabRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	cap, ok := s.siglab.getCapture(req.CaptureID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: capture not found")
		return
	}
	cfg, err := req.Config.config(cap)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	if cfg.SampleRateHz <= 0 {
		s.writeError(w, http.StatusBadRequest, "siglab: sample_rate_hz is required (capture has none)")
		return
	}

	id := randomID(16)
	job := newSiglabJob(id, cap.ID)
	s.siglab.putJob(job)
	path := cap.Path
	go func() {
		res, runErr := siglab.RunStream(path, cfg, job.appendEvent)
		job.finish(res, nil, runErr)
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": id})
}

// handleSiglabJobResult answers GET /api/v1/siglab/jobs/{id} with the job's
// current state and (when done) the Result.
func (s *Server) handleSiglabJobResult(w http.ResponseWriter, r *http.Request) {
	job, ok := s.siglab.getJob(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: job not found")
		return
	}
	writeJSON(w, http.StatusOK, job.snapshot())
}

// handleSiglabJobStream streams a job's decode events over SSE: it replays
// the buffered events then follows live updates until the job finishes.
func (s *Server) handleSiglabJobStream(w http.ResponseWriter, r *http.Request) {
	job, ok := s.siglab.getJob(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: job not found")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(": gophertrunk siglab events\n\n"))
	flusher.Flush()

	ctx := r.Context()
	cursor := 0
	// Watcher goroutine wakes the cond when the client disconnects so the
	// main loop doesn't block forever in cond.Wait after a hangup.
	go func() {
		<-ctx.Done()
		job.mu.Lock()
		job.cond.Broadcast()
		job.mu.Unlock()
	}()

	for {
		job.mu.Lock()
		for cursor >= len(job.events) && job.state == "running" && ctx.Err() == nil {
			job.cond.Wait()
		}
		pending := append([]siglab.EventRecord(nil), job.events[cursor:]...)
		cursor = len(job.events)
		state, errMsg := job.state, job.errMsg
		job.mu.Unlock()

		if ctx.Err() != nil {
			return
		}
		for _, ev := range pending {
			payload, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", sanitizeForHeader(ev.Kind), payload)
		}
		flusher.Flush()
		if state != "running" {
			// Emit a terminal event so the client knows to stop.
			done, _ := json.Marshal(map[string]string{"state": state, "error": errMsg})
			fmt.Fprintf(w, "event: done\ndata: %s\n\n", done)
			flusher.Flush()
			return
		}
	}
}

// handleSiglabJobIQ answers GET /api/v1/siglab/jobs/{id}/iq with the
// decimated IQ + soft samples captured for the run. ?kind=iq|soft narrows it.
func (s *Server) handleSiglabJobIQ(w http.ResponseWriter, r *http.Request) {
	job, ok := s.siglab.getJob(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: job not found")
		return
	}
	job.mu.Lock()
	taps := job.iqTaps
	job.mu.Unlock()
	if taps == nil {
		s.writeError(w, http.StatusNotFound, "siglab: no IQ captured (run with capture_iq=true)")
		return
	}
	switch r.URL.Query().Get("kind") {
	case "iq":
		writeJSON(w, http.StatusOK, map[string]any{
			"decimated_iq":      taps.DecimatedIQ,
			"decimated_rate_hz": taps.DecimatedRateHz,
			"stride":            taps.Stride,
		})
	case "soft":
		writeJSON(w, http.StatusOK, map[string]any{"soft_samples": taps.SoftSamples})
	default:
		writeJSON(w, http.StatusOK, taps)
	}
}

// handleSiglabExport streams a finished job's Result in the requested
// serialization format using the engine's own exporters.
func (s *Server) handleSiglabExport(w http.ResponseWriter, r *http.Request) {
	job, ok := s.siglab.getJob(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: job not found")
		return
	}
	job.mu.Lock()
	res := job.result
	job.mu.Unlock()
	if res == nil {
		s.writeError(w, http.StatusConflict, "siglab: job has no result yet")
		return
	}
	formatStr := r.URL.Query().Get("format")
	if formatStr == "" {
		formatStr = "json"
	}
	format, err := siglab.ParseFormat(formatStr)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	ct, ext := "application/json", "json"
	switch formatStr {
	case "yaml", "yml":
		ct, ext = "application/yaml", "yaml"
	case "csv", "csv-summary", "csv-events":
		ct, ext = "text/csv", "csv"
	case "jsonl":
		ct, ext = "application/x-ndjson", "jsonl"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=siglab-%s.%s", job.ID, ext))
	if err := siglab.WriteResult(w, res, format); err != nil {
		s.log.Warn("api: siglab export failed", "err", err)
	}
}

// handleSiglabIdentify scans a staged capture against candidate protocols and
// returns the ranked IdentifyResult. Runs synchronously (bounded by a prefix).
func (s *Server) handleSiglabIdentify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CaptureID    string  `json:"capture_id"`
		SampleRateHz float64 `json:"sample_rate_hz"`
		Format       string  `json:"format"`
		AutoTune     bool    `json:"auto_tune"`
		Conjugate    bool    `json:"conjugate"`
		IQCorrect    bool    `json:"iq_correct"`
		MaxSeconds   float64 `json:"max_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	cap, ok := s.siglab.getCapture(req.CaptureID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: capture not found")
		return
	}
	formatStr := req.Format
	if formatStr == "" {
		formatStr = cap.Format.String()
	}
	format, err := siglab.ParseSampleFormat(formatStr)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	rate := req.SampleRateHz
	if rate <= 0 {
		rate = cap.SampleRateHz
	}
	if rate <= 0 {
		s.writeError(w, http.StatusBadRequest, "siglab: sample_rate_hz is required")
		return
	}
	maxSeconds := req.MaxSeconds
	if maxSeconds <= 0 {
		maxSeconds = 3.0
	}
	out, err := siglab.Identify(cap.Path, siglab.IdentifyConfig{
		SampleRateHz: rate,
		Format:       format,
		AutoTune:     req.AutoTune,
		Conjugate:    req.Conjugate,
		IQCorrect:    req.IQCorrect,
		MaxSamples:   int64(maxSeconds * rate),
	})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// siglabSynthRequest is the body of POST /api/v1/siglab/synthesize. It mirrors
// the `gen` CLI surface.
type siglabSynthRequest struct {
	Protocol          string  `json:"protocol"`
	Format            string  `json:"format"`
	Modulation        string  `json:"modulation"`
	SNRdB             float64 `json:"snr_db"`
	FreqOffsetHz      float64 `json:"freq_offset_hz"`
	FreqDriftHzPerSec float64 `json:"freq_drift_hz_per_sec"`
	DCOffset          float64 `json:"dc_offset"`
	IQGainImbalance   float64 `json:"iq_gain_imbalance"`
	IQPhaseSkewRad    float64 `json:"iq_phase_skew_rad"`
	Seed              int64   `json:"seed"`
	Multipath         string  `json:"multipath"`
}

// handleSiglabSynthesize generates an idealized/impaired capture for a
// protocol. ?download=1 returns the raw capture bytes; otherwise the capture
// is staged (like an upload) and its DTO is returned so it can be run and
// compared against an uploaded capture.
func (s *Server) handleSiglabSynthesize(w http.ResponseWriter, r *http.Request) {
	var req siglabSynthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	proto, err := siglab.ParseProtocolCLI(req.Protocol)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	format, err := siglab.ParseSampleFormat(req.Format)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	taps, err := parseSiglabMultipath(req.Multipath)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	seed := req.Seed
	if seed == 0 {
		seed = 1
	}
	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol:   proto,
		Format:     format,
		Modulation: req.Modulation,
		Impairments: demod.Impairments{
			Multipath:         taps,
			SNRdB:             req.SNRdB,
			FreqOffsetHz:      req.FreqOffsetHz,
			FreqDriftHzPerSec: req.FreqDriftHzPerSec,
			DCOffset:          complex(float32(req.DCOffset), 0),
			IQGainImbalance:   req.IQGainImbalance,
			IQPhaseSkewRad:    req.IQPhaseSkewRad,
			Seed:              seed,
		},
	})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "siglab: "+err.Error())
		return
	}
	data := siglab.EncodeCapture(iq, format)

	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", proto.String(), format.String()))
		_, _ = w.Write(data)
		return
	}

	// Stage the synthesized capture so it can be run/compared like an upload.
	id := randomID(16)
	path := s.siglab.newCapturePath(id)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		s.writeError(w, http.StatusInternalServerError, "siglab: stage synth: "+err.Error())
		return
	}
	c := &siglabCapture{
		ID:           id,
		Name:         fmt.Sprintf("synth-%s", proto.String()),
		Path:         path,
		Format:       format,
		SampleRateHz: meta.SampleRateHz,
		Size:         int64(len(data)),
		Created:      time.Now(),
	}
	s.siglab.putCapture(c)
	writeJSON(w, http.StatusOK, map[string]any{
		"capture":  captureDTO(c),
		"metadata": meta,
	})
}

// parseSiglabMultipath mirrors cmd/gophertrunk/gen.go's parseMultipath: empty
// ⇒ none, "simulcast" ⇒ the issue #492 near-null two-path channel, otherwise
// comma-separated "delay:mag:deg" taps.
func parseSiglabMultipath(s string) ([]complex64, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "":
		return nil, nil
	case "simulcast":
		echo := complex(float32(0.95*math.Cos(70*math.Pi/180)), float32(0.95*math.Sin(70*math.Pi/180)))
		return []complex64{complex(1, 0), 0, 0, 0, 0, echo}, nil
	}
	type tap struct {
		delay int
		val   complex64
	}
	var taps []tap
	maxDelay := 0
	for _, field := range strings.Split(s, ",") {
		parts := strings.Split(strings.TrimSpace(field), ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("multipath tap %q: want delay:mag:deg", field)
		}
		delay, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || delay < 0 {
			return nil, fmt.Errorf("multipath tap %q: bad delay", field)
		}
		mag, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("multipath tap %q: bad magnitude", field)
		}
		deg, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return nil, fmt.Errorf("multipath tap %q: bad phase", field)
		}
		rad := deg * math.Pi / 180
		taps = append(taps, tap{delay: delay, val: complex(float32(mag*math.Cos(rad)), float32(mag*math.Sin(rad)))})
		if delay > maxDelay {
			maxDelay = delay
		}
	}
	fir := make([]complex64, maxDelay+1)
	for _, t := range taps {
		fir[t.delay] = t.val
	}
	return fir, nil
}
