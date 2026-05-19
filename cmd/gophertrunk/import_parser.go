package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

// parsedSystem is the result of importing one RadioReference PDF.
type parsedSystem struct {
	Name       string            `json:"name"`
	Location   string            `json:"location"`
	County     string            `json:"county"`
	SysID      string            `json:"sysid"`
	WACN       string            `json:"wacn"`
	SystemType string            `json:"system_type"`
	Protocol   string            `json:"protocol"`
	Sites      []parsedSite      `json:"sites"`
	Talkgroups []parsedTalkgroup `json:"talkgroups"`
	SourcePath string            `json:"-"`
}

type parsedSite struct {
	RFSS        int          `json:"rfss"`
	SiteID      int          `json:"site_id"`
	SiteName    string       `json:"site_name"`
	Cty         string       `json:"cty"`
	Frequencies []parsedFreq `json:"frequencies"`
	Include     bool         `json:"include"`
}

type parsedFreq struct {
	Hz             uint32 `json:"hz"`
	ControlChannel bool   `json:"cc"`
}

type parsedTalkgroup struct {
	Dec         uint32 `json:"dec"`
	Hex         string `json:"hex"`
	Mode        string `json:"mode"`
	Encrypted   bool   `json:"encrypted"`
	AlphaTag    string `json:"alpha_tag"`
	Description string `json:"description"`
	Tag         string `json:"tag"`
	Group       string `json:"group"`
	Scan        bool   `json:"scan"`
	Priority    int    `json:"priority"`
	Lockout     bool   `json:"lockout"`
}

// parseRow is one logical line extracted from a PDF after positioned-text
// bucketing. Cells are sorted left-to-right with gap-merging applied.
// Tests construct parseRow values directly to exercise parseSystem
// without going through the PDF library.
type parseRow struct {
	Y    float64 `json:"y"`
	Page int     `json:"page"`
	Text string  `json:"text"`
}

// parsePDFFile is the production entry point: PDF on disk → parsedSystem.
func parsePDFFile(path string) (parsedSystem, error) {
	rows, err := extractPDFRows(path)
	if err != nil {
		return parsedSystem{}, err
	}
	sys, err := parseSystem(rows)
	if err != nil {
		return parsedSystem{}, err
	}
	sys.SourcePath = path
	return sys, nil
}

// extractPDFRows opens the PDF, walks every page, buckets positioned
// glyphs into rows, and returns a single flat row list with the font
// encoding normalised. RadioReference has shipped two different PDF
// encodings:
//
//   - **Shifted (older PDFs, e.g. maricopa.pdf, rwc.pdf).** The custom
//     font subset has no CMap, so the PDF library returns the raw glyph
//     bytes. Every glyph sits 27 below its real ASCII codepoint
//     ('M'=0x4D stored as 0x32='2'). `decodeShift` adds 27 to recover.
//   - **Decoded (newer PDFs, e.g. mmr.pdf).** RadioReference now ships
//     a proper font CMap, so the PDF library returns already-correct
//     UTF-8 text. Running `decodeShift` would corrupt it
//     ('S'=0x53 → 'n'=0x6E).
//
// Which encoding a given PDF uses is detected by sniffing the first
// rows for anchor strings — see `pdfTextNeedsShift`.
func extractPDFRows(path string) ([]parseRow, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("import-pdf: open %s: %w", path, err)
	}
	defer f.Close()

	var raws []rawRow
	for pn := 1; pn <= r.NumPage(); pn++ {
		p := r.Page(pn)
		if p.V.IsNull() {
			continue
		}
		texts := p.Content().Text
		// Bucket into rows by Y position with a ~2pt tolerance.
		type bucket struct {
			y     float64
			items []pdf.Text
		}
		var buckets []*bucket
		for _, t := range texts {
			added := false
			for _, b := range buckets {
				if abs64(b.y-t.Y) < 2.0 {
					b.items = append(b.items, t)
					added = true
					break
				}
			}
			if !added {
				buckets = append(buckets, &bucket{y: t.Y, items: []pdf.Text{t}})
			}
		}
		// PDF origin is bottom-left; sort top-of-page first (highest Y).
		sort.Slice(buckets, func(i, j int) bool { return buckets[i].y > buckets[j].y })
		for _, b := range buckets {
			sort.Slice(b.items, func(i, j int) bool { return b.items[i].X < b.items[j].X })
			var sb strings.Builder
			lastEnd := -999.0
			for _, t := range b.items {
				if t.X-lastEnd > 1.5 && sb.Len() > 0 {
					sb.WriteByte(' ')
				}
				sb.WriteString(t.S)
				lastEnd = t.X + t.W
			}
			text := strings.TrimSpace(collapseSpaces(sb.String()))
			if text == "" {
				continue
			}
			raws = append(raws, rawRow{Y: b.y, Page: pn, Text: text})
		}
	}

	shift := pdfTextNeedsShift(raws)
	out := make([]parseRow, 0, len(raws))
	for _, rr := range raws {
		txt := rr.Text
		if shift {
			txt = decodeShift(txt)
		}
		txt = strings.TrimSpace(collapseSpaces(fixupLigatures(txt)))
		if txt == "" {
			continue
		}
		// Filter footer ("RRDB/TRS/...") at top-of-page.
		if strings.HasPrefix(txt, "RRDB/TRS/") {
			continue
		}
		out = append(out, parseRow{Y: rr.Y, Page: rr.Page, Text: txt})
	}
	return out, nil
}

