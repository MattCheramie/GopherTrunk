package hunt

import (
	"fmt"
	"io"
	"strings"
)

// Format selects an export encoding for a DiscoveredSystem.
type Format int

const (
	// FormatBundle is GopherTrunk's multi-section CSV import bundle (the exact
	// reverse of cmd/gophertrunk/import_csv.go parseCSVStream — it round-trips
	// straight back into config.yaml via `import-pdf -csv`).
	FormatBundle Format = iota
	// FormatTrunkRecorder is a trunk-recorder JSON system config stanza.
	FormatTrunkRecorder
	// FormatRR is a human-readable RadioReference.com submission package.
	FormatRR
)

// String renders the format as the value operators type on the CLI.
func (f Format) String() string {
	switch f {
	case FormatTrunkRecorder:
		return "trunk-recorder"
	case FormatRR:
		return "rr"
	default:
		return "bundle"
	}
}

// FileExtension is the conventional extension for a format's output file.
func (f Format) FileExtension() string {
	switch f {
	case FormatTrunkRecorder:
		return "json"
	case FormatRR:
		return "md"
	default:
		return "csv"
	}
}

// ParseFormat maps a -formats list value to a Format.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "bundle", "csv", "import":
		return FormatBundle, nil
	case "trunk-recorder", "trunkrecorder", "tr", "trunk_recorder":
		return FormatTrunkRecorder, nil
	case "rr", "radioreference", "submission":
		return FormatRR, nil
	default:
		return FormatBundle, fmt.Errorf("hunt: unknown export format %q (want bundle|trunk-recorder|rr)", s)
	}
}

// Write serializes sys to w in the requested format. RRHints, when non-empty,
// are rendered into the RR submission package (ignored by the other formats).
func Write(w io.Writer, sys *DiscoveredSystem, f Format, hints []DuplicateHint) error {
	sys.sortAll()
	switch f {
	case FormatBundle:
		return writeBundle(w, sys)
	case FormatTrunkRecorder:
		return writeTrunkRecorder(w, sys)
	case FormatRR:
		return writeRR(w, sys, hints)
	default:
		return fmt.Errorf("hunt: unsupported format %d", f)
	}
}

// DuplicateHint is a possible pre-existing RadioReference system match,
// surfaced by an optional read-only RR API lookup. It is rendered into the RR
// submission package so an operator doesn't submit a duplicate.
type DuplicateHint struct {
	SID        int     // RadioReference system id
	Name       string  // existing system name
	Reason     string  // why it matched (WACN+SYSID, overlapping CC, name/county)
	Confidence float64 // 0..1
}
