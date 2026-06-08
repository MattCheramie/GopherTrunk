package configbuilder

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// hzUnitRE strips a trailing/embedded "MHz"/"Hz" unit (case-insensitive) so
// operators can type "851.0375 MHz" or "851037500".
var hzUnitRE = regexp.MustCompile(`(?i)mhz|hz`)

// HzToDisplay renders an integer-Hz value as trimmed MHz (e.g. 851037500 →
// "851.0375 MHz", 852000000 → "852 MHz"). Mirrors hzToDisplay in
// web/configbuilder/src/components/fields.tsx.
func HzToDisplay(hz uint32) string {
	if hz == 0 {
		return "0"
	}
	s := strconv.FormatFloat(float64(hz)/1e6, 'f', 6, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s + " MHz"
}

// ParseFreqList parses a comma/semicolon/newline-separated list of
// frequencies, each MHz (values < 10000) or Hz, into integer Hz. Mirrors
// parseFreqList in web/configbuilder/src/components/fields.tsx. Used by the
// freq-list widget and, taking element [0], the single-Hz widget.
func ParseFreqList(text string) []uint32 {
	var out []uint32
	for _, raw := range strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	}) {
		tok := strings.TrimSpace(hzUnitRE.ReplaceAllString(raw, ""))
		if tok == "" {
			continue
		}
		n, err := strconv.ParseFloat(tok, 64)
		if err != nil || n <= 0 {
			continue
		}
		if n < 10000 {
			out = append(out, uint32(math.Round(n*1e6)))
		} else {
			out = append(out, uint32(math.Round(n)))
		}
	}
	return out
}

// ParseHz parses a single frequency (MHz or Hz) to integer Hz, returning 0
// when nothing parses — the single-field analogue of ParseFreqList.
func ParseHz(text string) uint32 {
	list := ParseFreqList(text)
	if len(list) == 0 {
		return 0
	}
	return list[0]
}
