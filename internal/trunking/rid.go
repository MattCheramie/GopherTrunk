package trunking

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// RID describes one radio unit (subscriber unit identifier) loaded from
// disk. RIDs are the per-radio analogue of talkgroups — a way to give
// operator-meaningful names, owners, and grouping to the numeric
// SourceID that rides on every grant/affiliation/registration.
//
// The schema intentionally mirrors TalkGroup's fields where it makes
// sense (Alias↔AlphaTag, Tag, Group, Priority, Lockout, Icon) so the
// CSV/JSON loaders share the same conventions, and an operator who
// already maintains a talkgroup catalog can drop a parallel RID file
// next to it.
type RID struct {
	ID          uint32 `json:"id"`
	Alias       string `json:"alias"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`   // department / role
	Group       string `json:"group,omitempty"` // top-level grouping (agency)
	Owner       string `json:"owner,omitempty"` // operator/badge assigned to the radio
	Priority    int    `json:"priority,omitempty"`
	// Lockout marks the radio as stale / decommissioned / known-bad so
	// the UI can de-emphasise it. RIDs are not gated like talkgroups
	// today — Lockout is informational only.
	Lockout bool `json:"lockout,omitempty"`
	// Watch flags RIDs of operator interest so they can be surfaced in
	// a watch-list UI. Defaults to true on every loader so a plain
	// catalogue without a Watch column keeps every RID visible.
	Watch bool `json:"watch"`
	// Icon is an optional glyph identifier used by the operator UIs to
	// render a per-RID icon.
	Icon string `json:"icon,omitempty"`
}

// RIDDB is a thread-safe lookup over loaded RIDs.
type RIDDB struct {
	mu   sync.RWMutex
	rids map[uint32]*RID
}

// NewRIDDB returns an empty DB.
func NewRIDDB() *RIDDB {
	return &RIDDB{rids: make(map[uint32]*RID)}
}

// Lookup returns the RID record for id, or nil if unknown.
func (d *RIDDB) Lookup(id uint32) *RID {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.rids[id]
}

// Add or replace a single RID record.
func (d *RIDDB) Add(r *RID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rids[r.ID] = r
}

// UpdateFields applies fn to the RID with the given id under the write
// lock. Returns false if no such RID exists.
func (d *RIDDB) UpdateFields(id uint32, fn func(*RID)) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	r, ok := d.rids[id]
	if !ok {
		return false
	}
	fn(r)
	return true
}

// Delete removes the RID with the given id. Returns false if no such
// RID exists.
func (d *RIDDB) Delete(id uint32) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.rids[id]; !ok {
		return false
	}
	delete(d.rids, id)
	return true
}

// All returns a snapshot of every RID in the DB.
func (d *RIDDB) All() []*RID {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]*RID, 0, len(d.rids))
	for _, r := range d.rids {
		out = append(out, r)
	}
	return out
}

// Len returns the number of loaded RIDs.
func (d *RIDDB) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.rids)
}

// LoadCSV reads RIDs from a CSV.
// Required column: a numeric Decimal/DEC column with the radio ID.
// Optional columns (case-insensitive, matched by header):
// Alias / Alpha Tag, Description, Tag, Group, Owner, Priority,
// Lockout, Watch, Icon.
func (d *RIDDB) LoadCSV(r io.Reader) (int, error) {
	cr := csv.NewReader(decodeReader(r))
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		// Empty file → legitimate "no RIDs" state, mirrors TalkgroupDB.
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("trunking: read rid csv header: %w", err)
	}
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	decCol := -1
	for _, name := range []string{"decimal", "dec", "id"} {
		if i, ok := colIdx[name]; ok {
			decCol = i
			break
		}
	}
	if decCol < 0 {
		return 0, errors.New("trunking: rid csv missing required Decimal/DEC/ID column")
	}

	loaded := 0
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return loaded, fmt.Errorf("trunking: read rid csv row: %w", err)
		}
		if decCol >= len(row) {
			continue
		}
		decStr := strings.TrimSpace(row[decCol])
		if decStr == "" {
			continue
		}
		idVal, err := strconv.ParseUint(decStr, 10, 32)
		if err != nil {
			continue // skip malformed rows
		}
		rid := &RID{ID: uint32(idVal), Watch: true}
		rid.Alias = field(row, colIdx, "alias", "alpha tag", "alphatag", "alpha_tag")
		rid.Description = field(row, colIdx, "description")
		rid.Tag = field(row, colIdx, "tag")
		rid.Group = field(row, colIdx, "group", "category", "agency")
		rid.Owner = field(row, colIdx, "owner", "user", "operator")
		if pStr := field(row, colIdx, "priority"); pStr != "" {
			if strings.EqualFold(pStr, "L") {
				rid.Lockout = true
			} else if p, err := strconv.Atoi(pStr); err == nil {
				rid.Priority = p
			}
		}
		if l := field(row, colIdx, "lockout"); l != "" {
			switch strings.ToLower(l) {
			case "y", "yes", "true", "1":
				rid.Lockout = true
			}
		}
		// Watch defaults true; explicit "no"/"false"/"0"/"n" turns it off.
		if s := field(row, colIdx, "watch", "active"); s != "" {
			switch strings.ToLower(s) {
			case "n", "no", "false", "0", "off":
				rid.Watch = false
			}
		}
		rid.Icon = field(row, colIdx, "icon")
		d.Add(rid)
		loaded++
	}
	return loaded, nil
}

// LoadCSVFile is a thin wrapper over LoadCSV for a path on disk.
func (d *RIDDB) LoadCSVFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return d.LoadCSV(f)
}

// LoadJSON reads a JSON array of RID records. Records missing the
// "watch" key resolve to Watch=true so legacy JSON dumps keep every
// RID visible; explicit `"watch": false` opts a record out of the
// watch list.
func (d *RIDDB) LoadJSON(r io.Reader) (int, error) {
	type ridRaw struct {
		ID          uint32 `json:"id"`
		Alias       string `json:"alias"`
		Description string `json:"description,omitempty"`
		Tag         string `json:"tag,omitempty"`
		Group       string `json:"group,omitempty"`
		Owner       string `json:"owner,omitempty"`
		Priority    int    `json:"priority,omitempty"`
		Lockout     bool   `json:"lockout,omitempty"`
		Watch       *bool  `json:"watch"`
		Icon        string `json:"icon,omitempty"`
	}
	var arr []ridRaw
	if err := json.NewDecoder(decodeReader(r)).Decode(&arr); err != nil {
		return 0, fmt.Errorf("trunking: decode rid json: %w", err)
	}
	for _, raw := range arr {
		rid := &RID{
			ID:          raw.ID,
			Alias:       sanitizeText(raw.Alias),
			Description: sanitizeText(raw.Description),
			Tag:         sanitizeText(raw.Tag),
			Group:       sanitizeText(raw.Group),
			Owner:       sanitizeText(raw.Owner),
			Priority:    raw.Priority,
			Lockout:     raw.Lockout,
			Watch:       true,
			Icon:        sanitizeText(raw.Icon),
		}
		if raw.Watch != nil {
			rid.Watch = *raw.Watch
		}
		d.Add(rid)
	}
	return len(arr), nil
}

// LoadJSONFile is a thin wrapper over LoadJSON for a path on disk.
func (d *RIDDB) LoadJSONFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return d.LoadJSON(f)
}
