package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeMutator records EndCall calls and returns whatever the test
// pre-loads into pending.
type fakeMutator struct {
	mu      sync.Mutex
	pending map[string]bool // serial -> EndCall returns true
	called  []endCallRecord
}

type endCallRecord struct {
	Serial string
	Reason trunking.EndReason
}

func (f *fakeMutator) EndCall(serial string, reason trunking.EndReason) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = append(f.called, endCallRecord{Serial: serial, Reason: reason})
	return f.pending[serial]
}

type fakeRetention struct {
	mu     sync.Mutex
	swept  int
}

func (f *fakeRetention) SweepOnce(_ context.Context) {
	f.mu.Lock()
	f.swept++
	f.mu.Unlock()
}

type fakeTones struct {
	mu     sync.Mutex
	resets []string
}

func (f *fakeTones) ResetDevice(serial string) {
	f.mu.Lock()
	f.resets = append(f.resets, serial)
	f.mu.Unlock()
}

func TestMutations_Disabled_ReturnForbidden(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: false, // explicit
		Mutator:        &fakeMutator{pending: map[string]bool{"abc": true}},
		Retention:      &fakeRetention{},
		Tones:          &fakeTones{},
	})
	defer teardown()

	for _, c := range []struct {
		method, path string
	}{
		{"POST", "/api/v1/calls/abc/end"},
		{"PATCH", "/api/v1/talkgroups/42"},
		{"POST", "/api/v1/retention/sweep"},
		{"POST", "/api/v1/devices/abc/tone-reset"},
	} {
		req, _ := http.NewRequest(c.method, base+c.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", c.method, c.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: status=%d, want 403", c.method, c.path, resp.StatusCode)
		}
	}
}

func TestMutationStatus_AlwaysExposed(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Mutator:        &fakeMutator{},
		Retention:      &fakeRetention{},
	})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/mutations")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body["allow_mutations"] {
		t.Errorf("allow_mutations = false")
	}
	if !body["engine_writable"] {
		t.Errorf("engine_writable = false")
	}
	if !body["retention_writable"] {
		t.Errorf("retention_writable = false")
	}
	if body["tones_writable"] {
		t.Errorf("tones_writable = true (none wired)")
	}
}

func TestEndCall_OK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	mut := &fakeMutator{pending: map[string]bool{"abc": true}}
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, AllowMutations: true, Mutator: mut,
	})
	defer teardown()

	body := strings.NewReader(`{"reason":"manual"}`)
	resp, err := http.Post(base+"/api/v1/calls/abc/end", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if len(mut.called) != 1 || mut.called[0].Serial != "abc" || mut.called[0].Reason != trunking.EndReasonManual {
		t.Errorf("EndCall not invoked correctly: %+v", mut.called)
	}
}

func TestEndCall_NotFound(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, AllowMutations: true, Mutator: &fakeMutator{pending: map[string]bool{}},
	})
	defer teardown()

	resp, err := http.Post(base+"/api/v1/calls/missing/end", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status=%d, want 404", resp.StatusCode)
	}
}

func TestUpdateTalkgroup_AppliesPriorityAndLockout(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tgs := trunking.NewTalkgroupDB()
	tgs.Add(&trunking.TalkGroup{ID: 42, AlphaTag: "Dispatch", Priority: 5, Lockout: false})
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		AllowMutations: true,
		Talkgroups:     tgs,
	})
	defer teardown()

	body := bytes.NewReader([]byte(`{"priority":2,"lockout":true}`))
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/talkgroups/42", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	tg := tgs.Lookup(42)
	if tg.Priority != 2 || !tg.Lockout {
		t.Errorf("update not applied: %+v", tg)
	}
}

func TestUpdateTalkgroup_NotFound(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, AllowMutations: true, Talkgroups: trunking.NewTalkgroupDB(),
	})
	defer teardown()

	body := bytes.NewReader([]byte(`{"priority":1}`))
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/talkgroups/999", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status=%d, want 404", resp.StatusCode)
	}
}

func TestUpdateTalkgroup_RejectsEmptyBody(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tgs := trunking.NewTalkgroupDB()
	tgs.Add(&trunking.TalkGroup{ID: 42})
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, AllowMutations: true, Talkgroups: tgs,
	})
	defer teardown()

	body := bytes.NewReader([]byte(`{}`))
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/talkgroups/42", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
}

func TestRetentionSweep_Invokes(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	ret := &fakeRetention{}
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, AllowMutations: true, Retention: ret,
	})
	defer teardown()

	resp, err := http.Post(base+"/api/v1/retention/sweep", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if ret.swept != 1 {
		t.Errorf("SweepOnce called %d times, want 1", ret.swept)
	}
}

func TestRetentionSweep_Unwired(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, AllowMutations: true,
	})
	defer teardown()

	resp, err := http.Post(base+"/api/v1/retention/sweep", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Errorf("status=%d, want 503", resp.StatusCode)
	}
}

func TestToneReset_Invokes(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tn := &fakeTones{}
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, AllowMutations: true, Tones: tn,
	})
	defer teardown()

	resp, err := http.Post(base+"/api/v1/devices/00000001/tone-reset", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if len(tn.resets) != 1 || tn.resets[0] != "00000001" {
		t.Errorf("ResetDevice not called correctly: %+v", tn.resets)
	}
}

func TestTalkgroupDB_UpdateFields(t *testing.T) {
	db := trunking.NewTalkgroupDB()
	db.Add(&trunking.TalkGroup{ID: 1, Priority: 5})

	ok := db.UpdateFields(1, func(tg *trunking.TalkGroup) {
		tg.Priority = 3
		tg.Lockout = true
	})
	if !ok {
		t.Fatalf("UpdateFields returned false for existing id")
	}
	got := db.Lookup(1)
	if got.Priority != 3 || !got.Lockout {
		t.Errorf("UpdateFields did not apply: %+v", got)
	}

	if db.UpdateFields(999, func(*trunking.TalkGroup) {}) {
		t.Errorf("UpdateFields(missing) returned true")
	}
	if !db.Delete(1) {
		t.Errorf("Delete(existing) returned false")
	}
	if db.Delete(999) {
		t.Errorf("Delete(missing) returned true")
	}
}
