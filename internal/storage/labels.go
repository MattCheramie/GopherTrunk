// Operator-applied names for radios and talkgroups. Persisted to SQLite
// alongside the call log so a name applied from a console while watching live
// traffic is still there after a restart.
//
// These layer OVER the per-system talkgroup_file / rid_alias_file catalogues
// rather than replacing them: the daemon loads the files first, then applies
// label rows on top (see the daemon's label merge). An operator who prefers to
// keep a single hand-edited file can export the merged result back to CSV and
// stop using the store.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// LabelKind distinguishes the two catalogues a label can target.
type LabelKind string

const (
	LabelKindRID       LabelKind = "rid"
	LabelKindTalkgroup LabelKind = "talkgroup"
)

// ParseLabelKind validates a wire/query value.
func ParseLabelKind(s string) (LabelKind, error) {
	switch LabelKind(s) {
	case LabelKindRID:
		return LabelKindRID, nil
	case LabelKindTalkgroup:
		return LabelKindTalkgroup, nil
	}
	return "", fmt.Errorf("storage/labels: unknown kind %q (want rid or talkgroup)", s)
}

// Label is one operator-applied name plus the descriptive fields that travel
// with it. Policy fields (priority, lockout, scan, watch) are deliberately NOT
// here: those are existing in-memory mutations, and making them durable is a
// separate behaviour change.
type Label struct {
	ID   int64     `json:"id"`
	Kind LabelKind `json:"kind"`
	// System scopes the label to one trunking system. Empty applies it
	// wherever the id is seen, which matches the daemon's single merged
	// catalogue.
	System   string `json:"system,omitempty"`
	TargetID uint32 `json:"target_id"`
	// Name is the RID alias or the talkgroup alpha tag.
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Tag         string    `json:"tag,omitempty"`
	Group       string    `json:"group,omitempty"`
	Owner       string    `json:"owner,omitempty"` // radios only
	Icon        string    `json:"icon,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Empty reports whether a label carries no operator content, i.e. saving it
// would store nothing worth reloading. Callers delete instead of upserting.
func (l Label) Empty() bool {
	return l.Name == "" && l.Description == "" && l.Tag == "" &&
		l.Group == "" && l.Owner == "" && l.Icon == ""
}

// LabelStore is the CRUD layer over the labels table.
type LabelStore struct {
	db  *DB
	bus *events.Bus
}

// NewLabelStore returns a store backed by the given DB. The bus is optional —
// when nil no mutation events are published.
func NewLabelStore(db *DB, bus *events.Bus) (*LabelStore, error) {
	if db == nil {
		return nil, errors.New("storage/labels: DB is required")
	}
	return &LabelStore{db: db, bus: bus}, nil
}

// Upsert writes a complete label row, replacing any existing row for the same
// (kind, system, target_id). The caller supplies the whole record — it builds
// it from the already-merged in-memory catalogue entry — so there is no
// read-modify-write here and no lost-update race between two consoles.
func (s *LabelStore) Upsert(ctx context.Context, l Label) (Label, error) {
	if _, err := ParseLabelKind(string(l.Kind)); err != nil {
		return Label{}, err
	}
	if l.TargetID == 0 {
		return Label{}, errors.New("storage/labels: target_id is required")
	}
	l.System = strings.TrimSpace(l.System)
	now := time.Now()
	const q = `
INSERT INTO labels (kind, system, target_id, name, description, tag, grouping, owner, icon, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(kind, system, target_id) DO UPDATE SET
    name        = excluded.name,
    description = excluded.description,
    tag         = excluded.tag,
    grouping    = excluded.grouping,
    owner       = excluded.owner,
    icon        = excluded.icon,
    updated_at  = excluded.updated_at`
	if _, err := s.db.SQL().ExecContext(ctx, q,
		string(l.Kind), l.System, l.TargetID, l.Name, l.Description, l.Tag,
		l.Group, l.Owner, l.Icon, now.UnixNano(), now.UnixNano(),
	); err != nil {
		return Label{}, fmt.Errorf("storage/labels: upsert: %w", err)
	}
	out, err := s.Get(ctx, l.Kind, l.System, l.TargetID)
	if err != nil {
		return Label{}, err
	}
	s.publish(events.KindLabelUpserted, out)
	return out, nil
}

// Get returns one label, or sql.ErrNoRows when it is missing.
func (s *LabelStore) Get(ctx context.Context, kind LabelKind, system string, id uint32) (Label, error) {
	row := s.db.SQL().QueryRowContext(ctx, `
SELECT id, kind, system, target_id, name, description, tag, grouping, owner, icon, created_at, updated_at
  FROM labels WHERE kind = ? AND system = ? AND target_id = ?`,
		string(kind), system, id)
	return scanLabel(row)
}

// List returns labels of one kind, newest-updated first. An empty kind returns
// both kinds. A non-empty system returns that system's labels AND the
// system-agnostic ones, since both apply to it.
func (s *LabelStore) List(ctx context.Context, kind LabelKind, system string) ([]Label, error) {
	q := `
SELECT id, kind, system, target_id, name, description, tag, grouping, owner, icon, created_at, updated_at
  FROM labels WHERE 1=1`
	var args []any
	if kind != "" {
		q += " AND kind = ?"
		args = append(args, string(kind))
	}
	if system != "" {
		q += " AND system IN ('', ?)"
		args = append(args, system)
	}
	q += " ORDER BY kind, target_id"

	rows, err := s.db.SQL().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage/labels: list: %w", err)
	}
	defer rows.Close()
	var out []Label
	for rows.Next() {
		l, err := scanLabel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Delete removes one label. Returns sql.ErrNoRows when there was none.
func (s *LabelStore) Delete(ctx context.Context, kind LabelKind, system string, id uint32) error {
	res, err := s.db.SQL().ExecContext(ctx,
		`DELETE FROM labels WHERE kind = ? AND system = ? AND target_id = ?`,
		string(kind), system, id)
	if err != nil {
		return fmt.Errorf("storage/labels: delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage/labels: delete rows: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	s.publish(events.KindLabelDeleted, Label{Kind: kind, System: system, TargetID: id})
	return nil
}

func (s *LabelStore) publish(kind events.Kind, l Label) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(events.Event{Kind: kind, Payload: l})
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanLabel(row scanner) (Label, error) {
	var (
		l         Label
		kind      string
		createdNs int64
		updatedNs int64
	)
	if err := row.Scan(&l.ID, &kind, &l.System, &l.TargetID, &l.Name, &l.Description,
		&l.Tag, &l.Group, &l.Owner, &l.Icon, &createdNs, &updatedNs); err != nil {
		return Label{}, err
	}
	l.Kind = LabelKind(kind)
	l.CreatedAt = time.Unix(0, createdNs)
	l.UpdatedAt = time.Unix(0, updatedNs)
	return l, nil
}
