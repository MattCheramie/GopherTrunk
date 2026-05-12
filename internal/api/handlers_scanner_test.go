package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// fakeCockpit captures every mutation method call so tests can
// verify the route → cockpit wiring.
type fakeCockpit struct {
	mu       sync.Mutex
	status   ScannerStatus
	modeErr  error
	holdSys  []string
	resSys   []string
	retSys   []string
	holdConv int
	resConv  int
	dwell    []int
	missingSys map[string]bool // systems for which Hold/Resume/Retune should return false
	convNotConfigured bool
	dwellRange int // valid range [0..dwellRange); -1 means any index accepted
	prevMode string
	manualReqs []ManualTuneRequest
	manualNextIdx int
	clearedManual []int
	clearOK bool
}

func (f *fakeCockpit) Status() ScannerStatus { return f.status }
func (f *fakeCockpit) SetScanMode(m string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.modeErr != nil {
		return "", f.modeErr
	}
	prev := f.prevMode
	f.prevMode = m
	f.status.ScanMode = m
	return prev, nil
}
func (f *fakeCockpit) systemKnown(sys string) bool {
	if f.missingSys == nil {
		return true
	}
	return !f.missingSys[sys]
}
func (f *fakeCockpit) HoldHunt(sys string) bool {
	if !f.systemKnown(sys) {
		return false
	}
	f.mu.Lock()
	f.holdSys = append(f.holdSys, sys)
	f.mu.Unlock()
	return true
}
func (f *fakeCockpit) ResumeHunt(sys string) bool {
	if !f.systemKnown(sys) {
		return false
	}
	f.mu.Lock()
	f.resSys = append(f.resSys, sys)
	f.mu.Unlock()
	return true
}
func (f *fakeCockpit) ForceRetuneHunt(sys string) bool {
	if !f.systemKnown(sys) {
		return false
	}
	f.mu.Lock()
	f.retSys = append(f.retSys, sys)
	f.mu.Unlock()
	return true
}
func (f *fakeCockpit) HoldConventional() bool {
	if f.convNotConfigured {
		return false
	}
	f.mu.Lock()
	f.holdConv++
	f.mu.Unlock()
	return true
}
func (f *fakeCockpit) ResumeConventional() bool {
	if f.convNotConfigured {
		return false
	}
	f.mu.Lock()
	f.resConv++
	f.mu.Unlock()
	return true
}
func (f *fakeCockpit) DwellConventional(i int) bool {
	if f.dwellRange == 0 {
		return false
	}
	if f.dwellRange > 0 && (i < 0 || i >= f.dwellRange) {
		return false
	}
	f.mu.Lock()
	f.dwell = append(f.dwell, i)
	f.mu.Unlock()
	return true
}
func (f *fakeCockpit) ManualTune(req ManualTuneRequest) (int, bool) {
	if f.convNotConfigured {
		return 0, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.manualNextIdx
	f.manualNextIdx++
	f.manualReqs = append(f.manualReqs, req)
	return idx, true
}
func (f *fakeCockpit) ClearManualTune(i int) bool {
	if !f.clearOK {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clearedManual = append(f.clearedManual, i)
	return true
}

func TestScannerStatus_AlwaysOK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{status: ScannerStatus{ScanMode: "list", TalkgroupTotalCount: 12, TalkgroupScanCount: 4}}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/scanner")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var st ScannerStatus
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.ScanMode != "list" || st.TalkgroupTotalCount != 12 || st.TalkgroupScanCount != 4 {
		t.Errorf("decoded = %+v", st)
	}
}

func TestScannerStatus_NoCockpitReturnsEmpty(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/scanner")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status=%d, want 200 even when cockpit nil", resp.StatusCode)
	}
}

func TestScannerSetMode_Applies(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{prevMode: "all"}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	body := bytes.NewReader([]byte(`{"scan_mode":"list"}`))
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/scanner", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, buf)
	}
	if cock.status.ScanMode != "list" {
		t.Errorf("mode not applied, got %q", cock.status.ScanMode)
	}
}

