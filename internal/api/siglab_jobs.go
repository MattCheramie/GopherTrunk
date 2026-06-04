package api

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

const (
	// defaultSiglabMaxUploadBytes bounds a single capture upload. Raw IQ
	// captures are large; 512 MiB is generous while keeping an accidental
	// upload from filling the disk.
	defaultSiglabMaxUploadBytes = 512 << 20
	// siglabJobTTL is how long a finished job (and any uploaded capture)
	// is retained before the sweeper reclaims it.
	siglabJobTTL = 30 * time.Minute
	// siglabMaxBufferedEvents caps the per-job streamed-event buffer so a
	// pathological capture can't grow it without bound. Older events past
	// the cap are dropped from the replay buffer (the final Result still
	// carries the complete event list).
	siglabMaxBufferedEvents = 100_000
)

// SiglabOptions configures the offline signal-analysis subsystem. When
// Enabled, NewServer constructs a siglabService and the /api/v1/siglab/*
// routes are mounted. Both the daemon and the standalone `siglab serve`
// command supply this.
type SiglabOptions struct {
	Enabled bool
	// TempDir is where uploaded captures are staged. Empty ⇒ a fresh
	// os.MkdirTemp directory owned by the service.
	TempDir string
	// MaxUploadBytes bounds one capture upload. 0 ⇒ defaultSiglabMaxUploadBytes.
	MaxUploadBytes int64
}

// siglabCapture is one staged capture (uploaded or synthesized-to-disk)
// available to run. The bytes live in a temp file under the service dir.
type siglabCapture struct {
	ID           string
	Name         string
	Path         string
	Format       siglab.SampleFormat
	SampleRateHz float64
	Size         int64
	Created      time.Time
}

// siglabJob is one async engine run. Events stream in via the engine's
// onEvent sink; SSE subscribers replay the buffer then follow live updates
// via the condition variable. The final Result has its (heavy) IQTaps moved
// into iqTaps so the summary endpoint stays small.
type siglabJob struct {
	ID        string
	CaptureID string
	Created   time.Time

	mu     sync.Mutex
	cond   *sync.Cond
	state  string // "running" | "done" | "error"
	errMsg string
	events []siglab.EventRecord
	result *siglab.Result
	iqTaps *siglab.IQTaps
	meta   *siglab.Metadata
}

func newSiglabJob(id, captureID string) *siglabJob {
	j := &siglabJob{ID: id, CaptureID: captureID, Created: time.Now(), state: "running"}
	j.cond = sync.NewCond(&j.mu)
	return j
}

// appendEvent buffers an event and wakes any SSE subscribers.
func (j *siglabJob) appendEvent(ev siglab.EventRecord) {
	j.mu.Lock()
	if len(j.events) < siglabMaxBufferedEvents {
		j.events = append(j.events, ev)
	}
	j.cond.Broadcast()
	j.mu.Unlock()
}

// finish records the terminal state (result+meta or error) and wakes
// subscribers so they can flush and disconnect.
func (j *siglabJob) finish(res *siglab.Result, meta *siglab.Metadata, err error) {
	j.mu.Lock()
	if err != nil {
		j.state = "error"
		j.errMsg = err.Error()
	} else {
		j.state = "done"
		j.result = res
		j.meta = meta
		if res != nil {
			j.iqTaps = res.IQTaps
			res.IQTaps = nil // keep the summary endpoint light
		}
	}
	j.cond.Broadcast()
	j.mu.Unlock()
}

// snapshot returns a copy of the job's current state for the JSON summary.
func (j *siglabJob) snapshot() siglabJobDTO {
	j.mu.Lock()
	defer j.mu.Unlock()
	dto := siglabJobDTO{
		ID:        j.ID,
		CaptureID: j.CaptureID,
		State:     j.state,
		Error:     j.errMsg,
		Events:    len(j.events),
		CreatedAt: j.Created,
	}
	if j.result != nil {
		dto.Result = j.result
	}
	if j.meta != nil {
		dto.Metadata = j.meta
	}
	if j.iqTaps != nil {
		dto.HasIQ = len(j.iqTaps.DecimatedIQ) > 0 || len(j.iqTaps.SoftSamples) > 0
	}
	return dto
}

