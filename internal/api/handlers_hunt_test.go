package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

type fakeHuntCockpit struct {
	status     HuntStatus
	startID    int
	startErr   error
	stopped    bool
	exportData []byte
	exportName string
	exportErr  error
	commitChg  []string
	commitErr  error
	rrReport   HuntRRReport
	rrErr      error
	captureRes HuntCaptureResult
	captureErr error

	lastStart   HuntStartRequest
	lastFormat  string
	lastID      int
	lastCapture HuntCaptureRequest
}

func (f *fakeHuntCockpit) Status() HuntStatus { return f.status }
func (f *fakeHuntCockpit) Start(req HuntStartRequest) (int, error) {
	f.lastStart = req
	return f.startID, f.startErr
}
func (f *fakeHuntCockpit) Stop() bool { return f.stopped }
func (f *fakeHuntCockpit) Export(id int, format string) ([]byte, string, error) {
	f.lastID = id
	f.lastFormat = format
	return f.exportData, f.exportName, f.exportErr
}
func (f *fakeHuntCockpit) ExportSurvey(id int, format string) ([]byte, string, error) {
	f.lastID = id
	f.lastFormat = format
	return f.exportData, f.exportName, f.exportErr
}
func (f *fakeHuntCockpit) Commit(id int, force, dryRun bool) ([]string, error) {
	f.lastID = id
	return f.commitChg, f.commitErr
}

func (f *fakeHuntCockpit) CaptureSignal(req HuntCaptureRequest) (HuntCaptureResult, error) {
	f.lastCapture = req
	return f.captureRes, f.captureErr
}

func (f *fakeHuntCockpit) RadioReference(id, countyID int, checkSIDs []int) (HuntRRReport, error) {
	f.lastID = id
	return f.rrReport, f.rrErr
}

func TestHuntStatus_NoCockpitReturnsIdle(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/hunt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var st HuntStatus
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.State != "idle" {
		t.Errorf("state=%q, want idle", st.State)
	}
}

func TestHuntStatus_ReportsRun(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{status: HuntStatus{RunID: 3, State: "running", Running: true, Sites: 2, Talkgroups: 7}}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/hunt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var st HuntStatus
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.RunID != 3 || !st.Running || st.Sites != 2 || st.Talkgroups != 7 {
		t.Errorf("decoded = %+v", st)
	}
}

func TestHuntStart_GatedAndApplies(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{startID: 5}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock, AllowMutations: true})
	defer teardown()
	resp, err := http.Post(base+"/api/v1/hunt/start", "application/json",
		strings.NewReader(`{"bands":["851:869"],"name":"X"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		OK    bool `json:"ok"`
		RunID int  `json:"run_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || body.RunID != 5 {
		t.Errorf("body = %+v", body)
	}
	if len(cock.lastStart.Bands) != 1 || cock.lastStart.Name != "X" {
		t.Errorf("request not forwarded: %+v", cock.lastStart)
	}
}

func TestHuntCapture_GatedAndForwards(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{captureRes: HuntCaptureResult{
		Path: "/tmp/x.cfile", SampleRateHz: 2_400_000, CenterHz: 751_000_000, Samples: 4800, Identify: "p25 (0.82)",
	}}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock, AllowMutations: true})
	defer teardown()
	resp, err := http.Post(base+"/api/v1/hunt/capture", "application/json",
		strings.NewReader(`{"freq_hz":751000000,"seconds":5,"target":"siglab"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var got HuntCaptureResult
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Path != "/tmp/x.cfile" || got.Identify != "p25 (0.82)" {
		t.Errorf("result not returned: %+v", got)
	}
	if cock.lastCapture.FreqHz != 751_000_000 || cock.lastCapture.Target != "siglab" {
		t.Errorf("request not forwarded: %+v", cock.lastCapture)
	}
}

func TestHuntCapture_MissingFreq400(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock, AllowMutations: true})
	defer teardown()
	resp, err := http.Post(base+"/api/v1/hunt/capture", "application/json", strings.NewReader(`{"target":"siglab"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestHuntStart_ForwardsSurveyDetectFlags(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{startID: 7}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock, AllowMutations: true})
	defer teardown()
	resp, err := http.Post(base+"/api/v1/hunt/start", "application/json",
		strings.NewReader(`{"bands":["851:869"],"survey":true,"detect_wideband":true,"detect_encryption":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if !cock.lastStart.Survey || !cock.lastStart.DetectWideband || !cock.lastStart.DetectEncryption {
		t.Errorf("survey detect flags not forwarded: %+v", cock.lastStart)
	}
}

func TestHuntStart_NoCockpit503(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus, AllowMutations: true})
	defer teardown()
	resp, err := http.Post(base+"/api/v1/hunt/start", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", resp.StatusCode)
	}
}

func TestHuntStart_AlreadyRunningConflict(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{startErr: errors.New("hunt: a run is already in progress")}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock, AllowMutations: true})
	defer teardown()
	resp, err := http.Post(base+"/api/v1/hunt/start", "application/json", strings.NewReader(`{"bands":["1:2"]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status=%d, want 409", resp.StatusCode)
	}
}

func TestHuntExport_StreamsBytes(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{exportData: []byte("# Section: metadata\n"), exportName: "sys.csv"}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/hunt/export?format=bundle")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "sys.csv") {
		t.Errorf("content-disposition=%q", cd)
	}
	if cock.lastFormat != "bundle" {
		t.Errorf("format=%q", cock.lastFormat)
	}
}

func TestHuntExport_ByRunID(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{exportData: []byte("x"), exportName: "sys.csv"}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/hunt/7/export?format=rr")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if cock.lastID != 7 || cock.lastFormat != "rr" {
		t.Errorf("forwarded id=%d format=%q, want 7/rr", cock.lastID, cock.lastFormat)
	}
}

func TestHuntExport_UnknownRunID404(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{exportErr: errors.New("hunt: no such run id")}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/hunt/999/export")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", resp.StatusCode)
	}
}

func TestHuntExport_BadRunID400(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{exportData: []byte("x"), exportName: "s.csv"}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/hunt/abc/export")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestHuntExport_NoSystemConflict(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{exportErr: errors.New("hunt: no discovered system to export yet")}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock})
	defer teardown()
	resp, err := http.Get(base + "/api/v1/hunt/export?format=bundle")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status=%d, want 409", resp.StatusCode)
	}
}

func TestHuntStop_OK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cock := &fakeHuntCockpit{stopped: true}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Hunt: cock, AllowMutations: true})
	defer teardown()
	resp, err := http.Post(base+"/api/v1/hunt/stop", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		Stopped bool `json:"stopped"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !body.Stopped {
		t.Error("stopped=false, want true")
	}
}