// rawRow is one raw (un-shift-decoded) positioned-text row collected
// during PDF extraction's first pass. It's an internal type because
// the shift-decision is per-PDF: we need to look at the whole document
// before deciding whether to apply `decodeShift` to each row's text.
type rawRow struct {
	Y    float64
	Page int
	Text string
}

// pdfTextNeedsShift returns true when the PDF library handed back
// shifted-encoding bytes (older RadioReference PDFs) and false when it
// handed back already-decoded text (newer PDFs with a proper font
// CMap). Probes the first 50 rows (capped at 8 KiB) for anchor markers
// that only appear in decoded text — every uppercase letter and most
// punctuation is shifted out of the printable range in the encoded
// case, so anchors like "System Name" can't accidentally pre-exist.
func pdfTextNeedsShift(rows []rawRow) bool {
	var b strings.Builder
	for i, r := range rows {
		if i >= 50 || b.Len() > 8*1024 {
			break
		}
		b.WriteString(r.Text)
		b.WriteByte(' ')
	}
	sample := b.String()
	for _, anchor := range []string{
		"System Name", "Sites and Frequencies",
		"Talkgroups", "WACN", "Last Updated",
	} {
		if strings.Contains(sample, anchor) {
			return false
		}
	}
	return true
}

// decodeShift reverses the PDF font's -27 ASCII shift. Bytes already
// in the printable range that aren't part of the shifted space (e.g.,
// the literal Ñ used as a tabular bullet) pass through unchanged.
//
// Literal space (0x20) is treated as pass-through: it does occasionally
// appear in otherwise-encoded text (the PDF text-show operator emits
// an actual space byte between adjacent text runs), and shifting it to
// ';' (0x3B) corrupts the output. An encoded space arrives as 0x05
// (a C0 control byte) which still hits the shift branch normally.
func decodeShift(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c <= 0x63 && c != 0x20 {
			d := c + 27
			if d >= 0x20 && d <= 0x7E {
				b.WriteRune(d)
				continue
			}
		}
		b.WriteRune(c)
	}
	return b.String()
}

// fixupLigatures patches common ffi/fi/fl ligature drops. The custom
// PDF font subset doesn't carry these glyphs in the shifted ASCII
// range, so they decode to whatever byte the embedded font assigned —
// usually nothing or a stray uppercase letter. We post-process by
// recognising the broken-word patterns in agency vocabulary. This is
// cosmetic only: the daemon never sees these strings as identifiers,
// just as Alpha Tag / Description / Group text.
func fixupLigatures(s string) string {
	// Common broken-ffi patterns: "ONce" → "Office", "ONcers" →
	// "Officers", "ONcial" → "Official", etc.
	for _, p := range ligatureFixes {
		s = strings.ReplaceAll(s, p[0], p[1])
	}
	return s
}

var ligatureFixes = [][2]string{
	{"ONcers", "Officers"},
	{"ONcer", "Officer"},
	{"ONcial", "Official"},
	{"ONce", "Office"},
}

