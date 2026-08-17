package trunking

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// CSV writers for the talkgroup and RID catalogues.
//
// These exist so an operator who has named radios and talkgroups from a
// console can pull the result out as a file that drops straight into the
// system's talkgroup_file / rid_alias_file — the store is a convenience, not a
// lock-in. Round-trip tests pin every column against the matching LoadCSV, so
// a column that stops loading fails here first.

// talkgroupCSVHeader matches what the import writer emits, so a file exported
// from a running daemon is interchangeable with one produced by `import`.
var talkgroupCSVHeader = []string{
	"Decimal", "Hex", "Mode", "Alpha Tag", "Description", "Tag", "Group",
	"Priority", "Lockout", "Scan",
}

// ridCSVHeader lists the columns RIDDB.LoadCSV reads, in catalogue order.
var ridCSVHeader = []string{
	"Decimal", "Alias", "Description", "Tag", "Group", "Owner",
	"Priority", "Lockout", "Watch", "Icon",
}

// WriteTalkgroupCSV writes tgs as a talkgroup_file, sorted by id. A nil or
// empty slice still emits the header, so the file is loadable rather than
// zero-length.
func WriteTalkgroupCSV(w io.Writer, tgs []*TalkGroup) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(talkgroupCSVHeader); err != nil {
		return fmt.Errorf("trunking: write talkgroup csv header: %w", err)
	}
	sorted := sortedByID(tgs, func(tg *TalkGroup) uint32 { return tg.ID })
	for _, tg := range sorted {
		row := []string{
			strconv.FormatUint(uint64(tg.ID), 10),
			fmt.Sprintf("%X", tg.ID),
			tg.Mode,
			tg.AlphaTag,
			tg.Description,
			tg.Tag,
			tg.Group,
			optionalInt(tg.Priority),
			yesNo(tg.Lockout),
			yesNo(tg.Scan),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("trunking: write talkgroup csv row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteRIDCSV writes rids as a rid_alias_file, sorted by id.
func WriteRIDCSV(w io.Writer, rids []*RID) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(ridCSVHeader); err != nil {
		return fmt.Errorf("trunking: write rid csv header: %w", err)
	}
	sorted := sortedByID(rids, func(r *RID) uint32 { return r.ID })
	for _, r := range sorted {
		row := []string{
			strconv.FormatUint(uint64(r.ID), 10),
			r.Alias,
			r.Description,
			r.Tag,
			r.Group,
			r.Owner,
			optionalInt(r.Priority),
			yesNo(r.Lockout),
			yesNo(r.Watch),
			r.Icon,
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("trunking: write rid csv row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// sortedByID returns a copy ordered by id so an export is stable and diffable
// between runs (the catalogues are maps, so All() order is random).
func sortedByID[T any](in []*T, id func(*T) uint32) []*T {
	out := make([]*T, 0, len(in))
	for _, v := range in {
		if v != nil {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i]) < id(out[j]) })
	return out
}

// optionalInt renders 0 as an empty cell — the loaders treat a blank Priority
// as unset, and writing a literal 0 would be indistinguishable from a real
// priority in a hand-edited file.
func optionalInt(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}

// yesNo renders a boolean the way the loaders read it back.
func yesNo(b bool) string {
	if b {
		return "Y"
	}
	return "N"
}
