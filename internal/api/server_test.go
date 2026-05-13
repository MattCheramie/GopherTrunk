package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/gorilla/websocket"
)

type fakeEngine struct {
	calls []*trunking.ActiveCall
}

func (f *fakeEngine) ActiveCalls() []*trunking.ActiveCall { return f.calls }

// mkServer wires a Server on a random localhost port and returns the
// base URL plus a teardown function.
func mkServer(t *testing.T, opts ServerOptions) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	opts.Addr = addr

	s, err := NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = s.Run(ctx) }()
	// Wait for listener to come up.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return "http://" + addr, func() {
				cancel()
				s.Close()
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server didn't start on %s", addr)
	return "", nil
}

func TestHealth(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Version: "test"})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("health status = %v", body["status"])
	}
	if body["version"] != "test" {
		t.Errorf("health version = %v, want %q", body["version"], "test")
	}
	// Extended diagnostic fields default to zero/false when their
	// collaborators aren't wired (this test doesn't pass Engine,
	// Devices, History, MetricsHandler).
	if got := body["pool_attached_count"]; got != 0.0 {
		t.Errorf("pool_attached_count = %v, want 0", got)
	}
	if got := body["active_calls"]; got != 0.0 {
		t.Errorf("active_calls = %v, want 0", got)
	}
	if got := body["db_connected"]; got != false {
		t.Errorf("db_connected = %v, want false", got)
	}
	if got := body["metrics_enabled"]; got != false {
		t.Errorf("metrics_enabled = %v, want false", got)
	}
	if _, ok := body["now"]; !ok {
		t.Error("health body missing 'now' timestamp")
	}
}

func TestVersion(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Version: "v1.2.3"})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["version"] != "v1.2.3" {
		t.Errorf("version = %s, want v1.2.3", body["version"])
	}
}

func mustGet(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func TestListAndGetSystems(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	systems := []trunking.System{
		{Name: "Alpha", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{851_000_000}},
		{Name: "Bravo", Protocol: trunking.ProtocolDMR, ControlChannels: []uint32{460_000_000}},
	}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/systems")
	defer resp.Body.Close()
	var listBody struct {
		Systems []SystemDTO `json:"systems"`
	}
	json.NewDecoder(resp.Body).Decode(&listBody)
	if len(listBody.Systems) != 2 || listBody.Systems[0].Name != "Alpha" {
		t.Errorf("systems = %+v", listBody.Systems)
	}

	resp2 := mustGet(t, base+"/api/v1/systems/Bravo")
	defer resp2.Body.Close()
	var s SystemDTO
	json.NewDecoder(resp2.Body).Decode(&s)
	if s.Name != "Bravo" || s.Protocol != "dmr" {
		t.Errorf("system = %+v", s)
	}

	resp3 := mustGet(t, base+"/api/v1/systems/missing")
	defer resp3.Body.Close()
	if resp3.StatusCode != 404 {
		t.Errorf("missing system status = %d, want 404", resp3.StatusCode)
	}
}

func TestListAndGetTalkgroups(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	db := trunking.NewTalkgroupDB()
	db.Add(&trunking.TalkGroup{ID: 100, AlphaTag: "OPS-1", Priority: 1})
	db.Add(&trunking.TalkGroup{ID: 200, AlphaTag: "OPS-2", Priority: 5, Lockout: true})
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Talkgroups: db})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/talkgroups")
	defer resp.Body.Close()
	var listBody struct {
		Talkgroups []TalkgroupDTO `json:"talkgroups"`
	}
	json.NewDecoder(resp.Body).Decode(&listBody)
	if len(listBody.Talkgroups) != 2 {
		t.Errorf("talkgroups = %d, want 2", len(listBody.Talkgroups))
	}

	resp2 := mustGet(t, base+"/api/v1/talkgroups/100")
	defer resp2.Body.Close()
	var tg TalkgroupDTO
	json.NewDecoder(resp2.Body).Decode(&tg)
	if tg.AlphaTag != "OPS-1" {
		t.Errorf("OPS-1 = %+v", tg)
	}

	resp3 := mustGet(t, base+"/api/v1/talkgroups/9999")
	defer resp3.Body.Close()
	if resp3.StatusCode != 404 {
		t.Errorf("missing tg status = %d, want 404", resp3.StatusCode)
	}

	resp4 := mustGet(t, base+"/api/v1/talkgroups/notanumber")
	defer resp4.Body.Close()
	if resp4.StatusCode != 400 {
		t.Errorf("bad id status = %d, want 400", resp4.StatusCode)
	}
}

