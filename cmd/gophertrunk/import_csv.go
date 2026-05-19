package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// csvImportOpts carries operator-supplied metadata for CSV imports.
// The fields are only consulted by the native-RadioReference-CSV path
// (which has no metadata section); the multi-section bundle format
// ignores them and continues to require metadata.name in the file.
type csvImportOpts struct {
	Name  string
	SysID string
}

// parseCSVFile loads a CSV from disk and turns it into a parsedSystem.
// Two formats are supported and detected by content sniffing:
//
//   - **Multi-section bundle** — `# Section: …` markers delimit
//     metadata / sites / talkgroups blocks. See docs/import.md.
//   - **Native RadioReference CSV** — the flat talkgroup table that
//     RadioReference's `/db/sid/<sid>/download` page serves. Carries
//     no metadata; opts.Name and opts.SysID supply it (with the
//     filename stem as a fallback for the name).
func parseCSVFile(path string, opts csvImportOpts) (parsedSystem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return parsedSystem{}, fmt.Errorf("import-pdf: open %s: %w", path, err)
	}
	var sys parsedSystem
	if looksLikeRRNativeCSV(data) {
		fillOpts := opts
		if fillOpts.Name == "" {
			fillOpts.Name = filenameStem(path)
		}
		sys, err = parseRRNativeCSVStream(bytes.NewReader(data), fillOpts)
	} else {
		sys, err = parseCSVStream(bytes.NewReader(data))
	}
	if err != nil {
		return parsedSystem{}, fmt.Errorf("import-pdf: %s: %w", path, err)
	}
	sys.SourcePath = path
	return sys, nil
}

// filenameStem returns the base filename with its extension stripped —
// used as the default system name for native RR CSV imports when the
// operator didn't pass -name.
func filenameStem(path string) string {
	base := filepath.Base(path)
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	return base
}

// csvSection collects raw lines (as []string slices, already CSV-split)
// for one section so the per-section parsers can run after the splitter
// has visited the whole file.
type csvSection struct {
	name    string
	startLn int
	rows    [][]string
}

// parseCSVStream is the testable entry point — reads a multi-section
// CSV from any io.Reader.
func parseCSVStream(r io.Reader) (parsedSystem, error) {
	sections, err := splitCSVSections(r)
	if err != nil {
		return parsedSystem{}, err
	}

	sys := parsedSystem{Protocol: "p25"}
	saw := map[string]bool{}
	for _, sec := range sections {
		if saw[sec.name] {
			return sys, fmt.Errorf("duplicate section %q at line %d", sec.name, sec.startLn)
		}
		saw[sec.name] = true
		switch sec.name {
		case "metadata":
			if err := parseMetadataSection(sec, &sys); err != nil {
				return sys, err
			}
		case "sites":
			sites, err := parseSitesSection(sec)
			if err != nil {
				return sys, err
			}
			sys.Sites = sites
		case "talkgroups":
			tgs, err := parseTalkgroupsSection(sec)
			if err != nil {
				return sys, err
			}
			sys.Talkgroups = tgs
		default:
			return sys, fmt.Errorf("unknown section %q at line %d (expected: metadata, sites, talkgroups)", sec.name, sec.startLn)
		}
	}

	if sys.Name == "" {
		return sys, errors.New("CSV missing metadata.name — every system needs a name")
	}
	if len(sys.Sites) == 0 && len(sys.Talkgroups) == 0 {
		return sys, errors.New("CSV has neither sites nor talkgroups — nothing to import")
	}
	if sys.Protocol == "" {
		sys.Protocol = "p25"
	}
	return sys, nil
}

// splitCSVSections does a single pass over the input, dropping comment
// lines (`# …`) but recognising `# Section: <name>` as a divider. The
// CSV split inside each section uses encoding/csv-compatible semantics
// (commas, double-quote escaping). We hand-roll the splitter instead
// of running encoding/csv per-line because section markers and
// in-section comments need preserving / dropping inline.
func splitCSVSections(r io.Reader) ([]csvSection, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []csvSection
	var current *csvSection
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		// Section marker?
		if name, ok := matchSectionMarker(trimmed); ok {
			out = append(out, csvSection{name: name, startLn: lineNum})
			current = &out[len(out)-1]
			continue
		}
		// Other comments / blank lines.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("data at line %d before any `# Section: …` marker", lineNum)
		}
		fields, err := splitCSVLine(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		current.rows = append(current.rows, fields)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("csv scan: %w", err)
	}
	return out, nil
}

