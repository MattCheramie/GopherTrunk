package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func labelTestStore(t *testing.T) (*storage.LabelStore, func()) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "labels.db"))
	if err != nil {
		t.Fatal(err)
	}
	bus := events.NewBus(8)
	s, err := storage.NewLabelStore(db, bus)
	if err != nil {
		t.Fatal(err)
	}
	return s, func() {
		bus.Close()
		db.Close()
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// The operator's case: a radio seen on the air, in no rid_alias_file, named
// from a console. On the next start there is no catalogue entry to update, so
// the merge has to create one or the name is lost.
func TestApplyLabelsCreatesEntryForAnUncataloguedRadio(t *testing.T) {
	store, cleanup := labelTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := store.Upsert(ctx, storage.Label{
		Kind: storage.LabelKindRID, System: "250_013", TargetID: 1005736,
		Name: "CPL-SMITH", Owner: "badge-41",
	}); err != nil {
		t.Fatal(err)
	}

	rids := trunking.NewRIDDB()
	tgs := trunking.NewTalkgroupDB()
	applied, created, err := applyLabels(ctx, store, rids, tgs, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 || created != 1 {
		t.Errorf("applied=%d created=%d, want 0/1", applied, created)
	}
	rec := rids.Lookup(1005736)
	if rec == nil {
		t.Fatal("radio was not created from its label")
	}
	if rec.Alias != "CPL-SMITH" || rec.Owner != "badge-41" {
		t.Errorf("record = %+v", *rec)
	}
	// Watch defaults true on every loader; a created record has to match.
	if !rec.Watch {
		t.Error("created RID has Watch=false, want the loader default")
	}
}

// The label is the operator's most recent explicit act, and the only one they
// can perform without a restart, so it wins over the alias file's value.
func TestApplyLabelsOverridesTheAliasFileValue(t *testing.T) {
	store, cleanup := labelTestStore(t)
	defer cleanup()
	ctx := context.Background()

	rids := trunking.NewRIDDB()
	rids.Add(&trunking.RID{ID: 42, Alias: "FROM-FILE", Tag: "file-tag", Watch: true})
	if _, err := store.Upsert(ctx, storage.Label{
		Kind: storage.LabelKindRID, TargetID: 42, Name: "RENAMED",
	}); err != nil {
		t.Fatal(err)
	}

	applied, created, err := applyLabels(ctx, store, rids, trunking.NewTalkgroupDB(), discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 || created != 0 {
		t.Errorf("applied=%d created=%d, want 1/0", applied, created)
	}
	rec := rids.Lookup(42)
	if rec.Alias != "RENAMED" {
		t.Errorf("alias = %q, want RENAMED", rec.Alias)
	}
	// A label that only sets a name must not blank out fields the file
	// supplied — otherwise naming a radio silently drops its tag/owner.
	if rec.Tag != "file-tag" {
		t.Errorf("tag = %q, want the alias file's file-tag to survive", rec.Tag)
	}
}

func TestApplyLabelsCreatesTalkgroupWithLoaderDefaults(t *testing.T) {
	store, cleanup := labelTestStore(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := store.Upsert(ctx, storage.Label{
		Kind: storage.LabelKindTalkgroup, TargetID: 1020545, Name: "TAC-1",
	}); err != nil {
		t.Fatal(err)
	}
	tgs := trunking.NewTalkgroupDB()
	if _, _, err := applyLabels(ctx, store, trunking.NewRIDDB(), tgs, discardLogger()); err != nil {
		t.Fatal(err)
	}
	tg := tgs.Lookup(1020545)
	if tg == nil {
		t.Fatal("talkgroup was not created from its label")
	}
	if tg.AlphaTag != "TAC-1" {
		t.Errorf("alpha tag = %q", tg.AlphaTag)
	}
	if !tg.Scan || !tg.Stream || !tg.Record {
		t.Errorf("created talkgroup = %+v, want the loader's scan/stream/record defaults", *tg)
	}
	// DeleteDiscovered would retract an operator's name if the created row
	// were tagged Discovered.
	if tgs.DeleteDiscovered(1020545) {
		t.Error("a labelled talkgroup was swept by DeleteDiscovered")
	}
}

// A daemon with no storage.path has no label store, and the merge has to be a
// no-op rather than a panic.
func TestApplyLabelsWithoutAStoreIsANoOp(t *testing.T) {
	applied, created, err := applyLabels(context.Background(), nil,
		trunking.NewRIDDB(), trunking.NewTalkgroupDB(), discardLogger())
	if err != nil || applied != 0 || created != 0 {
		t.Errorf("applied=%d created=%d err=%v, want 0/0/nil", applied, created, err)
	}
}
