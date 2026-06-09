package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
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

func TestConfigBuilder_TalkgroupRoundTrip(t *testing.T) {
	base, dir, teardown := cbServer(t)
	defer teardown()

	// Save a config with a system that references a talkgroup sidecar.
	cfg := config.Default()
	cfg.Trunking.Systems = []config.SystemConfig{{
		Name: "Metro", Protocol: "p25", ControlChannels: []uint32{851_037_500},
		TalkgroupFile: "metro-tgs.csv",
	}}
	path := filepath.Join(dir, "config.yaml")
	code := postJSONStatus(t, base+"/api/v1/config/file", ConfigSaveRequest{
		Path:   path,
		Config: cfg,
		Talkgroups: map[string][]TalkgroupCSVRow{
			"metro-tgs.csv": {
				{Decimal: 101, AlphaTag: "PD Disp", Tag: "Law Dispatch", Mode: "D"},
				{Decimal: 202, AlphaTag: "FD Ops"},
			},
		},
	}, nil)
	if code != http.StatusOK {
		t.Fatalf("save status=%d", code)
	}

	// Read the talkgroups back for that config.
	var resp struct {
		Talkgroups map[string][]TalkgroupCSVRow `json:"talkgroups"`
	}
	getJSON(t, base+"/api/v1/config/file/talkgroups?path="+path, &resp)
	rows := resp.Talkgroups["metro-tgs.csv"]
	if len(rows) != 2 {
		t.Fatalf("expected 2 talkgroups, got %+v", resp.Talkgroups)
	}
	if rows[0].Decimal != 101 || rows[0].AlphaTag != "PD Disp" || rows[0].Mode != "D" {
		t.Fatalf("row0 mismatch: %+v", rows[0])
	}
}

func TestConfigBuilder_FileManagement(t *testing.T) {
	base, dir, teardown := cbServer(t)
	defer teardown()

	// mkdir a subdir, save into it, rename, then delete.
	sub := filepath.Join(dir, "sites")
	if postJSONStatus(t, base+"/api/v1/config/dir", map[string]string{"path": sub}, nil) != http.StatusOK {
		t.Fatalf("mkdir failed")
	}
	if _, err := os.Stat(sub); err != nil {
		t.Fatalf("subdir not created: %v", err)
	}

	src := filepath.Join(sub, "a.yaml")
	if postJSONStatus(t, base+"/api/v1/config/file", ConfigSaveRequest{Path: src, Config: config.Default()}, nil) != http.StatusOK {
		t.Fatalf("save into subdir failed")
	}

	dst := filepath.Join(sub, "b.yaml")
	if postJSONStatus(t, base+"/api/v1/config/file/rename", map[string]string{"from": src, "to": dst}, nil) != http.StatusOK {
		t.Fatalf("rename failed")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("old name should be gone")
	}

	// Delete via DELETE.
	reqDelete(t, base+"/api/v1/config/file?path="+dst)
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("file should be deleted")
	}

	// mkdir / rename outside the root are rejected.
	if postJSONStatus(t, base+"/api/v1/config/dir", map[string]string{"path": "/etc/evil"}, nil) != http.StatusBadRequest {
		t.Fatalf("mkdir outside root should be 400")
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
	// No env key and no built-in key → the RR routes 503.
	t.Setenv("GOPHERTRUNK_RR_KEY", "")
	setDefaultAppKey(t, "")
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

// setDefaultAppKey overrides the build-time radioreference.DefaultAppKey for a
// test and restores it afterwards.
func setDefaultAppKey(t *testing.T, key string) {
	t.Helper()
	prev := radioreference.DefaultAppKey
	radioreference.DefaultAppKey = key
	t.Cleanup(func() { radioreference.DefaultAppKey = prev })
}

// fakeRRSOAP points the handlers' RR client at a local SOAP server returning
// the given body, via the newRRClient seam. It returns the captured request
// body pointer so a test can assert the edited creds reached RR.
func fakeRRSOAP(t *testing.T, response string) *string {
	t.Helper()
	got := new(string)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = string(b)
		w.Header().Set("Content-Type", "text/xml")
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)
	prev := newRRClient
	newRRClient = func(a radioreference.Auth) (*radioreference.Client, error) {
		c, err := radioreference.NewClient(a)
		if err != nil {
			return nil, err
		}
		c.SetEndpoint(srv.URL)
		return c, nil
	}
	t.Cleanup(func() { newRRClient = prev })
	return got
}

func TestConfigBuilder_RRVerify(t *testing.T) {
	got := fakeRRSOAP(t, `<?xml version="1.0"?><E><username>alice</username>`+
		`<subExpireDate>2999-01-01 00:00:00</subExpireDate></E>`)
	base, _, teardown := cbServer(t)
	defer teardown()

	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/config/rr/verify", nil)
	req.Header.Set("X-RR-Key", "KEY")
	req.Header.Set("X-RR-User", "alice")
	req.Header.Set("X-RR-Pass", "pw")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out RRVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || !out.Premium || out.Username != "alice" {
		t.Fatalf("verify response = %+v, want ok+premium for alice", out)
	}
	// The edited username (from the header) must have reached RadioReference.
	if !strings.Contains(*got, "alice") {
		t.Errorf("RR request did not carry the edited username:\n%s", *got)
	}
}

func TestConfigBuilder_RRVerifyFault(t *testing.T) {
	fakeRRSOAP(t, `<?xml version="1.0"?><SOAP-ENV:Envelope `+
		`xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"><SOAP-ENV:Body>`+
		`<SOAP-ENV:Fault><faultstring>Invalid username or password</faultstring>`+
		`</SOAP-ENV:Fault></SOAP-ENV:Body></SOAP-ENV:Envelope>`)
	base, _, teardown := cbServer(t)
	defer teardown()

	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/config/rr/verify", nil)
	req.Header.Set("X-RR-Key", "KEY")
	req.Header.Set("X-RR-User", "alice")
	req.Header.Set("X-RR-Pass", "bad")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out RRVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.OK || out.Fault == "" {
		t.Fatalf("verify response = %+v, want ok=false with a fault", out)
	}
}

func TestRRAuthFromRequest_HeaderMerge(t *testing.T) {
	t.Setenv("GOPHERTRUNK_RR_KEY", "")
	setDefaultAppKey(t, "builtin")
	svc := &configBuilderService{rrAuth: radioreference.Auth{AppKey: "startup", Username: "startup-user"}}

	// Headers override startup creds; an empty key still backstops to built-in.
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-RR-User", "edited")
	r.Header.Set("X-RR-Pass", "edited-pw")
	auth := svc.rrAuthFromRequest(r)
	if auth.Username != "edited" || auth.Password != "edited-pw" {
		t.Errorf("user/pass = %q/%q, want edited", auth.Username, auth.Password)
	}
	if auth.AppKey != "startup" { // startup key present, so it wins over built-in
		t.Errorf("AppKey = %q, want startup", auth.AppKey)
	}

	// With no startup key and no header key, the built-in default fills in.
	svc.rrAuth = radioreference.Auth{}
	if auth := svc.rrAuthFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); auth.AppKey != "builtin" {
		t.Errorf("AppKey = %q, want builtin", auth.AppKey)
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

func reqDelete(t *testing.T, url string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("DELETE %s: status=%d body=%s", url, resp.StatusCode, b)
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