// siglabJobDTO is the JSON shape of GET /api/v1/siglab/jobs/{id}.
type siglabJobDTO struct {
	ID        string           `json:"id"`
	CaptureID string           `json:"capture_id"`
	State     string           `json:"state"`
	Error     string           `json:"error,omitempty"`
	Events    int              `json:"events"`
	CreatedAt time.Time        `json:"created_at"`
	Result    *siglab.Result   `json:"result,omitempty"`
	Metadata  *siglab.Metadata `json:"metadata,omitempty"`
	HasIQ     bool             `json:"has_iq"`
}

// siglabService owns the temp directory, the staged captures, and the jobs.
type siglabService struct {
	tempDir        string
	ownTempDir     bool
	maxUploadBytes int64

	mu       sync.Mutex
	captures map[string]*siglabCapture
	jobs     map[string]*siglabJob
}

func newSiglabService(opts SiglabOptions) (*siglabService, error) {
	dir := opts.TempDir
	ownDir := false
	if dir == "" {
		d, err := os.MkdirTemp("", "gophertrunk-siglab-")
		if err != nil {
			return nil, fmt.Errorf("siglab: temp dir: %w", err)
		}
		dir = d
		ownDir = true
	} else {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("siglab: temp dir %s: %w", dir, err)
		}
	}
	max := opts.MaxUploadBytes
	if max <= 0 {
		max = defaultSiglabMaxUploadBytes
	}
	return &siglabService{
		tempDir:        dir,
		ownTempDir:     ownDir,
		maxUploadBytes: max,
		captures:       map[string]*siglabCapture{},
		jobs:           map[string]*siglabJob{},
	}, nil
}

// putCapture registers a staged capture file. The caller has already written
// path under the service temp dir.
func (svc *siglabService) putCapture(c *siglabCapture) {
	svc.mu.Lock()
	svc.captures[c.ID] = c
	svc.mu.Unlock()
}

func (svc *siglabService) getCapture(id string) (*siglabCapture, bool) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	c, ok := svc.captures[id]
	return c, ok
}

func (svc *siglabService) deleteCapture(id string) bool {
	svc.mu.Lock()
	c, ok := svc.captures[id]
	if ok {
		delete(svc.captures, id)
	}
	svc.mu.Unlock()
	if ok && c.Path != "" {
		_ = os.Remove(c.Path)
	}
	return ok
}

func (svc *siglabService) putJob(j *siglabJob) {
	svc.mu.Lock()
	svc.jobs[j.ID] = j
	svc.mu.Unlock()
}

func (svc *siglabService) getJob(id string) (*siglabJob, bool) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	j, ok := svc.jobs[id]
	return j, ok
}

// newCapturePath returns a fresh temp-file path under the service dir.
func (svc *siglabService) newCapturePath(id string) string {
	return filepath.Join(svc.tempDir, id+".iq")
}

// sweep reclaims jobs and captures older than the TTL, deleting their files.
func (svc *siglabService) sweep(now time.Time) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	for id, j := range svc.jobs {
		if now.Sub(j.Created) > siglabJobTTL {
			delete(svc.jobs, id)
		}
	}
	for id, c := range svc.captures {
		if now.Sub(c.Created) > siglabJobTTL {
			delete(svc.captures, id)
			if c.Path != "" {
				_ = os.Remove(c.Path)
			}
		}
	}
}

// runSweeper kicks a background sweep every TTL/4 until stopCh closes.
func (svc *siglabService) runSweeper(stopCh <-chan struct{}) {
	t := time.NewTicker(siglabJobTTL / 4)
	defer t.Stop()
	for {
		select {
		case <-stopCh:
			return
		case now := <-t.C:
			svc.sweep(now)
		}
	}
}

// close removes the temp directory if the service created it.
func (svc *siglabService) close() {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	for _, c := range svc.captures {
		if c.Path != "" {
			_ = os.Remove(c.Path)
		}
	}
	if svc.ownTempDir && svc.tempDir != "" {
		_ = os.RemoveAll(svc.tempDir)
	}
}
