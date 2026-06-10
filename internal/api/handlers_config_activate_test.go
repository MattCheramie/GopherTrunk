package api

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// fakeActivator records the activate calls so the handler tests can assert
// the right path/mode reached the daemon without a real re-exec.
type fakeActivator struct {
	reloadPath  string
	restartPath string
	applied     []string
	restartReq  []string
	reloadErr   error
}

func (f *fakeActivator) ActivateReload(path string) ([]string, []string, error) {
	f.reloadPath = path
	return f.applied, f.restartReq, f.reloadErr
}

func (f *fakeActivator) ActivateRestart(path string) error {
	f.restartPath = path
	return nil
}

func cbActivateServer(t *testing.T, act ConfigActivator) (string, string, func()) {
	t.Helper()
	dir := t.TempDir()
	bus := events.NewBus(8)
	base, teardown := mkServer(t, ServerOptions{
		Bus:             bus,
		AllowMutations:  true,
		ConfigBuilder:   ConfigBuilderOptions{Enabled: true, ConfigDir: dir},
		ConfigActivator: act,
	})
	return base, dir, func() {
		teardown()
		bus.Close()
	}
}

func seedDefaultConfig(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := config.Marshal(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConfigActivate_Reload(t *testing.T) {
	act := &fakeActivator{applied: []string{"log.level"}, restartReq: []string{"sdr.sample_rate"}}
	base, dir, teardown := cbActivateServer(t, act)
	defer teardown()
	p := seedDefaultConfig(t, dir, "config.yaml")

	var resp ConfigActivateResponse
	postJSON(t, base+"/api/v1/config/activate", ConfigActivateRequest{Path: p, Mode: "reload"}, &resp)
	if act.reloadPath != p {
		t.Fatalf("ActivateReload got %q, want %q", act.reloadPath, p)
	}
	if resp.Mode != "reload" || len(resp.Applied) != 1 || len(resp.RestartRequired) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestConfigActivate_DefaultModeIsReload(t *testing.T) {
	act := &fakeActivator{}
	base, dir, teardown := cbActivateServer(t, act)
	defer teardown()
	p := seedDefaultConfig(t, dir, "config.yaml")

	var resp ConfigActivateResponse
	postJSON(t, base+"/api/v1/config/activate", ConfigActivateRequest{Path: p}, &resp)
	if act.reloadPath != p || act.restartPath != "" {
		t.Fatalf("empty mode should reload: reload=%q restart=%q", act.reloadPath, act.restartPath)
	}
}

func TestConfigActivate_Restart(t *testing.T) {
	act := &fakeActivator{}
	base, dir, teardown := cbActivateServer(t, act)
	defer teardown()
	p := seedDefaultConfig(t, dir, "config.yaml")

	code := postJSONStatus(t, base+"/api/v1/config/activate", ConfigActivateRequest{Path: p, Mode: "restart"}, nil)
	if code != http.StatusAccepted {
		t.Fatalf("restart status=%d, want 202", code)
	}
	if act.restartPath != p {
		t.Fatalf("ActivateRestart got %q, want %q", act.restartPath, p)
	}
}

func TestConfigActivate_NoActivatorIs503(t *testing.T) {
	base, dir, teardown := cbActivateServer(t, nil)
	defer teardown()
	p := seedDefaultConfig(t, dir, "config.yaml")

	code := postJSONStatus(t, base+"/api/v1/config/activate", ConfigActivateRequest{Path: p}, nil)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", code)
	}
}

func TestConfigActivate_RejectsPathOutsideRoots(t *testing.T) {
	act := &fakeActivator{}
	base, _, teardown := cbActivateServer(t, act)
	defer teardown()

	code := postJSONStatus(t, base+"/api/v1/config/activate",
		ConfigActivateRequest{Path: "/etc/elsewhere.yaml", Mode: "reload"}, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
	if act.reloadPath != "" {
		t.Fatalf("activator must not be called for a rejected path (got %q)", act.reloadPath)
	}
}

func TestConfigActivate_RejectsUnknownMode(t *testing.T) {
	act := &fakeActivator{}
	base, dir, teardown := cbActivateServer(t, act)
	defer teardown()
	p := seedDefaultConfig(t, dir, "config.yaml")

	code := postJSONStatus(t, base+"/api/v1/config/activate",
		ConfigActivateRequest{Path: p, Mode: "bogus"}, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}
