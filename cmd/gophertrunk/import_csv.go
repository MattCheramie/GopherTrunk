package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// parseCSVFile loads a multi-section CSV bundle from disk and turns it
// into a parsedSystem. See docs/import.md for the format. Sections
// supported: metadata, sites, talkgroups. Order doesn't matter; any
// section may be omitted (but metadata.name is required and at least
// one of sites/talkgroups must be present).
func parseCSVFile(path string) (parsedSystem, error) {
	f, err := os.Open(path)
	if err != nil {
		return parsedSystem{}, fmt.Errorf("import-pdf: open %s: %w", path, err)
	}
	defer f.Close()
	sys, err := parseCSVStream(f)
	if err != nil {
		return parsedSystem{}, fmt.Errorf("import-pdf: %s: %w", path, err)
	}
	sys.SourcePath = path
	return sys, nil
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

// parseTalkgroupsSection expects decimal + alpha_tag at minimum.
// Aliases match Trunk Recorder's CSV column names exactly so users can
// reuse files they already have.
func parseTalkgroupsSection(sec csvSection) ([]parsedTalkgroup, error) {
	if len(sec.rows) == 0 {
		return nil, errors.New("talkgroups section is empty")
	}
	header := sec.rows[0]
	col, _ := columnMap(header, map[string][]string{
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
	})
	if _, ok := col["decimal"]; !ok {
		return nil, fmt.Errorf("talkgroups header missing required column `decimal` (or `dec`) (header: %s)", strings.Join(header, ","))
	}

	var out []parsedTalkgroup
	for i, row := range sec.rows[1:] {
		lineNum := sec.startLn + i + 1
		decStr := cell(row, col, "decimal")
		if decStr == "" {
			continue
		}
		dec64, err := strconv.ParseUint(decStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("talkgroups row %d: invalid decimal %q: %w", lineNum, decStr, err)
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
			return nil, fmt.Errorf("talkgroups row %d: invalid mode %q (use D, A, or M)", lineNum, mode)
		}
		if mode == "" {
			mode = "D"
		}
		pri := 0
		if p := cell(row, col, "priority"); p != "" {
			pri, err = strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("talkgroups row %d: invalid priority %q", lineNum, p)
			}
		}
		out = append(out, parsedTalkgroup{
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
		})
	}
	return out, nil
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
