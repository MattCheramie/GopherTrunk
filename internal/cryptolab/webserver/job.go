package webserver

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
)

// event is one captured progress line from a running mode.
type event struct {
	Seq   int    `json:"seq"`
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Attrs string `json:"attrs,omitempty"`
}

// job is one async tool run: its captured log events, terminal state, and the
// structured Result.
type job struct {
	ID, Tool, Mode, dir string
	created             time.Time

	mu     sync.Mutex
	cond   *sync.Cond
	state  string // running | done | error
	errMsg string
	events []event
	result *cryptolab.Result
}

func newJob(id, tool, mode, dir string) *job {
	j := &job{ID: id, Tool: tool, Mode: mode, dir: dir, created: time.Now(), state: "running"}
	j.cond = sync.NewCond(&j.mu)
	return j
}

// handler returns a slog.Handler that captures the mode's progress lines as
// streamable events.
func (j *job) handler() slog.Handler { return &jobHandler{job: j} }

func (j *job) append(ev event) {
	j.mu.Lock()
	ev.Seq = len(j.events)
	if len(j.events) < 100_000 {
		j.events = append(j.events, ev)
	}
	j.cond.Broadcast()
	j.mu.Unlock()
}

func (j *job) finish(res *cryptolab.Result, err error) {
	j.mu.Lock()
	if err != nil {
		j.state, j.errMsg = "error", err.Error()
	} else {
		j.state, j.result = "done", res
	}
	j.cond.Broadcast()
	j.mu.Unlock()
}

func (j *job) wake() {
	j.mu.Lock()
	j.cond.Broadcast()
	j.mu.Unlock()
}

// eventsFrom blocks until new events arrive or the job ends, then returns the
// events past *cursor and the current state.
func (j *job) eventsFrom(cursor *int, ctx context.Context) ([]event, string, string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for *cursor >= len(j.events) && j.state == "running" && ctx.Err() == nil {
		j.cond.Wait()
	}
	out := append([]event(nil), j.events[*cursor:]...)
	*cursor = len(j.events)
	return out, j.state, j.errMsg
}

// jobDTO is the JSON shape of GET /jobs/{id}.
type jobDTO struct {
	ID        string            `json:"id"`
	Tool      string            `json:"tool"`
	Mode      string            `json:"mode"`
	State     string            `json:"state"`
	Error     string            `json:"error,omitempty"`
	Events    int               `json:"events"`
	Result    *cryptolab.Result `json:"result,omitempty"`
	Artifacts []artifactDTO     `json:"artifacts,omitempty"`
}

type artifactDTO struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (j *job) snapshot() jobDTO {
	j.mu.Lock()
	dto := jobDTO{
		ID:     j.ID,
		Tool:   j.Tool,
		Mode:   j.Mode,
		State:  j.state,
		Error:  j.errMsg,
		Events: len(j.events),
		Result: j.result,
	}
	done := j.state != "running"
	j.mu.Unlock()
	if done {
		dto.Artifacts = listArtifacts(j.dir)
	}
	return dto
}

// listArtifacts returns the regular files a finished job wrote to its dir
// (survivor logs, checkpoints, descrambled output, …) for download.
func listArtifacts(dir string) []artifactDTO {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []artifactDTO
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, artifactDTO{Name: e.Name(), Size: info.Size()})
	}
	return out
}

// jobHandler is a minimal slog.Handler that turns log records into job events.
type jobHandler struct {
	job   *job
	attrs string
}

func (h *jobHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *jobHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := h.attrs
	r.Attrs(func(a slog.Attr) bool {
		if attrs != "" {
			attrs += " "
		}
		attrs += a.Key + "=" + a.Value.String()
		return true
	})
	h.job.append(event{
		Time:  r.Time.Format("15:04:05"),
		Level: r.Level.String(),
		Msg:   r.Message,
		Attrs: attrs,
	})
	return nil
}

func (h *jobHandler) WithAttrs(as []slog.Attr) slog.Handler {
	extra := h.attrs
	for _, a := range as {
		if extra != "" {
			extra += " "
		}
		extra += a.Key + "=" + a.Value.String()
	}
	return &jobHandler{job: h.job, attrs: extra}
}

func (h *jobHandler) WithGroup(_ string) slog.Handler { return h }
