// Package configbuilder holds the HTTP-free core shared by the web Config
// Builder (internal/api) and the terminal Config Builder (internal/configtui):
// talkgroup CSV sidecar IO, the RadioReference→config projection, the
// canonical section list + validation-error section bucketing, frequency
// formatting, and the AdvancedJSON field allow-list. Keeping these here means
// the two builders cannot disagree on behaviour.
package configbuilder

import (
	"bytes"
	"encoding/csv"
	"os"
	"strconv"
	"strings"
)

// TalkgroupCSVRow is one row of a Trunk Recorder–style talkgroup CSV sidecar.
// The json tags are the wire shape the web builder exchanges, so internal/api
// aliases this type to keep its responses byte-identical.
type TalkgroupCSVRow struct {
	Decimal     uint32 `json:"decimal"`
	AlphaTag    string `json:"alpha_tag,omitempty"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

// ReadTalkgroupCSV parses a Trunk Recorder–style talkgroup CSV into rows,
// mapping columns by (case/space-insensitive) header name so column order and
// extra columns are tolerated. Rows without a numeric Decimal are skipped.
func ReadTalkgroupCSV(path string) ([]TalkgroupCSVRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cr := csv.NewReader(f)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	idx := make(map[string]int, len(records[0]))
	for i, h := range records[0] {
		idx[normalizeHeader(h)] = i
	}
	col := func(rec []string, key string) string {
		if j, ok := idx[key]; ok && j < len(rec) {
			return strings.TrimSpace(rec[j])
		}
		return ""
	}
	var rows []TalkgroupCSVRow
	for _, rec := range records[1:] {
		decStr := col(rec, "decimal")
		if decStr == "" {
			continue
		}
		dec, err := strconv.ParseUint(decStr, 10, 32)
		if err != nil {
			continue
		}
		rows = append(rows, TalkgroupCSVRow{
			Decimal:     uint32(dec),
			AlphaTag:    col(rec, "alphatag"),
			Description: col(rec, "description"),
			Tag:         col(rec, "tag"),
			Group:       col(rec, "group"),
			Mode:        col(rec, "mode"),
		})
	}
	return rows, nil
}

func normalizeHeader(h string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(h)), " ", "")
}

// WriteTalkgroupCSV writes a Trunk Recorder–style talkgroup CSV that
// trunking.TalkGroup loads (header columns it recognises).
func WriteTalkgroupCSV(path string, rows []TalkgroupCSVRow) error {
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	_ = cw.Write([]string{"Decimal", "Alpha Tag", "Description", "Tag", "Group", "Mode"})
	for _, row := range rows {
		_ = cw.Write([]string{
			strconv.FormatUint(uint64(row.Decimal), 10),
			row.AlphaTag, row.Description, row.Tag, row.Group, row.Mode,
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