func collapseSpaces(s string) string {
	return regexp.MustCompile(`[ \t]+`).ReplaceAllString(s, " ")
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// parseSystem turns rows into a parsedSystem. Stateful walk through the
// three logical sections — metadata, sites, talkgroups. Returns an
// error if no system name is found or if both sites and talkgroups are
// empty (signals an unrelated PDF format).
func parseSystem(rows []parseRow) (parsedSystem, error) {
	sys := parsedSystem{Protocol: "p25"}

	const (
		sectMeta = iota
		sectSites
		sectTGs
	)
	section := sectMeta

	var currentGroup string
	var lastSite *parsedSite

	for _, row := range rows {
		line := row.Text

		// Section transitions. RR has shipped two heading styles
		// across PDF layouts — accept either.
		switch {
		case strings.EqualFold(line, "Sites and Frequencies"),
			strings.EqualFold(line, "System Frequencies"):
			section = sectSites
			continue
		case strings.EqualFold(line, "Talkgroups"),
			strings.EqualFold(line, "System Talkgroups"):
			section = sectTGs
			continue
		case strings.HasPrefix(line, "Red (c) are"):
			continue
		}

		switch section {
		case sectMeta:
			parseMetaLine(line, &sys)
		case sectSites:
			// Column header / footer rows we skip.
			if strings.HasPrefix(line, "RFSSSite") ||
				strings.HasPrefix(line, "RFSS Site") ||
				strings.HasPrefix(line, "Site Description") {
				continue
			}
			if site, ok := parseSiteRow(line); ok {
				sys.Sites = append(sys.Sites, site)
				lastSite = &sys.Sites[len(sys.Sites)-1]
				continue
			}
			// Continuation line: bare frequencies belong to the prior site.
			if freqs := parseFreqList(line); len(freqs) > 0 && lastSite != nil {
				lastSite.Frequencies = append(lastSite.Frequencies, freqs...)
				continue
			}
		case sectTGs:
			// Skip column header rows ("DEC HEX Mode Alpha Tag ...").
			if isTGHeader(line) {
				continue
			}
			if tg, ok := parseTGRow(line); ok {
				tg.Group = currentGroup
				sys.Talkgroups = append(sys.Talkgroups, tg)
				continue
			}
			// Anything else is a group heading (Sheriffs Office,
			// Probation Department, etc.). Cosmetic noise like
			// "Grouped All Talkgroups New/Updated Talkgroups" also
			// lands here — filter those out.
			if isTGNavLine(line) {
				continue
			}
			currentGroup = strings.TrimSpace(line)
		}
	}

	if sys.Name == "" {
		// Fallback: RadioReference pages have a "<System> Menu"
		// banner at the top of page 1. If the explicit "System
		// Name:" line is missing (e.g. RR tweaked the field label)
		// we can still recover the name from the banner.
		if name := inferSystemNameFromPageTitle(rows); name != "" {
			sys.Name = name
		}
	}
	if sys.Name == "" {
		return sys, formatNoSystemNameError(rows)
	}
	if len(sys.Sites) == 0 && len(sys.Talkgroups) == 0 {
		return sys, errors.New("import-pdf: PDF contained no sites or talkgroups")
	}
	// Default site Include=true so the TUI starts with everything on.
	for i := range sys.Sites {
		sys.Sites[i].Include = true
	}
	for i := range sys.Talkgroups {
		sys.Talkgroups[i].Scan = true
	}
	// PDF type → protocol is currently always p25 (validated upstream by
	// the importer; we don't support DMR/NXDN PDF layouts yet).
	if strings.Contains(strings.ToLower(sys.SystemType), "project 25") {
		sys.Protocol = "p25"
	}
	return sys, nil
}

var (
	// siteRowRE matches the "two-parenthesised-pair" layout used by
	// older US-style RR PDFs: "1 (1) 016 (10) Oatman Mountain Maricopa
	// 769.54375c …". RFSS, alt-RFSS, SiteID, alt-SiteID, then the
	// name + county + freq tail.
	siteRowRE = regexp.MustCompile(`^(\d+)\s*\((\w+)\)\s*(\d+)\s*\((\w+)\)\s*(.+)$`)
	// siteRowDashRE matches the "dash-joined" layout used by newer
	// non-US RR PDFs (e.g. MMR): "1-001 (1-1) Melbourne 420.01250c …".
	// RFSS and SiteID are dash-joined; there's no county column.
	siteRowDashRE = regexp.MustCompile(`^(\d+)-(\d+)\s*\((\d+)-(\d+)\)\s*(.+)$`)
	freqRE        = regexp.MustCompile(`(\d{3}\.\d{2,5})([ca]?)`)
)

// parseSiteRow turns one site-row line into a parsedSite. Tries the
// two-parenthesised-pair layout first; falls back to the dash-joined
// layout. Returns ok=false when neither pattern matches (caller treats
// the line as a continuation or skips it).
func parseSiteRow(line string) (parsedSite, bool) {
	if m := siteRowRE.FindStringSubmatch(line); m != nil {
		rfss, err1 := strconv.Atoi(m[1])
		siteID, err2 := strconv.Atoi(m[3])
		if err1 != nil || err2 != nil {
			return parsedSite{}, false
		}
		tail := m[5]
		nameAndCounty, freqText := splitTailAtFreq(tail)
		siteName, county := splitNameAndCounty(nameAndCounty)
		return parsedSite{
			RFSS:        rfss,
			SiteID:      siteID,
			SiteName:    siteName,
			Cty:         county,
			Frequencies: parseFreqList(freqText),
			Include:     true,
		}, true
	}
	if m := siteRowDashRE.FindStringSubmatch(line); m != nil {
		rfss, err1 := strconv.Atoi(m[1])
		siteID, err2 := strconv.Atoi(m[2])
		if err1 != nil || err2 != nil {
			return parsedSite{}, false
		}
		tail := m[5]
		nameText, freqText := splitTailAtFreq(tail)
		return parsedSite{
			RFSS:        rfss,
			SiteID:      siteID,
			SiteName:    strings.TrimSpace(nameText),
			Frequencies: parseFreqList(freqText),
			Include:     true,
		}, true
	}
	return parsedSite{}, false
}

// splitTailAtFreq splits a site-row tail at the first frequency token,
// returning the name/county portion on the left and the frequency
// portion on the right. When the tail has no frequencies (continuation
// lines, malformed rows) the whole string is returned as the name.
func splitTailAtFreq(tail string) (name, freqs string) {
	idx := freqRE.FindStringIndex(tail)
	if idx == nil {
		return tail, ""
	}
	return strings.TrimSpace(tail[:idx[0]]), tail[idx[0]:]
}

// splitNameAndCounty pulls the County off the end of "Oatman Mountain
// Maricopa" or "Smith Peak La Paz". County is the trailing 1-2 capitalised
// alphabetic tokens; everything before is the site name. The site name
// can contain parentheses, slashes, colons, and periods (e.g. "Far
// North Mountain (Mt. Gillen)").
func splitNameAndCounty(s string) (name, county string) {
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return "", ""
	}
	// Try the 2-token county form first ("La Paz", "Santa Cruz", "St.
	// Johns", etc.). Only when the penultimate token is a recognised
	// multi-token-county prefix.
	if len(tokens) >= 2 {
		prev := tokens[len(tokens)-2]
		if isMultiTokenCountyPrefix(prev) {
			county = prev + " " + tokens[len(tokens)-1]
			name = strings.Join(tokens[:len(tokens)-2], " ")
			return name, county
		}
	}
	county = tokens[len(tokens)-1]
	name = strings.Join(tokens[:len(tokens)-1], " ")
	return name, county
}

