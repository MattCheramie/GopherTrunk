package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeLabelStore is an in-memory LabelProvider keyed the same way the SQLite
// table is, so the handlers' key handling is exercised without a database.
type fakeLabelStore struct {
	mu   sync.Mutex
	rows map[string]storage.Label
	err  error
}

func newFakeLabelStore() *fakeLabelStore {
	return &fakeLabelStore{rows: map[string]storage.Label{}}
}

func (f *fakeLabelStore) key(k storage.LabelKind, system string, id uint32) string {
	return string(k) + "|" + system + "|" + string(rune(id))
}

func (f *fakeLabelStore) ListLabels(kind storage.LabelKind, system string) ([]storage.Label, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	var out []storage.Label
	for _, l := range f.rows {
		if kind != "" && l.Kind != kind {
			continue
		}
		if system != "" && l.System != "" && l.System != system {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

func (f *fakeLabelStore) UpsertLabel(l storage.Label) (storage.Label, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return storage.Label{}, f.err
	}
	l.ID = int64(len(f.rows) + 1)
	f.rows[f.key(l.Kind, l.System, l.TargetID)] = l
	return l, nil
}

func (f *fakeLabelStore) DeleteLabel(kind storage.LabelKind, system string, id uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := f.key(kind, system, id)
	if _, ok := f.rows[k]; !ok {
		return errors.New("sql: no rows in result set")
	}
	delete(f.rows, k)
	return nil
}

func (f *fakeLabelStore) get(kind storage.LabelKind, system string, id uint32) (storage.Label, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.rows[f.key(kind, system, id)]
	return l, ok
}

func patchJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
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

// Naming a talkgroup was impossible before: the PATCH body had no field for it.
func TestUpdateTalkgroupAppliesAlphaTagAndDescription(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	tgs := trunking.NewTalkgroupDB()
	tgs.Add(&trunking.TalkGroup{ID: 1020545, Scan: true})
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, Talkgroups: tgs, AllowMutations: true,
	})
	defer teardown()

	resp := patchJSON(t, base+"/api/v1/talkgroups/1020545",
		map[string]any{"alpha_tag": "TAC-1", "description": "tactical"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (%s)", resp.StatusCode, body)
	}
	got := tgs.Lookup(1020545)
	if got.AlphaTag != "TAC-1" || got.Description != "tactical" {
		t.Errorf("talkgroup = %+v, want the name applied", *got)
	}
}

func TestUpdateTalkgroupCreatesMissingTalkgroup(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	tgs := trunking.NewTalkgroupDB()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, Talkgroups: tgs, AllowMutations: true,
	})
	defer teardown()

	resp := patchJSON(t, base+"/api/v1/talkgroups/777", map[string]any{"alpha_tag": "NEW"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	got := tgs.Lookup(777)
	if got == nil {
		t.Fatal("talkgroup was not created")
	}
	if got.AlphaTag != "NEW" {
		t.Errorf("alpha tag = %q", got.AlphaTag)
	}
	// The CSV loader's defaults, so a synthesised row behaves like a loaded
	// one rather than silently opting the talkgroup out of scan/record.
	if !got.Scan || !got.Stream || !got.Record {
		t.Errorf("synthesised talkgroup = %+v, want scan/stream/record defaults", *got)
	}
	// Tagging it Discovered would let DeleteDiscovered retract the operator's
	// name behind their back.
	if strings.EqualFold(got.Tag, "discovered") {
		t.Error("synthesised talkgroup was tagged Discovered")
	}
}

func TestUpdateRIDPersistsLabel(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	store := newFakeLabelStore()
	rids := trunking.NewRIDDB()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: rids, Labels: store, AllowMutations: true,
	})
	defer teardown()

	resp := patchJSON(t, base+"/api/v1/rids/1005736?system=250_013",
		map[string]any{"alias": "CPL-SMITH", "owner": "badge-41"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	l, ok := store.get(storage.LabelKindRID, "250_013", 1005736)
	if !ok {
		t.Fatal("label was not persisted")
	}
	if l.Name != "CPL-SMITH" || l.Owner != "badge-41" {
		t.Errorf("persisted label = %+v", l)
	}
}

// Clearing a name has to remove the row, not store an empty one — an empty
// label would otherwise shadow whatever the alias file supplies at startup.
func TestUpdateRIDClearingTheNameRemovesTheLabel(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	store := newFakeLabelStore()
	rids := trunking.NewRIDDB()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: rids, Labels: store, AllowMutations: true,
	})
	defer teardown()

	resp := patchJSON(t, base+"/api/v1/rids/42", map[string]any{"alias": "TEMP"})
	resp.Body.Close()
	if _, ok := store.get(storage.LabelKindRID, "", 42); !ok {
		t.Fatal("label was not persisted in the first place")
	}
	resp = patchJSON(t, base+"/api/v1/rids/42", map[string]any{"alias": ""})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if l, ok := store.get(storage.LabelKindRID, "", 42); ok {
		t.Errorf("cleared label was kept: %+v", l)
	}
}