func TestScannerSetMode_GatedByAllowMutations(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: false})
	defer teardown()

	body := bytes.NewReader([]byte(`{"scan_mode":"list"}`))
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/scanner", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("status=%d, want 403 (gated)", resp.StatusCode)
	}
}

func TestScannerHuntHold_Routes(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	for _, op := range []string{"hold", "resume", "retune"} {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/scanner/hunt/Demo/%s", base, op), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("%s status=%d", op, resp.StatusCode)
		}
	}
	if len(cock.holdSys) != 1 || cock.holdSys[0] != "Demo" {
		t.Errorf("holdSys=%v", cock.holdSys)
	}
	if len(cock.resSys) != 1 || len(cock.retSys) != 1 {
		t.Errorf("resume/retune not recorded: %v %v", cock.resSys, cock.retSys)
	}
}

func TestScannerHuntHold_NotFound(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{missingSys: map[string]bool{"Missing": true}}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/scanner/hunt/Missing/hold", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status=%d, want 404", resp.StatusCode)
	}
}

func TestScannerConvDwell_BadIndex(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{dwellRange: 3}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	// Valid index inside range.
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/scanner/conventional/1/dwell", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
	// Out-of-range.
	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/scanner/conventional/99/dwell", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("out-of-range status=%d, want 404", resp.StatusCode)
	}
}

func TestScannerConvHold_NoConvScanner(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{convNotConfigured: true}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/scanner/conventional/hold", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Errorf("status=%d, want 503 (no conv scanner)", resp.StatusCode)
	}
}

func TestScannerManualTune_OK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	body := bytes.NewReader([]byte(`{"frequency_hz":155895000,"label":"sheriff","mode":"fm"}`))
	resp, err := http.Post(base+"/api/v1/scanner/manual_tune", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if len(cock.manualReqs) != 1 || cock.manualReqs[0].FrequencyHz != 155895000 || cock.manualReqs[0].Label != "sheriff" {
		t.Errorf("ManualTune not routed: %+v", cock.manualReqs)
	}
}

func TestScannerManualTune_RejectsMissingFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	body := bytes.NewReader([]byte(`{"label":"x"}`))
	resp, err := http.Post(base+"/api/v1/scanner/manual_tune", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
	if len(cock.manualReqs) != 0 {
		t.Errorf("ManualTune should not be called on rejected request")
	}
}

func TestScannerManualTune_RejectsOutOfRangeFreq(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	body := bytes.NewReader([]byte(`{"frequency_hz":5000000000}`))
	resp, err := http.Post(base+"/api/v1/scanner/manual_tune", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
}

func TestScannerManualTune_503WhenNoConvScanner(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{convNotConfigured: true}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	body := bytes.NewReader([]byte(`{"frequency_hz":155895000}`))
	resp, err := http.Post(base+"/api/v1/scanner/manual_tune", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Errorf("status=%d, want 503", resp.StatusCode)
	}
}

func TestScannerClearManualTune(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{clearOK: true}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock, AllowMutations: true})
	defer teardown()

	req, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/scanner/manual_tune/3", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
	if len(cock.clearedManual) != 1 || cock.clearedManual[0] != 3 {
		t.Errorf("ClearManualTune not routed: %v", cock.clearedManual)
	}

	// Same call with clearOK=false → 404.
	cock.clearOK = false
	req, _ = http.NewRequest(http.MethodDelete, base+"/api/v1/scanner/manual_tune/3", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("clearOK=false status=%d, want 404", resp.StatusCode)
	}
}

func TestScannerManualTune_Gated(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeCockpit{}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Scanner: cock /* AllowMutations:false */})
	defer teardown()

	body := bytes.NewReader([]byte(`{"frequency_hz":155895000}`))
	resp, err := http.Post(base+"/api/v1/scanner/manual_tune", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("status=%d, want 403", resp.StatusCode)
	}
	if len(cock.manualReqs) != 0 {
		t.Errorf("ManualTune called despite gate")
	}
}

// suppress "unused import" issues; using io / fmt may be needed via bytes.
var _ = io.Discard
var _ = fmt.Sprintf