// matchSectionMarker recognises `# Section: <name>` (case-insensitive,
// flexible whitespace). The word "Section" must be followed by a colon
// or whitespace — otherwise "Sections" or "Sectioned" inside a comment
// would be misread as a marker.
func matchSectionMarker(line string) (string, bool) {
	if !strings.HasPrefix(line, "#") {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "#"))
	lower := strings.ToLower(rest)
	if !strings.HasPrefix(lower, "section") {
		return "", false
	}
	rest = rest[len("section"):]
	// Require a word boundary after "section".
	if rest == "" {
		return "", false
	}
	switch rest[0] {
	case ':', ' ', '\t':
	default:
		return "", false
	}
	rest = strings.TrimSpace(rest)
	rest = strings.TrimPrefix(rest, ":")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	return strings.ToLower(rest), true
}

// splitCSVLine parses one CSV record (a single line — no embedded
// newlines). Supports double-quoted fields with escaped quotes (`""`),
// matching the encoding/csv default dialect.
func splitCSVLine(line string) ([]string, error) {
	// Trim trailing CR (Windows line endings) since bufio.Scanner
	// keeps them when configured with the default split func.
	line = strings.TrimRight(line, "\r")
	var fields []string
	var cur strings.Builder
	inQuote := false
	i := 0
	for i < len(line) {
		c := line[i]
		switch {
		case inQuote:
			if c == '"' {
				if i+1 < len(line) && line[i+1] == '"' {
					cur.WriteByte('"')
					i += 2
					continue
				}
				inQuote = false
				i++
				continue
			}
			cur.WriteByte(c)
			i++
		case c == '"':
			if cur.Len() > 0 {
				return nil, errors.New(`unexpected " in middle of unquoted field`)
			}
			inQuote = true
			i++
		case c == ',':
			fields = append(fields, cur.String())
			cur.Reset()
			i++
		default:
			cur.WriteByte(c)
			i++
		}
	}
	if inQuote {
		return nil, errors.New("unterminated quoted field")
	}
	fields = append(fields, cur.String())
	// Trim whitespace from every field.
	for i, f := range fields {
		fields[i] = strings.TrimSpace(f)
	}
	return fields, nil
}

// columnMap builds a case-insensitive lookup from the section header
// row. Aliases let the user write "Alpha Tag", "alpha_tag", or
// "alphatag" and get the same column.
func columnMap(header []string, aliases map[string][]string) (map[string]int, error) {
	col := map[string]int{}
	for i, h := range header {
		norm := strings.ToLower(strings.TrimSpace(h))
		col[norm] = i
	}
	for canon, alts := range aliases {
		canon = strings.ToLower(canon)
		if _, ok := col[canon]; ok {
			continue
		}
		for _, a := range alts {
			a = strings.ToLower(a)
			if idx, ok := col[a]; ok {
				col[canon] = idx
				break
			}
		}
	}
	return col, nil
}

