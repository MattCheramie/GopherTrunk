package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// fakeConfigWriter is a thread-safe ConfigWriter that records every
// patch it received and returns a fixed merged Config.
type fakeConfigWriter struct {
	path    string
	patches atomic.Int32
	last    atomic.Value // config.Patch
	failErr error
}

func (f *fakeConfigWriter) WritePatch(p config.Patch) (config.Config, error) {
	if f.failErr != nil {
		return config.Config{}, f.failErr
	}
	f.patches.Add(1)
	f.last.Store(p)
	return p.Apply(config.Config{}), nil
}

func (f *fakeConfigWriter) Path() string { return f.path }

// fakeSettingsApplier counts hot-reload dispatches per knob so the
// applied/restart_required classification can be verified.
type fakeSettingsApplier struct {
	volCalls  atomic.Int32
	muteCalls atomic.Int32
	scanCalls atomic.Int32
	logCalls  atomic.Int32
	recCalls  atomic.Int32
	scanErr   error
	logErr    error
}

func (a *fakeSettingsApplier) SetLogLevel(level string) error {
	a.logCalls.Add(1)
	return a.logErr
}
func (a *fakeSettingsApplier) SetAudioVolume(v float32)   { a.volCalls.Add(1) }
func (a *fakeSettingsApplier) SetAudioMuted(m bool)       { a.muteCalls.Add(1) }
func (a *fakeSettingsApplier) SetAudioEnabled(e bool)     {}
func (a *fakeSettingsApplier) SetRecordingEnabled(e bool) { a.recCalls.Add(1) }
func (a *fakeSettingsApplier) SetScannerScanMode(m string) error {
	a.scanCalls.Add(1)
	return a.scanErr
}

func TestSettingsPatch_NoWriter(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
	})
	defer teardown()

	resp, err := http.Post(base+"/api/v1/settings", "application/json", strings.NewReader(`{"log_level":"debug"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// PATCH route — http.Post sends POST; expect 405 from the mux.
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d (expected 405 for POST on PATCH-only route)", resp.StatusCode)
	}
}

func patch(t *testing.T, base, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, base+"/api/v1/settings",
		strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestSettingsPatch_NoWriterReturns503(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
	})
	defer teardown()
	resp := patch(t, base, `{"log_level":"debug"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503", resp.StatusCode)
	}
}

func TestSettingsPatch_HotAndCold(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fw := &fakeConfigWriter{path: "/tmp/cfg.yaml"}
	fa := &fakeSettingsApplier{}
	base, teardown := mkServer(t, ServerOptions{
		Bus:             bus,
		AllowMutations:  true,
		ConfigWriter:    fw,
		SettingsApplier: fa,
	})
	defer teardown()

	resp := patch(t, base, `{"audio_volume":0.42,"recordings_dir":"/tmp/recs"}`)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, buf.String())
	}

	var body SettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !contains(body.Applied, "audio.volume") {
		t.Errorf("applied=%v missing audio.volume", body.Applied)
	}
	if !contains(body.RestartRequired, "recordings.dir") {
		t.Errorf("restart_required=%v missing recordings.dir", body.RestartRequired)
	}
	if body.ConfigPath != "/tmp/cfg.yaml" {
		t.Errorf("config_path=%q want /tmp/cfg.yaml", body.ConfigPath)
	}
	if got := fa.volCalls.Load(); got != 1 {
		t.Errorf("SetAudioVolume calls=%d want 1", got)
	}
	if got := fw.patches.Load(); got != 1 {
		t.Errorf("WritePatch calls=%d want 1", got)
	}
}

func TestSettingsPatch_EmptyBody400(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	fw := &fakeConfigWriter{path: "/tmp/cfg.yaml"}
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		ConfigWriter:   fw,
	})
	defer teardown()

	resp := patch(t, base, `{}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", resp.StatusCode)
	}
}

func TestSettingsPatch_RealWriterRoundTrip(t *testing.T) {
	// End-to-end with the real config.Writer to assert the wire
	// shape really lands on disk.
	bus := events.NewBus(8)
	defer bus.Close()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("log:\n  level: info\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := config.NewWriter(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	base, teardown := mkServer(t, ServerOptions{
		Bus:             bus,
		AllowMutations:  true,
		ConfigWriter:    w,
		SettingsApplier: &fakeSettingsApplier{},
	})
	defer teardown()

	resp := patch(t, base, `{"log_level":"warn"}`)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, buf.String())
	}
	out, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "level: warn") {
		t.Errorf("expected updated level in file, got:\n%s", out)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
