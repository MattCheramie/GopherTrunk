package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

type fakeRuntime struct{ dto RuntimeDTO }

func (f fakeRuntime) Runtime() RuntimeDTO { return f.dto }

func TestHandleRuntime_503WhenNotConfigured(t *testing.T) {
	bus := events.NewBus(8)
	s, err := NewServer(ServerOptions{
		Addr: ":0",
		Bus:  bus,
	})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runtime", nil)
	s.handleRuntime(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("got %d, want 503", rr.Code)
	}
}

func TestHandleRuntime_ServesDTO(t *testing.T) {
	bus := events.NewBus(8)
	fake := fakeRuntime{dto: RuntimeDTO{
		HTTPAddr:          "127.0.0.1:8080",
		AllowMutations:    true,
		LogLevel:          "info",
		LogFormat:         "text",
		Version:           "v0.0-test",
		AudioEnabled:      true,
		AudioDevice:       "default",
		AudioSampleRate:   8000,
		AudioBackends:     []string{"default", "null"},
		RetentionInterval: 1 * time.Hour,
		VocoderMap:        map[string]string{"p25": "imbe"},
	}}
	s, err := NewServer(ServerOptions{Addr: ":0", Bus: bus, Runtime: fake})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runtime", nil)
	s.handleRuntime(rr, req)
	if rr.Code != http.StatusOK {
		body, _ := io.ReadAll(rr.Body)
		t.Fatalf("got %d, want 200: %s", rr.Code, body)
	}
	var out RuntimeDTO
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.HTTPAddr != fake.dto.HTTPAddr {
		t.Errorf("HTTPAddr = %q, want %q", out.HTTPAddr, fake.dto.HTTPAddr)
	}
	if out.VocoderMap["p25"] != "imbe" {
		t.Errorf("vocoder map round-trip lost data: %v", out.VocoderMap)
	}
	if out.RetentionInterval != fake.dto.RetentionInterval {
		t.Errorf("retention interval lost: got %v want %v", out.RetentionInterval, fake.dto.RetentionInterval)
	}
}