func cell(row []string, col map[string]int, key string) string {
	idx, ok := col[strings.ToLower(key)]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// parseMetadataSection expects a two-column key/value table. The first
// row is the header (e.g. `key,value`) — case-insensitive.
func parseMetadataSection(sec csvSection, sys *parsedSystem) error {
	if len(sec.rows) == 0 {
		return errors.New("metadata section is empty")
	}
	header := sec.rows[0]
	col, _ := columnMap(header, map[string][]string{
		"key":   {"field", "name"},
		"value": {"val", "v"},
	})
	keyIdx, valIdx := col["key"], col["value"]
	if _, ok := col["key"]; !ok {
		return errors.New("metadata header missing `key` column")
	}
	if _, ok := col["value"]; !ok {
		return errors.New("metadata header missing `value` column")
	}
	for i, row := range sec.rows[1:] {
		if keyIdx >= len(row) || valIdx >= len(row) {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(row[keyIdx]))
		v := strings.TrimSpace(row[valIdx])
		switch k {
		case "name":
			sys.Name = v
		case "protocol":
			sys.Protocol = strings.ToLower(v)
		case "sysid":
			sys.SysID = v
		case "wacn":
			sys.WACN = v
		case "location":
			sys.Location = v
		case "county":
			sys.County = v
		case "system_type", "system type":
			sys.SystemType = v
		default:
			return fmt.Errorf("metadata row %d: unknown key %q (allowed: name, protocol, sysid, wacn, location, county, system_type)", sec.startLn+i+1, k)
		}
	}
	return nil
}

// parseSitesSection expects rfss/site_id/site_name/county/frequencies.
func parseSitesSection(sec csvSection) ([]parsedSite, error) {
	if len(sec.rows) == 0 {
		return nil, errors.New("sites section is empty")
	}
	header := sec.rows[0]
	col, _ := columnMap(header, map[string][]string{
		"rfss":        {"rfss_id", "rfssid"},
		"site_id":     {"siteid", "site"},
		"site_name":   {"sitename", "name"},
		"county":      {"county_name", "cty"},
		"frequencies": {"freq", "freqs", "freq_list"},
	})
	for _, req := range []string{"site_name", "frequencies"} {
		if _, ok := col[req]; !ok {
			return nil, fmt.Errorf("sites header missing required column %q (header: %s)", req, strings.Join(header, ","))
		}
	}

	var out []parsedSite
	for i, row := range sec.rows[1:] {
		lineNum := sec.startLn + i + 1
		name := cell(row, col, "site_name")
		if name == "" {
			continue
		}
		rfss, _ := strconv.Atoi(cell(row, col, "rfss"))
		siteID, _ := strconv.Atoi(cell(row, col, "site_id"))
		freqText := cell(row, col, "frequencies")
		freqs, err := parseCSVFrequencies(freqText)
		if err != nil {
			return nil, fmt.Errorf("sites row %d (%q): %w", lineNum, name, err)
		}
		if len(freqs) == 0 {
			return nil, fmt.Errorf("sites row %d (%q): no valid frequencies", lineNum, name)
		}
		out = append(out, parsedSite{
			RFSS:        rfss,
			SiteID:      siteID,
			SiteName:    name,
			Cty:         cell(row, col, "county"),
			Frequencies: freqs,
			Include:     true,
		})
	}
	return out, nil
}

// parseCSVFrequencies accepts a `|`-separated frequency list (each
// MHz[c]) — the canonical sites CSV shape. Also accepts space-,
// semicolon-, or comma-separated lists for tolerance with hand-edited
// spreadsheets.
func parseCSVFrequencies(s string) ([]parsedFreq, error) {
	if s == "" {
		return nil, nil
	}
	repl := strings.NewReplacer(";", "|", " ", "|", "\t", "|")
	parts := strings.Split(repl.Replace(s), "|")
	var out []parsedFreq
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cc := false
		if strings.HasSuffix(strings.ToLower(p), "c") {
			cc = true
			p = p[:len(p)-1]
		}
		mhz, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid frequency %q", p)
		}
		if mhz < 25 || mhz > 1300 {
			return nil, fmt.Errorf("frequency %.5f MHz outside reasonable trunking range (25-1300 MHz)", mhz)
		}
		out = append(out, parsedFreq{Hz: uint32(mhz*1_000_000 + 0.5), ControlChannel: cc})
	}
	return out, nil
}

// talkgroupAliases is the canonical column-alias map shared by both the
// bundle's `talkgroups` section and the native-RR-CSV parser. Keeping
// it in one place means a single header change covers both formats.
var talkgroupAliases = map[string][]string{
	"decimal":     {"dec"},
	"hex":         {"hexadecimal"},
	"mode":        {"type"},
	"alpha_tag":   {"alpha tag", "alphatag", "tag_short"},
	"description": {"desc", "name"},
	"tag":         {"category"},
	"group":       {"group_name", "alpha_group"},
	"priority":    {"prio", "pri"},
	"lockout":     {"locked", "block"},
	"scan":        {"active", "monitor"},
}

// parseTalkgroupsSection expects decimal + alpha_tag at minimum.
// Aliases match Trunk Recorder's CSV column names exactly so users can
// reuse files they already have.
func parseTalkgroupsSection(sec csvSection) ([]parsedTalkgroup, error) {
	if len(sec.rows) == 0 {
		return nil, errors.New("talkgroups section is empty")
	}
	header := sec.rows[0]
	col, _ := columnMap(header, talkgroupAliases)
	if _, ok := col["decimal"]; !ok {
		return nil, fmt.Errorf("talkgroups header missing required column `decimal` (or `dec`) (header: %s)", strings.Join(header, ","))
	}

	var out []parsedTalkgroup
	for i, row := range sec.rows[1:] {
		lineNum := sec.startLn + i + 1
		tg, ok, err := tgFromCSVRow(row, col, lineNum)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out = append(out, tg)
	}
	return out, nil
}

