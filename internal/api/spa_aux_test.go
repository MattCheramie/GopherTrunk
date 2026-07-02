package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// auxSPAFS is a minimal SPA tree for the /siglab/ and /rfscope/ mount tests.
func auxSPAFS(marker string) fstest.MapFS {
	return fstest.MapFS{
		"index.html":   &fstest.MapFile{Data: []byte("<html>" + marker + "</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('" + marker + "');")},
	}
}

func TestSPA_SiglabMountedOnDaemon(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	// Daemon-style: main SPA at /, Signal Lab SPA at /siglab/.
	base, teardown := mkServer(t, ServerOptions{
		Bus:       bus,
		WebAssets: fakeSPAFS(),
		Runtime:   fakeRuntime{},
		Siglab:    SiglabOptions{Enabled: true, Assets: auxSPAFS("siglab-root")},
	})
	defer teardown()

	// /siglab/ serves the Signal Lab index.
	resp, err := http.Get(base + "/siglab/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !bytesContains(body, "siglab-root") {
		t.Fatalf("/siglab/ status=%d body=%q", resp.StatusCode, body)
	}

	// A deep client route falls back to the Signal Lab index.
	resp2, err := http.Get(base + "/siglab/captures")
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !bytesContains(body2, "siglab-root") {
		t.Fatalf("/siglab/captures did not fall back to Signal Lab index: %q", body2)
	}

	// A real asset is served directly.
	resp3, err := http.Get(base + "/siglab/assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if resp3.StatusCode != 200 || !bytesContains(body3, "console.log") {
		t.Fatalf("/siglab/assets/app.js status=%d body=%q", resp3.StatusCode, body3)
	}

	// The runtime endpoint advertises the console so the other UIs link to it.
	assertConsoleFlag(t, base, "siglab_console", true)

	// The main SPA still owns /.
	resp4, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	if !bytesContains(body4, "spa-root") {
		t.Fatalf("main SPA no longer owns /: %q", body4)
	}
}

func TestSPA_RFScopeMountedOnDaemon(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus:       bus,
		WebAssets: fakeSPAFS(),
		Runtime:   fakeRuntime{},
		RFScope:   RFScopeOptions{Enabled: true, Assets: auxSPAFS("rfscope-root")},
	})
	defer teardown()

	// /rfscope/ serves the RF Scope index.
	resp, err := http.Get(base + "/rfscope/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !bytesContains(body, "rfscope-root") {
		t.Fatalf("/rfscope/ status=%d body=%q", resp.StatusCode, body)
	}

	// The RF Scope analysis API is mounted alongside the SPA.
	resp2, err := http.Get(base + "/api/v1/rfscope/analyzers")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("/api/v1/rfscope/analyzers status=%d want 200", resp2.StatusCode)
	}

	assertConsoleFlag(t, base, "rfscope_console", true)
}

// TestSPA_AuxConsolesAbsentWhenNotMounted confirms the runtime flags stay off
// (and the routes 404) when the aux consoles aren't wired — e.g. a daemon whose
// SPAs weren't bundled.
func TestSPA_AuxConsolesAbsentWhenNotMounted(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus:       bus,
		WebAssets: fakeSPAFS(),
		Runtime:   fakeRuntime{},
		// Siglab API enabled but no SPA assets → no /siglab/ mount.
		Siglab: SiglabOptions{Enabled: true},
	})
	defer teardown()

	assertConsoleFlag(t, base, "siglab_console", false)
	assertConsoleFlag(t, base, "rfscope_console", false)

	// /siglab/ falls through to the main SPA (client route), not a siglab mount.
	resp, err := http.Get(base + "/siglab/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if bytesContains(body, "siglab-root") {
		t.Fatalf("/siglab/ served a Signal Lab mount that should not exist: %q", body)
	}
}

// assertConsoleFlag fetches GET /api/v1/runtime and asserts the boolean field
// `key` equals `want` (absent counts as false, since the fields are omitempty).
func assertConsoleFlag(t *testing.T, base, key string, want bool) {
	t.Helper()
	resp, err := http.Get(base + "/api/v1/runtime")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("/api/v1/runtime status=%d want 200", resp.StatusCode)
	}
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode runtime: %v", err)
	}
	got, _ := m[key].(bool)
	if got != want {
		t.Errorf("runtime.%s = %v, want %v (full=%v)", key, got, want, m)
	}
}