func isMultiTokenCountyPrefix(s string) bool {
	switch s {
	case "La", "Santa", "St.", "St":
		return true
	}
	return false
}

// parseFreqList parses a run of "769.54375c 770.04375 ..." into
// parsedFreq values. Out-of-band freqs (<136 or >960 MHz, outside the
// VHF-Hi/UHF/700-800 P25 bands) are dropped; the caller's data is
// almost always valid since RadioReference enforces this upstream.
func parseFreqList(s string) []parsedFreq {
	matches := freqRE.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]parsedFreq, 0, len(matches))
	for _, m := range matches {
		mhz, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		if mhz < 136 || mhz > 960 {
			continue
		}
		hz := uint32(mhz*1_000_000 + 0.5)
		// "c" = primary control channel, "a" = secondary control
		// channel capable. Both count as control-channel-capable
		// for the daemon's purposes — `control_channels` in the
		// generated config lists every one of them.
		cc := m[2] == "c" || m[2] == "a"
		out = append(out, parsedFreq{Hz: hz, ControlChannel: cc})
	}
	return out
}

// tgRowRE peels DEC + HEX + Mode (T|TE|D|DE|A) + rest off a talkgroup row.
// DEC is 1-5 digits (P25 max 65535); HEX immediately follows DEC without
// any separator and is matched against `fmt.Sprintf("%x", dec)` in
// parseTGRow for cross-validation.
var tgRowRE = regexp.MustCompile(`^(\d{1,5})\s*([0-9a-fA-F]{1,4})\s*(TE|DE|T|D|A)\s+(.+)$`)