// tgFromCSVRow turns one CSV row into a parsedTalkgroup using the
// canonical column map. ok=false (with nil error) for rows that have
// no decimal value — typically blank rows or footer comments left in
// place by spreadsheet exports.
func tgFromCSVRow(row []string, col map[string]int, lineNum int) (parsedTalkgroup, bool, error) {
	decStr := cell(row, col, "decimal")
	if decStr == "" {
		return parsedTalkgroup{}, false, nil
	}
	dec64, err := strconv.ParseUint(decStr, 10, 32)
	if err != nil {
		return parsedTalkgroup{}, false, fmt.Errorf("talkgroups row %d: invalid decimal %q: %w", lineNum, decStr, err)
	}
	hex := strings.ToLower(strings.TrimSpace(cell(row, col, "hex")))
	if hex == "" {
		hex = fmt.Sprintf("%x", dec64)
	}
	mode := strings.ToUpper(cell(row, col, "mode"))
	switch mode {
	case "", "D", "A", "M":
		// ok
	case "DE":
		mode = "D"
	case "TE", "T":
		mode = "D"
	default:
		return parsedTalkgroup{}, false, fmt.Errorf("talkgroups row %d: invalid mode %q (use D, A, or M)", lineNum, mode)
	}
	if mode == "" {
		mode = "D"
	}
	pri := 0
	if p := cell(row, col, "priority"); p != "" {
		pri, err = strconv.Atoi(p)
		if err != nil {
			return parsedTalkgroup{}, false, fmt.Errorf("talkgroups row %d: invalid priority %q", lineNum, p)
		}
	}
	return parsedTalkgroup{
		Dec:         uint32(dec64),
		Hex:         hex,
		Mode:        mode,
		AlphaTag:    cell(row, col, "alpha_tag"),
		Description: cell(row, col, "description"),
		Tag:         cell(row, col, "tag"),
		Group:       cell(row, col, "group"),
		Priority:    pri,
		Lockout:     parseBool(cell(row, col, "lockout"), false),
		Scan:        parseBool(cell(row, col, "scan"), true),
	}, true, nil
}

// looksLikeRRNativeCSV returns true when the input looks like a
// RadioReference-native talkgroup CSV: no `# Section: …` markers
// anywhere AND a header row whose normalised column names include at
// least three of the canonical talkgroup fields. The check stops after
// scanning the first 8 KiB so it stays cheap on large files.
func looksLikeRRNativeCSV(data []byte) bool {
	head := data
	const peek = 8 * 1024
	if len(head) > peek {
		head = head[:peek]
	}
	scanner := bufio.NewScanner(bytes.NewReader(head))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var headerLine string
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Any section marker disqualifies the file: it's a bundle.
		if _, ok := matchSectionMarker(trimmed); ok {
			return false
		}
		// Other comments are skipped; the bundle format uses them
		// freely and RR's native export never does, but a stray "#"
		// at the top shouldn't trip the sniffer either way.
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if headerLine == "" {
			headerLine = line
		}
	}
	if headerLine == "" {
		return false
	}
	fields, err := splitCSVLine(headerLine)
	if err != nil || len(fields) < 5 {
		return false
	}
	known := map[string]bool{
		"dec": true, "decimal": true, "hex": true, "mode": true,
		"alpha tag": true, "alpha_tag": true, "alphatag": true,
		"description": true, "tag": true, "category": true, "group": true,
	}
	hits := 0
	for _, f := range fields {
		norm := strings.ToLower(strings.TrimSpace(f))
		if known[norm] {
			hits++
		}
	}
	return hits >= 3
}

// parseRRNativeCSVStream parses RadioReference's flat talkgroup CSV
// (the file served from `/db/sid/<sid>/download`). The format has no
// metadata or sites — the operator supplies the system name and ID
// out-of-band via opts. Empty Name yields an error: the caller is
// expected to default to the filename stem before calling.
func parseRRNativeCSVStream(r io.Reader, opts csvImportOpts) (parsedSystem, error) {
	sys := parsedSystem{Protocol: "p25", Name: opts.Name, SysID: opts.SysID}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var header []string
	var col map[string]int
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields, err := splitCSVLine(raw)
		if err != nil {
			return sys, fmt.Errorf("line %d: %w", lineNum, err)
		}
		if header == nil {
			header = fields
			col, _ = columnMap(header, talkgroupAliases)
			if _, ok := col["decimal"]; !ok {
				return sys, fmt.Errorf("native RR CSV header missing `decimal`/`dec` column (header: %s)", strings.Join(header, ","))
			}
			continue
		}
		tg, ok, err := tgFromCSVRow(fields, col, lineNum)
		if err != nil {
			return sys, err
		}
		if !ok {
			continue
		}
		sys.Talkgroups = append(sys.Talkgroups, tg)
	}
	if err := scanner.Err(); err != nil {
		return sys, fmt.Errorf("csv scan: %w", err)
	}
	if sys.Name == "" {
		return sys, errors.New("native RR CSV: missing system name (pass -name or use a filename stem)")
	}
	if len(sys.Talkgroups) == 0 {
		return sys, errors.New("native RR CSV: no talkgroups found")
	}
	return sys, nil
}

// parseBool turns the Trunk-Recorder-style yes/no/Y/N/1/0/true/false
// cell into a boolean. Empty falls back to the supplied default.
func parseBool(s string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return def
	case "y", "yes", "true", "1", "on":
		return true
	case "n", "no", "false", "0", "off":
		return false
	}
	return def
}