// Without a label store the rename must still work in memory — that is the
// behaviour every daemon without storage.path has always had.
func TestUpdateRIDWithoutLabelStoreStillMutatesMemory(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	rids := trunking.NewRIDDB()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: rids, AllowMutations: true,
	})
	defer teardown()

	resp := patchJSON(t, base+"/api/v1/rids/9", map[string]any{"alias": "X"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if rec := rids.Lookup(9); rec == nil || rec.Alias != "X" {
		t.Errorf("in-memory rename did not take: %+v", rec)
	}
}

func TestLabelExportRIDCSVRoundTripsIntoRIDDB(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	store := newFakeLabelStore()
	rids := trunking.NewRIDDB()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: rids, Labels: store, AllowMutations: true,
	})
	defer teardown()

	patchJSON(t, base+"/api/v1/rids/1005736", map[string]any{"alias": "CPL-SMITH"}).Body.Close()
	patchJSON(t, base+"/api/v1/rids/77", map[string]any{"alias": "SGT-JONES"}).Body.Close()

	resp := mustGet(t, base+"/api/v1/labels/export?kind=rid")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("content-type = %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") ||
		!strings.Contains(cd, "gophertrunk-rid-all.csv") {
		t.Errorf("content-disposition = %q", cd)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	// The point of the export is that it loads back into an alias file.
	back := trunking.NewRIDDB()
	n, err := back.LoadCSV(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("exported CSV does not load: %v\n%s", err, body)
	}
	if n != 2 {
		t.Fatalf("loaded %d rows, want 2\n%s", n, body)
	}
	if rec := back.Lookup(1005736); rec == nil || rec.Alias != "CPL-SMITH" {
		t.Errorf("round-tripped rid = %+v", rec)
	}
}

// scope=labels must not dump the whole catalogue: an operator exporting their
// own names does not want a system's entire imported talkgroup list back.
func TestLabelExportScopeLabelsOnlyIncludesLabelledIDs(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	store := newFakeLabelStore()
	tgs := trunking.NewTalkgroupDB()
	tgs.Add(&trunking.TalkGroup{ID: 100, AlphaTag: "FROM-FILE", Scan: true})
	tgs.Add(&trunking.TalkGroup{ID: 200, AlphaTag: "ALSO-FILE", Scan: true})
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, Talkgroups: tgs, Labels: store, AllowMutations: true,
	})
	defer teardown()

	patchJSON(t, base+"/api/v1/talkgroups/100", map[string]any{"alpha_tag": "RENAMED"}).Body.Close()

	rows := exportRows(t, base+"/api/v1/labels/export?kind=talkgroup&scope=labels")
	if len(rows) != 1 {
		t.Fatalf("scope=labels exported %d rows, want 1: %v", len(rows), rows)
	}
	if rows[0][0] != "100" {
		t.Errorf("exported id = %s, want 100", rows[0][0])
	}

	all := exportRows(t, base+"/api/v1/labels/export?kind=talkgroup&scope=all")
	if len(all) != 2 {
		t.Errorf("scope=all exported %d rows, want 2", len(all))
	}
}

func TestLabelExportRejectsBadParams(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: trunking.NewRIDDB(), Labels: newFakeLabelStore(),
	})
	defer teardown()

	for _, url := range []string{
		base + "/api/v1/labels/export",
		base + "/api/v1/labels/export?kind=nonsense",
		base + "/api/v1/labels/export?kind=rid&scope=nonsense",
	} {
		resp := mustGet(t, url)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("GET %s: status = %d, want 400", url, resp.StatusCode)
		}
	}
}

func TestLabelRoutesReturn503WithoutAStore(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: trunking.NewRIDDB(), AllowMutations: true,
	})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/labels")
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("GET /labels status = %d, want 503", resp.StatusCode)
	}
	// scope=all needs no store — it reads the live catalogue — so it must
	// still work, which is the escape hatch the 503 message points at.
	resp = mustGet(t, base+"/api/v1/labels/export?kind=rid&scope=all")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("scope=all export status = %d, want 200", resp.StatusCode)
	}
}

func TestDeleteLabelRemovesTheRow(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	store := newFakeLabelStore()
	rids := trunking.NewRIDDB()
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: rids, Labels: store, AllowMutations: true,
	})
	defer teardown()

	patchJSON(t, base+"/api/v1/rids/5", map[string]any{"alias": "X"}).Body.Close()

	req, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/labels/rid/5", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if _, ok := store.get(storage.LabelKindRID, "", 5); ok {
		t.Error("label row survived the delete")
	}

	// A second delete is a 404, not a silent success.
	req, _ = http.NewRequest(http.MethodDelete, base+"/api/v1/labels/rid/5", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("second delete status = %d, want 404", resp.StatusCode)
	}
}

// exportRows fetches a CSV export and returns its data rows (header dropped).
func exportRows(t *testing.T, url string) [][]string {
	t.Helper()
	resp := mustGet(t, url)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status = %d", url, resp.StatusCode)
	}
	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		t.Fatalf("export is not valid CSV: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("export had no header")
	}
	return rows[1:]
}
