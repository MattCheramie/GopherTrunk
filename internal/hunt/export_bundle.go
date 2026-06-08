package hunt

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// writeBundle emits the multi-section CSV import bundle that
// cmd/gophertrunk/import_csv.go parseCSVStream reads. The section markers and
// column names match parseMetadataSection / parseSitesSection /
// parseTalkgroupsSection exactly so a discovery round-trips back into
// config.yaml without loss. Rows are written through encoding/csv, whose
// dialect matches the importer's splitCSVLine (comma-separated, double-quote
// escaping), so site names and locations with commas survive.
func writeBundle(w io.Writer, sys *DiscoveredSystem) error {
	bw := &bundleWriter{w: w}

	// --- metadata ---
	bw.section("metadata")
	bw.row("key", "value")
	bw.row("name", sys.DisplayName())
	if sys.Protocol != "" {
		bw.row("protocol", sys.Protocol)
	}
	if sys.SystemID != 0 {
		bw.row("sysid", fmt.Sprintf("%X", sys.SystemID))
	}
	if sys.WACN != 0 {
		bw.row("wacn", fmt.Sprintf("%X", sys.WACN))
	}
	if sys.Location != "" {
		bw.row("location", sys.Location)
	}
	if sys.County != "" {
		bw.row("county", sys.County)
	}

	// --- sites ---
	if len(sys.Sites) > 0 {
		bw.blank()
		bw.section("sites")
		bw.row("rfss", "site_id", "site_name", "county", "frequencies")
		for _, st := range sys.Sites {
			name := st.SiteName
			if name == "" {
				name = fmt.Sprintf("Site %d-%d", st.RFSS, st.SiteID)
			}
			bw.row(
				strconv.Itoa(int(st.RFSS)),
				strconv.Itoa(int(st.SiteID)),
				name,
				st.County,
				freqList(st),
			)
		}
	}

	// --- talkgroups ---
	if len(sys.Talkgroups) > 0 {
		bw.blank()
		bw.section("talkgroups")
		bw.row("decimal", "hex", "mode", "alpha_tag", "description", "tag", "group", "priority", "lockout", "scan")
		for _, tg := range sys.Talkgroups {
			bw.row(
				strconv.FormatUint(uint64(tg.Dec), 10),
				tg.Hex,
				"D", // blind discovery: default digital; operator/RR refine
				"",  // alpha_tag unknown
				"",  // description unknown
				"",  // tag unknown
				"",  // group unknown
				"",  // priority unset
				"",  // lockout off
				"Y", // scan on
			)
		}
	}

	return bw.flush()
}

// freqList renders a site's channels as the `|`-separated MHz list the sites
// section expects, with control channels carrying the trailing `c` flag.
func freqList(st DiscoveredSite) string {
	parts := make([]string, 0, len(st.ControlChannels)+len(st.Secondary))
	for _, ch := range st.ControlChannels {
		s := formatMHz(ch.FrequencyHz)
		if ch.IsControl {
			s += "c"
		}
		parts = append(parts, s)
	}
	for _, hz := range st.Secondary {
		parts = append(parts, formatMHz(hz)+"c")
	}
	return strings.Join(parts, "|")
}

// formatMHz renders a frequency in Hz as a MHz string with trailing zeros
// trimmed (e.g. 851012500 → "851.0125"), the spelling the importer parses.
func formatMHz(hz uint32) string {
	s := strconv.FormatFloat(float64(hz)/1_000_000, 'f', 6, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// bundleWriter buffers a single error and wraps an encoding/csv.Writer so the
// per-row error handling stays out of writeBundle.
type bundleWriter struct {
	w   io.Writer
	cw  *csv.Writer
	err error
}

func (b *bundleWriter) section(name string) {
	if b.err != nil {
		return
	}
	if b.cw != nil {
		b.cw.Flush()
		b.err = b.cw.Error()
		b.cw = nil
	}
	_, b.err = fmt.Fprintf(b.w, "# Section: %s\n", name)
	if b.err == nil {
		b.cw = csv.NewWriter(b.w)
	}
}

func (b *bundleWriter) row(fields ...string) {
	if b.err != nil || b.cw == nil {
		return
	}
	b.err = b.cw.Write(fields)
}

func (b *bundleWriter) blank() {
	if b.err != nil {
		return
	}
	if b.cw != nil {
		b.cw.Flush()
		b.err = b.cw.Error()
		b.cw = nil
	}
	if b.err == nil {
		_, b.err = io.WriteString(b.w, "\n")
	}
}

func (b *bundleWriter) flush() error {
	if b.err != nil {
		return b.err
	}
	if b.cw != nil {
		b.cw.Flush()
		b.err = b.cw.Error()
	}
	return b.err
}
