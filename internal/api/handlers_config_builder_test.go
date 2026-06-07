package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// cbServer spins up a Server with the Config Builder enabled, rooted at a
// fresh temp dir, with mutations allowed so save is reachable.
func cbServer(t *testing.T) (string, string, func()) {
	t.Helper()
	dir := t.TempDir()
	bus := events.NewBus(8)
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		ConfigBuilder:  ConfigBuilderOptions{Enabled: true, ConfigDir: dir},
	})
	return base, dir, func() {
		teardown()
		bus.Close()
	}
}

func TestConfigBuilder_ListLoad(t *testing.T) {
	base, dir, teardown := cbServer(t)
	defer teardown()

	// Seed a valid config file.
	cfg := config.Default()
	cfg.Trunking.Systems = []config.SystemConfig{{
		Name: "Metro", Protocol: "p25", ControlChannels: []uint32{851_037_500},
	}}
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// List.
	var list ConfigListResponse
	getJSON(t, base+"/api/v1/config/files", &list)
	if len(list.Files) != 1 || list.Files[0].Name != "config.yaml" || !list.Files[0].Valid {
		t.Fatalf("unexpected list: %+v", list)
	}

	// Load.
	var loaded ConfigLoadResponse
	getJSON(t, base+"/api/v1/config/file?path="+path, &loaded)
	if !loaded.Validation.OK {
		t.Fatalf("expected valid config, got %+v", loaded.Validation)
	}
	if len(loaded.Config.Trunking.Systems) != 1 || loaded.Config.Trunking.Systems[0].Name != "Metro" {
		t.Fatalf("config did not load: %+v", loaded.Config.Trunking.Systems)
	}
	if loaded.Mtime == 0 {
		t.Fatalf("expected mtime")
	}
}

func TestConfigBuilder_ValidateAndSave(t *testing.T) {
	base, dir, teardown := cbServer(t)
	defer teardown()

	// Validate an invalid config → expect a sdr section error.
	bad := config.Default()
	bad.SDR.SampleRate = 100
	var vr ValidationResult
	postJSON(t, base+"/api/v1/config/validate", ConfigValidateRequest{Config: bad}, &vr)
	if vr.OK || len(vr.Errors) == 0 || vr.Errors[0].Section != "sdr" {
		t.Fatalf("expected sdr validation error, got %+v", vr)
	}

	// Save a valid config.
	good := config.Default()
	good.Trunking.Systems = []config.SystemConfig{{
		Name: "Metro", Protocol: "p25", ControlChannels: []uint32{851_037_500},
		TalkgroupFile: "metro-tgs.csv",
	}}
	path := filepath.Join(dir, "new.yaml")
	var sr ConfigSaveResponse
	code := postJSONStatus(t, base+"/api/v1/config/file", ConfigSaveRequest{
		Path:   path,
		Config: good,
		Talkgroups: map[string][]TalkgroupCSVRow{
			"metro-tgs.csv": {{Decimal: 101, AlphaTag: "PD Disp", Tag: "Law Dispatch"}},
		},
	}, &sr)
	if code != http.StatusOK {
		t.Fatalf("save status=%d", code)
	}
	if _, err := config.Load(path); err != nil {
		t.Fatalf("saved config does not load: %v", err)
	}
	// Talkgroup sidecar was written.
	if len(sr.TalkgroupCSVs) != 1 {
		t.Fatalf("expected one talkgroup csv, got %+v", sr.TalkgroupCSVs)
	}
	if _, err := os.Stat(filepath.Join(dir, "metro-tgs.csv")); err != nil {
		t.Fatalf("talkgroup csv not written: %v", err)
	}

	// Saving again without overwrite → 409.
	code = postJSONStatus(t, base+"/api/v1/config/file", ConfigSaveRequest{Path: path, Config: good}, nil)
	if code != http.StatusConflict {
		t.Fatalf("expected 409 on existing file without overwrite, got %d", code)
	}
}

func TestConfigBuilder_PathTraversalRejected(t *testing.T) {
	base, _, teardown := cbServer(t)
	defer teardown()

	code := postJSONStatus(t, base+"/api/v1/config/file", ConfigSaveRequest{
		Path:   "/etc/passwd.yaml",
		Config: config.Default(),
	}, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-root path, got %d", code)
	}
}

func TestConfigBuilder_RRWithoutCreds(t *testing.T) {
	base, _, teardown := cbServer(t)
	defer teardown()

	resp, err := http.Get(base + "/api/v1/config/rr/search?zip=78701")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without RR creds, got %d", resp.StatusCode)
	}
}

func TestConfigBuilder_SecondSPAMount(t *testing.T) {
	dir := t.TempDir()
	bus := events.NewBus(8)
	defer bus.Close()
	// Daemon-style: main SPA at /, builder SPA at /config/.
	base, teardown := mkServer(t, ServerOptions{
		Bus:       bus,
		WebAssets: fakeSPAFS(),
		ConfigBuilder: ConfigBuilderOptions{
			Enabled:   true,
			ConfigDir: dir,
			Assets: fstest.MapFS{
				"index.html": &fstest.MapFile{Data: []byte("<html>config-builder-root</html>")},
			},
		},
	})
	defer teardown()

	// /config/ serves the builder index.
	resp, err := http.Get(base + "/config/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !bytes.Contains(body, []byte("config-builder-root")) {
		t.Fatalf("/config/ status=%d body=%q", resp.StatusCode, body)
	}

	// A deep builder route falls back to the builder index.
	resp2, err := http.Get(base + "/config/anything")
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !bytes.Contains(body2, []byte("config-builder-root")) {
		t.Fatalf("/config/anything did not fall back to builder index: %q", body2)
	}

	// The main SPA still owns /.
	resp3, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if !bytes.Contains(body3, []byte("spa-root")) {
		t.Fatalf("/ did not serve the main SPA: %q", body3)
	}
}

// --- helpers ---

func getJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s: status=%d body=%s", url, resp.StatusCode, b)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}

func postJSON(t *testing.T, url string, body, out any) {
	t.Helper()
	if code := postJSONStatus(t, url, body, out); code != http.StatusOK {
		t.Fatalf("POST %s: status=%d", url, code)
	}
}

func postJSONStatus(t *testing.T, url string, body, out any) int {
	t.Helper()
	buf, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode
}
