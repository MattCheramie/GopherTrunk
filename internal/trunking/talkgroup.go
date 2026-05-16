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

// TalkGroup describes one talkgroup loaded from disk. The schema follows
// the Trunk Recorder / RadioReference talkgroup CSV convention.
//
// Scan participates in the engine's per-talkgroup scan list when the
// engine runs in ScanModeList — only talkgroups with Scan == true get
// their grants followed (Emergency grants bypass the gate). In
// ScanModeAll (the default for backwards compat with pre-scanner
// configs) the field is moot. Defaults to true on every loader so a
// legacy CSV without a Scan column keeps the existing behavior.
type TalkGroup struct {
	ID          uint32 `json:"id"`
	AlphaTag    string `json:"alpha_tag"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`      // department / category
	Group       string `json:"group,omitempty"`    // top-level group
	Mode        string `json:"mode,omitempty"`     // D=digital, A=analog, M=mixed
	Priority    int    `json:"priority,omitempty"` // 1 = highest, 10 = lowest, 0 = unset
	Lockout     bool   `json:"lockout,omitempty"`
	Scan        bool   `json:"scan"`
}

// TalkgroupDB is a thread-safe lookup over loaded talkgroups.
type TalkgroupDB struct {
	mu  sync.RWMutex
	tgs map[uint32]*TalkGroup
}

// NewTalkgroupDB returns an empty DB.
func NewTalkgroupDB() *TalkgroupDB {
	return &TalkgroupDB{tgs: make(map[uint32]*TalkGroup)}
}

// Lookup returns the talkgroup record for id, or nil if unknown.
func (d *TalkgroupDB) Lookup(id uint32) *TalkGroup {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.tgs[id]
}

// Add or replace a single talkgroup record.
func (d *TalkgroupDB) Add(tg *TalkGroup) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tgs[tg.ID] = tg
}

// UpdateFields applies fn to the talkgroup with the given id under
// the write lock. Returns false if no such talkgroup exists. Used
// by the API to mutate Priority / Lockout without exposing the raw
// pointer to outside callers.
func (d *TalkgroupDB) UpdateFields(id uint32, fn func(*TalkGroup)) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	tg, ok := d.tgs[id]
	if !ok {
		return false
	}
	fn(tg)
	return true
}

// Delete removes the talkgroup with the given id. Returns false
// if no such talkgroup exists.
func (d *TalkgroupDB) Delete(id uint32) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.tgs[id]; !ok {
		return false
	}
	delete(d.tgs, id)
	return true
}

// All returns a snapshot of every talkgroup in the DB.
func (d *TalkgroupDB) All() []*TalkGroup {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]*TalkGroup, 0, len(d.tgs))
	for _, tg := range d.tgs {
		out = append(out, tg)
	}
	return out
}

// Len returns the number of loaded talkgroups.
func (d *TalkgroupDB) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.tgs)
}

// LoadCSV reads talkgroups from a Trunk-Recorder-style CSV.
// Required column: a numeric DEC/Decimal column. Optional columns
// (matched by header, case-insensitive): Alpha Tag, Description, Mode,
// Tag, Group, Priority, Lockout.
//
// A "Y" / "yes" / "true" value in Lockout sets the flag. Lockout is also
// inferred when Priority is set to a sentinel "L" value, matching common
// community CSVs.
func (d *TalkgroupDB) LoadCSV(r io.Reader) (int, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		return 0, fmt.Errorf("trunking: read csv header: %w", err)
	}
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	getDecCol := func() (int, bool) {
		for _, name := range []string{"decimal", "dec"} {
			if i, ok := colIdx[name]; ok {
				return i, true
			}
		}
		return 0, false
	}
	decCol, ok := getDecCol()
	if !ok {
		return 0, errors.New("trunking: csv missing required Decimal/DEC column")
	}

	loaded := 0
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return loaded, fmt.Errorf("trunking: read csv row: %w", err)
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
		tg := &TalkGroup{ID: uint32(idVal), Scan: true}
		tg.AlphaTag = field(row, colIdx, "alpha tag", "alphatag", "alpha_tag")
		tg.Description = field(row, colIdx, "description")
		tg.Mode = field(row, colIdx, "mode")
		tg.Tag = field(row, colIdx, "tag")
		tg.Group = field(row, colIdx, "group", "category")
		if pStr := field(row, colIdx, "priority"); pStr != "" {
			if strings.EqualFold(pStr, "L") {
				tg.Lockout = true
			} else if p, err := strconv.Atoi(pStr); err == nil {
				tg.Priority = p
			}
		}
		if l := field(row, colIdx, "lockout"); l != "" {
			switch strings.ToLower(l) {
			case "y", "yes", "true", "1":
				tg.Lockout = true
			}
		}
		// Optional Scan / Active column. Default is true; explicit
		// "no"/"false"/"0"/"n" turns it off so a CSV-shipped scan-list
		// can ride alongside an operator's runtime mutations.
		if s := field(row, colIdx, "scan", "active"); s != "" {
			switch strings.ToLower(s) {
			case "n", "no", "false", "0", "off":
				tg.Scan = false
			}
		}
		d.Add(tg)
		loaded++
	}
	return loaded, nil
}

// LoadCSVFile is a thin wrapper over LoadCSV for a path on disk.
func (d *TalkgroupDB) LoadCSVFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return d.LoadCSV(f)
}

// LoadJSON reads a JSON array of TalkGroup records. Records missing
// the "scan" key resolve to Scan=true so legacy JSON dumps keep the
// "follow every grant" behavior; explicit `"scan": false` turns off
// participation in the scan list.
func (d *TalkgroupDB) LoadJSON(r io.Reader) (int, error) {
	type talkGroupRaw struct {
		ID          uint32 `json:"id"`
		AlphaTag    string `json:"alpha_tag"`
		Description string `json:"description,omitempty"`
		Tag         string `json:"tag,omitempty"`
		Group       string `json:"group,omitempty"`
		Mode        string `json:"mode,omitempty"`
		Priority    int    `json:"priority,omitempty"`
		Lockout     bool   `json:"lockout,omitempty"`
		Scan        *bool  `json:"scan"`
	}
	var arr []talkGroupRaw
	if err := json.NewDecoder(r).Decode(&arr); err != nil {
		return 0, fmt.Errorf("trunking: decode json: %w", err)
	}
	for _, raw := range arr {
		tg := &TalkGroup{
			ID:          raw.ID,
			AlphaTag:    raw.AlphaTag,
			Description: raw.Description,
			Tag:         raw.Tag,
			Group:       raw.Group,
			Mode:        raw.Mode,
			Priority:    raw.Priority,
			Lockout:     raw.Lockout,
			Scan:        true,
		}
		if raw.Scan != nil {
			tg.Scan = *raw.Scan
		}
		d.Add(tg)
	}
	return len(arr), nil
}

func field(row []string, idx map[string]int, names ...string) string {
	for _, n := range names {
		if i, ok := idx[n]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}
