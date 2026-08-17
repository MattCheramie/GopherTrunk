package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func newLabelStore(t *testing.T) (*LabelStore, *events.Bus) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	s, err := NewLabelStore(db, bus)
	if err != nil {
		t.Fatal(err)
	}
	return s, bus
}

func TestLabelUpsertRoundTrip(t *testing.T) {
	s, _ := newLabelStore(t)
	ctx := context.Background()

	in := Label{
		Kind: LabelKindRID, System: "250_013", TargetID: 1005736,
		Name: "CPL-SMITH", Description: "patrol", Tag: "unit", Group: "police",
		Owner: "badge-41", Icon: "car",
	}
	got, err := s.Upsert(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == 0 || got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("upsert did not populate id/timestamps: %+v", got)
	}
	if got.Name != "CPL-SMITH" || got.Owner != "badge-41" || got.Group != "police" {
		t.Errorf("round-tripped label = %+v", got)
	}

	back, err := s.Get(ctx, LabelKindRID, "250_013", 1005736)
	if err != nil {
		t.Fatal(err)
	}
	if back.Name != got.Name || back.ID != got.ID {
		t.Errorf("Get = %+v, want %+v", back, got)
	}
}

func TestLabelUpsertReplacesExistingKey(t *testing.T) {
	s, _ := newLabelStore(t)
	ctx := context.Background()
	first, err := s.Upsert(ctx, Label{Kind: LabelKindTalkgroup, TargetID: 1020545, Name: "OLD"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Upsert(ctx, Label{Kind: LabelKindTalkgroup, TargetID: 1020545, Name: "NEW"})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Errorf("upsert created a second row (%d then %d)", first.ID, second.ID)
	}
	if second.Name != "NEW" {
		t.Errorf("name = %q, want NEW", second.Name)
	}
	all, err := s.List(ctx, LabelKindTalkgroup, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Errorf("rows = %d, want 1", len(all))
	}
}

// The (kind, system, id) key exists so the CSV export can emit one file per
// system: talkgroup_file / rid_alias_file are per-system config keys.
func TestLabelSameIDDifferentSystemsCoexist(t *testing.T) {
	s, _ := newLabelStore(t)
	ctx := context.Background()
	if _, err := s.Upsert(ctx, Label{Kind: LabelKindTalkgroup, System: "A", TargetID: 100, Name: "A-FIRE"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upsert(ctx, Label{Kind: LabelKindTalkgroup, System: "B", TargetID: 100, Name: "B-EMS"}); err != nil {
		t.Fatal(err)
	}
	a, err := s.Get(ctx, LabelKindTalkgroup, "A", 100)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Get(ctx, LabelKindTalkgroup, "B", 100)
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "A-FIRE" || b.Name != "B-EMS" {
		t.Errorf("systems collided: A=%q B=%q", a.Name, b.Name)
	}
}

// A system filter must also return the system-agnostic rows: both apply when
// that system is decoded.
func TestLabelListSystemFilterIncludesGlobalRows(t *testing.T) {
	s, _ := newLabelStore(t)
	ctx := context.Background()
	if _, err := s.Upsert(ctx, Label{Kind: LabelKindRID, System: "", TargetID: 1, Name: "GLOBAL"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upsert(ctx, Label{Kind: LabelKindRID, System: "250_013", TargetID: 2, Name: "SCOPED"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upsert(ctx, Label{Kind: LabelKindRID, System: "other", TargetID: 3, Name: "ELSEWHERE"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.List(ctx, LabelKindRID, "250_013")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, l := range got {
		names[l.Name] = true
	}
	if !names["GLOBAL"] || !names["SCOPED"] {
		t.Errorf("system filter dropped an applicable row: %v", names)
	}
	if names["ELSEWHERE"] {
		t.Errorf("system filter leaked another system's row: %v", names)
	}
}

func TestLabelRequiresKindAndTargetID(t *testing.T) {
	s, _ := newLabelStore(t)
	ctx := context.Background()
	if _, err := s.Upsert(ctx, Label{Kind: "nonsense", TargetID: 1, Name: "x"}); err == nil {
		t.Error("bad kind accepted")
	}
	if _, err := s.Upsert(ctx, Label{Kind: LabelKindRID, TargetID: 0, Name: "x"}); err == nil {
		t.Error("zero target_id accepted")
	}
}

func TestLabelDelete(t *testing.T) {
	s, _ := newLabelStore(t)
	ctx := context.Background()
	if _, err := s.Upsert(ctx, Label{Kind: LabelKindRID, TargetID: 7, Name: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, LabelKindRID, "", 7); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, LabelKindRID, "", 7); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Get after delete: err = %v, want sql.ErrNoRows", err)
	}
	if err := s.Delete(ctx, LabelKindRID, "", 7); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("second Delete: err = %v, want sql.ErrNoRows", err)
	}
}

func TestLabelMutationsPublishOnBus(t *testing.T) {
	s, bus := newLabelStore(t)
	sub := bus.Subscribe()
	defer sub.Close()
	ctx := context.Background()

	if _, err := s.Upsert(ctx, Label{Kind: LabelKindRID, TargetID: 42, Name: "x"}); err != nil {
		t.Fatal(err)
	}
	ev := <-sub.C
	if ev.Kind != events.KindLabelUpserted {
		t.Fatalf("kind = %s, want %s", ev.Kind, events.KindLabelUpserted)
	}
	l, ok := ev.Payload.(Label)
	if !ok || l.TargetID != 42 {
		t.Errorf("payload = %#v", ev.Payload)
	}
}

func TestLabelStoreWithoutBusStillWorks(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s, err := NewLabelStore(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upsert(context.Background(), Label{Kind: LabelKindRID, TargetID: 1, Name: "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestLabelEmpty(t *testing.T) {
	if !(Label{Kind: LabelKindRID, TargetID: 1}).Empty() {
		t.Error("a label with no content should report Empty")
	}
	if (Label{Kind: LabelKindRID, TargetID: 1, Tag: "unit"}).Empty() {
		t.Error("a label with a tag is not empty")
	}
}