func parseTGRow(line string) (parsedTalkgroup, bool) {
	m := tgRowRE.FindStringSubmatch(line)
	if m == nil {
		return parsedTalkgroup{}, false
	}
	dec, err := strconv.ParseUint(m[1], 10, 32)
	if err != nil || dec > 65535 {
		return parsedTalkgroup{}, false
	}
	// HEX may be zero-padded in newer PDFs (e.g. "065" for dec=101).
	// Parse and compare numerically so leading zeros don't matter.
	gotHex, hexErr := strconv.ParseUint(m[2], 16, 32)
	if hexErr != nil || gotHex != dec {
		return parsedTalkgroup{}, false
	}
	mode := m[3]
	encrypted := strings.HasSuffix(mode, "E")
	tail := strings.TrimSpace(m[4])

	// Tail is "<AlphaTag> <Description> <Tag>" — the last token (or
	// last 2-3 tokens) form the Tag column. RadioReference Tag values
	// are short canonical strings: "Interop", "Law Dispatch", "Law
	// Tac", "Corrections", "Public Works", "Fire-Tac", etc. We split
	// at the rightmost run of recognised Tag tokens; what remains is
	// "<AlphaTag> <Description>". Alpha Tag has no internal spaces in
	// 80% of records — when it does (e.g. "SP DTL 1NE"), we anchor on
	// the upper-case run.
	alphaTag, desc, tag := splitTalkgroupTail(tail)

	return parsedTalkgroup{
		Dec:         uint32(dec),
		Hex:         strings.ToLower(m[2]),
		Mode:        normaliseMode(mode),
		Encrypted:   encrypted,
		AlphaTag:    alphaTag,
		Description: desc,
		Tag:         tag,
		Scan:        true,
	}, true
}

func normaliseMode(m string) string {
	switch m {
	case "T", "TE", "D", "DE":
		return "D"
	case "A":
		return "A"
	}
	return ""
}

// knownTags lists the canonical RadioReference Tag column values. The
// list is finite by design — RR uses a controlled vocabulary. The
// splitter walks the tail tokens right-to-left, growing the Tag string
// as long as the running suffix matches one of these values.
var knownTags = []string{
	"Interop", "Law Dispatch", "Law Tac", "Law Talk",
	"Fire Dispatch", "Fire-Tac", "Fire-Tac/Talk", "Fire Talk",
	"EMS Dispatch", "EMS-Tac", "EMS Talk",
	"Corrections", "Public Works", "Public Health",
	"Multi-Tac", "Multi-Dispatch", "Multi-Talk",
	"Schools", "Transportation", "Utilities",
	"Federal", "Military", "Business", "Hospital",
	"Emergency Ops", "Emergency", "Hospitals",
	"Other", "Aviation", "Data",
}

func splitTalkgroupTail(tail string) (alpha, desc, tag string) {
	// Find the longest known-tag suffix.
	lower := strings.ToLower(tail)
	for _, kt := range knownTags {
		kl := strings.ToLower(kt)
		if strings.HasSuffix(lower, " "+kl) || lower == kl {
			tag = kt
			head := strings.TrimSpace(tail[:len(tail)-len(kt)])
			alpha, desc = splitAlphaAndDescription(head)
			return alpha, desc, tag
		}
	}
	// No known tag suffix — fall back to "last token = tag".
	tokens := strings.Fields(tail)
	if len(tokens) >= 2 {
		tag = tokens[len(tokens)-1]
		head := strings.Join(tokens[:len(tokens)-1], " ")
		alpha, desc = splitAlphaAndDescription(head)
		return alpha, desc, tag
	}
	return tail, "", ""
}