func TestActiveCallsReportsEngineSnapshot(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dev := &trunking.VoiceDevice{Serial: "VOICE-1"}
	engine := &fakeEngine{
		calls: []*trunking.ActiveCall{{
			Device:    dev,
			Grant:     trunking.Grant{System: "Alpha", Protocol: "p25", GroupID: 1234, FrequencyHz: 851_000_000},
			Talkgroup: &trunking.TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP"},
			StartedAt: time.Now().UTC(),
		}},
	}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Engine: engine})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/calls/active")
	defer resp.Body.Close()
	var body struct {
		Calls []ActiveCallDTO `json:"calls"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Calls) != 1 || body.Calls[0].Talkgroup.AlphaTag != "FIRE-DISP" {
		t.Errorf("calls = %+v", body.Calls)
	}
}

func TestSSEEmitsBusEvents(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	req, _ := http.NewRequest("GET", base+"/api/v1/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %s", resp.Header.Get("Content-Type"))
	}

	// Publish an event after the SSE stream has opened.
	go func() {
		time.Sleep(50 * time.Millisecond)
		bus.Publish(events.Event{
			Kind: events.KindCallStart,
			Payload: trunking.CallStart{
				Grant:        trunking.Grant{System: "Alpha", GroupID: 1234, FrequencyHz: 851_000_000},
				DeviceSerial: "VOICE-1",
				StartedAt:    time.Now().UTC(),
			},
		})
	}()

	// Read until we see a "data:" line containing our payload.
	br := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	got := false
	for time.Now().Before(deadline) {
		_ = resp.Request.Context()
		line, err := readSSELine(br)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"VOICE-1"`) {
			got = true
			break
		}
	}
	if !got {
		t.Error("did not receive call.start event in SSE stream")
	}
}

// readSSELine reads one line from an SSE stream with a soft deadline.
func readSSELine(br *bufio.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		l, err := br.ReadString('\n')
		ch <- result{l, err}
	}()
	select {
	case r := <-ch:
		return r.line, r.err
	case <-time.After(500 * time.Millisecond):
		return "", io.EOF
	}
}

func TestWebSocketStreamsBusEvents(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	wsURL := strings.Replace(base, "http://", "ws://", 1) + "/api/v1/events/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Publish after connection.
	go func() {
		time.Sleep(50 * time.Millisecond)
		bus.Publish(events.Event{
			Kind:    events.KindCCLocked,
			Payload: map[string]any{"freq_hz": 851_000_000},
		})
	}()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got EventDTO
	if err := conn.ReadJSON(&got); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got.Kind != string(events.KindCCLocked) {
		t.Errorf("kind = %s, want %s", got.Kind, events.KindCCLocked)
	}
}

func TestNewServerValidates(t *testing.T) {
	if _, err := NewServer(ServerOptions{}); err == nil {
		t.Error("expected error for missing Addr")
	}
	if _, err := NewServer(ServerOptions{Addr: ":0"}); err == nil {
		t.Error("expected error for missing Bus")
	}
}

// Round-tripping httptest.Server against handlers without binding a real
// port lets us cover responses cheaply.
func TestHandlersDirect(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	s := &Server{
		bus:        bus,
		systems:    []trunking.System{{Name: "X", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{1_000_000}}},
		talkgroups: trunking.NewTalkgroupDB(),
		log:        nopLogger(),
		version:    "test",
	}
	mux := s.routes()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/v1/health")
	if resp.StatusCode != 200 {
		t.Errorf("health status = %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func nopLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
