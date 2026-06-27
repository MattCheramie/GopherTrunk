package motorola

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/propagate"
)

// Row is one corpus record after CRC stripping. Alias may be empty, in
// which case the row is a ciphertext-only "growth" capture that still feeds
// the high-byte recurrence (the high bytes are readable without plaintext).
type Row struct {
	RID       uint32
	Talkgroup string
	Enc       []byte // encoded alias, trailing 2-byte CRC already removed
	Alias     string
}

// StripCRC removes the trailing 2-byte CRC from an encoded_hex field. This
// is the single most important correctness step in the whole analysis —
// feeding the CRC into the cipher fit was the documented past error.
func StripCRC(encWithCRC []byte) (enc []byte, ok bool) {
	if len(encWithCRC) < 2 {
		return nil, false
	}
	return encWithCRC[:len(encWithCRC)-2], true
}

// LoadCSV reads rid,talkgroup,encoded_hex,alias rows from path, hex-decoding
// and CRC-stripping each encoded field. A header row (non-hex first field)
// is skipped. Rows too short to hold a CRC are skipped with a warning.
func LoadCSV(path string, logger *slog.Logger) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("alias: open corpus: %w", err)
	}
	defer f.Close()
	return parseCSV(f, logger)
}

func parseCSV(r io.Reader, logger *slog.Logger) ([]Row, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	var rows []Row
	for line := 0; ; line++ {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("alias: csv line %d: %w", line, err)
		}
		if len(rec) < 3 {
			continue
		}
		encField := strings.TrimSpace(rec[2])
		raw, err := hex.DecodeString(encField)
		if err != nil {
			// Header row or malformed line: skip silently on line 0,
			// warn otherwise.
			if line > 0 && logger != nil {
				logger.Warn("skipping non-hex encoded field", "line", line, "field", encField)
			}
			continue
		}
		enc, ok := StripCRC(raw)
		if !ok {
			if logger != nil {
				logger.Warn("skipping row too short for CRC", "line", line)
			}
			continue
		}
		var rid uint32
		if v, err := strconv.ParseUint(strings.TrimSpace(rec[0]), 10, 32); err == nil {
			rid = uint32(v)
		}
		alias := ""
		if len(rec) >= 4 {
			alias = rec[3]
		}
		rows = append(rows, Row{RID: rid, Talkgroup: strings.TrimSpace(rec[1]), Enc: enc, Alias: alias})
	}
	return rows, nil
}

// HighBytes returns the observable accumulator high-byte sequence
// H[k] = LUT[enc[2k]] for a row. Readable from ciphertext alone.
func (t *Table) HighBytes(enc []byte) []uint8 {
	n := len(enc) / 2
	h := make([]uint8, n)
	for k := 0; k < n; k++ {
		h[k] = t.Fwd[enc[2*k]]
	}
	return h
}

// Context is the readable key the cell solver pins state against: the two
// preceding high bytes. For the first character (k==0) there is no H[k-1],
// so HPrev carries the length seed instead (the accumulator is length-seeded).
type Context struct {
	HPrev, HCur uint8
}

// CharObs is one decodable character observation; it needs a known alias.
type CharObs struct {
	Ctx  Context
	Char uint8
	Eo   uint8
}

// CharObservations extracts per-character observations from a row with a
// known ASCII alias. Returns nil for ciphertext-only rows.
func (t *Table) CharObservations(row Row) []CharObs {
	if row.Alias == "" {
		return nil
	}
	enc := row.Enc
	n := len(enc) / 2
	if n == 0 || n != len(row.Alias) {
		return nil // length mismatch / non-ASCII; skip rather than guess
	}
	h := t.HighBytes(enc)
	out := make([]CharObs, 0, n)
	for k := 0; k < n; k++ {
		ctx := Context{HCur: h[k]}
		if k == 0 {
			ctx.HPrev = uint8(n) // length seed
		} else {
			ctx.HPrev = h[k-1]
		}
		out = append(out, CharObs{Ctx: ctx, Char: row.Alias[k], Eo: enc[2*k+1]})
	}
	return out
}

// HighConstraints flattens the dense, plaintext-free high-byte recurrence
// H[k+1] = F(H[k-1], H[k], eo[k]) across all rows — the ~10k-key signal the
// structure mode runs over.
func (t *Table) HighConstraints(rows []Row) []propagate.Constraint {
	var cons []propagate.Constraint
	for _, row := range rows {
		enc := row.Enc
		h := t.HighBytes(enc)
		for k := 1; k+1 < len(h); k++ {
			cons = append(cons, propagate.Constraint{
				HPrev: h[k-1],
				HCur:  h[k],
				Eo:    enc[2*k+1],
				HNext: h[k+1],
			})
		}
	}
	return cons
}