// splitAlphaAndDescription tries to split "MCSO Ops Operations with
// outside agencies" into alpha="MCSO Ops", desc="Operations with outside
// agencies". The alpha tag in RadioReference is typically ALL-CAPS or
// short MixedCase; the description tends to start with a separate
// CapitalisedWord followed by lowercase prose. Heuristic: scan
// left-to-right; alpha ends at the first transition from
// short-uppercase-or-digit-heavy tokens to a lowercase-heavy token.
func splitAlphaAndDescription(s string) (alpha, desc string) {
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return "", ""
	}
	if len(tokens) == 1 {
		return tokens[0], ""
	}
	// Walk forward: while the running prefix "looks like an alpha tag"
	// (all upper-case letters, digits, or short MixedCase), keep
	// extending it. Stop when we hit a token that's lower-case-leading
	// or longer than 5 chars MixedCase.
	end := 1
	for end < len(tokens) {
		tok := tokens[end]
		if looksLikeDescriptionToken(tok) {
			break
		}
		end++
	}
	// Reasonable cap: alpha is at most 4 tokens.
	if end > 4 {
		end = 4
	}
	alpha = strings.Join(tokens[:end], " ")
	if end < len(tokens) {
		desc = strings.Join(tokens[end:], " ")
	}
	return alpha, desc
}

func looksLikeDescriptionToken(t string) bool {
	if t == "" {
		return false
	}
	// Tokens that are clearly description-y: start with a lowercase
	// letter, contain a hyphen/slash mid-word, or are 6+ letters with
	// only the first letter capitalised.
	r := []rune(t)
	if r[0] >= 'a' && r[0] <= 'z' {
		return true
	}
	if len(r) >= 6 {
		// Capitalised-then-lowercase = "Operations", "Information", etc.
		first := r[0]
		if first >= 'A' && first <= 'Z' {
			rest := r[1:]
			allLower := true
			for _, c := range rest {
				if c < 'a' || c > 'z' {
					allLower = false
					break
				}
			}
			if allLower {
				return true
			}
		}
	}
	return false
}

func isTGHeader(line string) bool {
	l := strings.ToLower(line)
	if !strings.HasPrefix(l, "dec") || !strings.Contains(l, "hex") || !strings.Contains(l, "mode") {
		return false
	}
	// Maricopa/RWC use "alpha tag"; MMR uses "display" for the same column.
	return strings.Contains(l, "alpha tag") || strings.Contains(l, "display")
}

func isTGNavLine(line string) bool {
	l := strings.TrimSpace(line)
	return l == "Grouped All Talkgroups New/Updated Talkgroups" ||
		l == "All Talkgroups" || l == "New/Updated Talkgroups" ||
		l == "Grouped"
}

// parseMetaLine extracts "System Name: …", "Location: …", "System ID: …"
// metadata. Matching is case-insensitive and tolerates extra whitespace
// around the label (e.g. "SYSTEM NAME : Foo", "  System Name:   Foo  ")
// — RadioReference occasionally tweaks the PDF capitalisation and we'd
// rather extract correctly than re-issue a parser patch every time.
// Lines that don't match any known key are ignored.
func parseMetaLine(line string, sys *parsedSystem) {
	key, val, ok := splitMetaLine(line)
	if !ok {
		return
	}
	switch key {
	case "system name":
		sys.Name = val
	case "location":
		sys.Location = val
	case "county":
		sys.County = val
	case "system type":
		sys.SystemType = val
	case "system id":
		// "Sysid: 49A WACN: BEE00" appears as the value when the
		// label is "System ID:" — split on the inner WACN: token.
		if i := indexFoldASCII(val, "WACN:"); i >= 0 {
			left := strings.TrimSpace(val[:i])
			left = strings.TrimPrefix(left, "Sysid:")
			left = strings.TrimPrefix(left, "sysid:")
			left = strings.TrimPrefix(left, "SYSID:")
			sys.SysID = strings.TrimSpace(left)
			sys.WACN = strings.TrimSpace(val[i+len("WACN:"):])
		} else {
			sys.SysID = val
		}
	}
}

