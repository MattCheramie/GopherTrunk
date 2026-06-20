package trunking

import (
	"io"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// decodeReader wraps r so a UTF-16 (LE or BE) or UTF-8 byte-order mark is
// honoured and transcoded to plain UTF-8, with a leading UTF-8 BOM stripped.
// Files with no BOM (plain ASCII/UTF-8) pass through unchanged.
//
// SDRTrunk and other Windows-born tools frequently export alias files as
// UTF-16-with-BOM; read as bytes those decode to interleaved-NUL mojibake.
// Normalising at the reader boundary means every loader (CSV and JSON, file
// and reader) gets clean UTF-8 before parsing.
func decodeReader(r io.Reader) io.Reader {
	// UTF-8 by default; BOMOverride switches to UTF-16 LE/BE when their BOM
	// is present and consumes a leading UTF-8 BOM.
	dec := unicode.BOMOverride(unicode.UTF8.NewDecoder())
	return transform.NewReader(r, dec)
}

// sanitizeText reduces s to printable ASCII (0x20..0x7E), dropping control
// characters, NULs, and any non-ASCII rune. Radio aliases are ASCII in
// practice, so this strips the CJK / private-use mojibake seen in junk
// SDRTrunk imports without losing real operator data.
func sanitizeText(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if b := s[i]; b >= 0x20 && b < 0x7F {
			out = append(out, b)
		}
	}
	return string(out)
}