// splitMetaLine splits a "Key: Value" line at the first colon, returning
// the normalised key (lower-case, whitespace-collapsed) and the trimmed
// value. ok=false when the line has no colon or an empty key.
func splitMetaLine(line string) (key, val string, ok bool) {
	i := strings.IndexByte(line, ':')
	if i <= 0 {
		return "", "", false
	}
	rawKey := strings.TrimSpace(line[:i])
	if rawKey == "" {
		return "", "", false
	}
	key = strings.ToLower(collapseSpaces(rawKey))
	val = strings.TrimSpace(line[i+1:])
	return key, val, true
}

// indexFoldASCII is a case-insensitive strings.Index for ASCII needles.
func indexFoldASCII(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	lh := strings.ToLower(haystack)
	ln := strings.ToLower(needle)
	return strings.Index(lh, ln)
}

// pageTitleRE matches RadioReference's top-of-page nav banner — the
// system page title followed by " Menu" (the dropdown trigger). When the
// PDF lacks an explicit "System Name:" line we fall back to this row.
// "Menu" is kept case-sensitive on purpose: the banner uses it
// verbatim, and a relaxed match would catch unrelated rows.
var pageTitleRE = regexp.MustCompile(`^(.+?)\s+Menu\s*$`)

// inferSystemNameFromPageTitle scans page 1 for the "<Name> Menu" banner
// row and returns the captured name. Empty string when no candidate is
// found. Used only when parseMetaLine never set sys.Name.
func inferSystemNameFromPageTitle(rows []parseRow) string {
	for _, r := range rows {
		if r.Page != 1 {
			continue
		}
		txt := strings.TrimSpace(r.Text)
		if txt == "" {
			continue
		}
		m := pageTitleRE.FindStringSubmatch(txt)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		return name
	}
	return ""
}

// formatNoSystemNameError builds the diagnostic error returned when the
// parser can't find a system name in the PDF. The error string carries
// the first 30 (or 4 KiB of) extracted rows so reporters and
// maintainers see the actual PDF contents without having to re-run with
// -extract-only. Each row's text is truncated at 120 runes.
func formatNoSystemNameError(rows []parseRow) error {
	const (
		maxRows  = 30
		maxBytes = 4 * 1024
		maxText  = 120
	)
	var b strings.Builder
	b.WriteString("import-pdf: no System Name found — wrong PDF?\n")
	b.WriteString("hint: re-run with -extract-only to share a JSON fixture (see docs/import.md)\n")
	b.WriteString("first extracted rows (page/Y/text, text truncated at 120 runes):")
	count := 0
	for _, r := range rows {
		if count >= maxRows {
			break
		}
		line := fmt.Sprintf("\n  p%d y=%.2f %q", r.Page, r.Y, truncateRunes(r.Text, maxText))
		if b.Len()+len(line) > maxBytes {
			break
		}
		b.WriteString(line)
		count++
	}
	if count == 0 {
		b.WriteString("\n  (no rows extracted — the PDF may be empty or use an unsupported encoding)")
	}
	return errors.New(b.String())
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// loadParseRowsJSON loads a serialised []parseRow from disk (used by
// tests so they don't depend on the PDF library at test time).
func loadParseRowsJSON(r io.Reader) ([]parseRow, error) {
	var out []parseRow
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// dumpParseRowsJSON serialises rows to JSON. Used by the generator
// command (-tags rr_import_extract) to refresh test fixtures.
func dumpParseRowsJSON(w io.Writer, rows []parseRow) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// SystemConfigYAML is the on-disk YAML shape of one trunking.systems[]
// entry. Mirrors internal/config.SystemConfig but we don't import that
// package here to keep the importer's dependency surface light.
type SystemConfigYAML struct {
	Name            string   `yaml:"name"`
	Protocol        string   `yaml:"protocol"`
	ControlChannels []uint32 `yaml:"control_channels"`
	TalkgroupFile   string   `yaml:"talkgroup_file"`
}

// readFile is a tiny helper centralised so callers can wrap errors
// uniformly. Returns the file contents and a wrapped error.
func readFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("import-pdf: read %s: %w", path, err)
	}
	return b, nil
}
