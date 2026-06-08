// Package configtui is a standalone Bubble Tea terminal Config Builder/Editor
// that mirrors the web Config Builder (web/configbuilder). It is reflection-
// driven: it walks config.Config so every field is editable automatically (it
// cannot drift from the schema), with a metadata table supplying the web-like
// polish — labels, help, select options, Hz formatting, fieldset grouping and
// the AdvancedJSON long-tail. It operates on the local filesystem via the
// config package directly (no daemon, no browser).
//
// The labels / help / select options / Hz+freq-list flags all come from the
// shared registry in internal/configbuilder (sections.go + fieldmeta.go) so the
// terminal builder and the web builder present identical help from one source.
package configtui

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

// selOpt is one select option (value + display label).
type selOpt struct{ Value, Label string }

// fieldMeta is the TUI-side view of a field's metadata, projected from the
// shared configbuilder.FieldMeta. Absent ⇒ label is the humanized field name
// and the widget is derived from the Go kind.
type fieldMeta struct {
	Label    string
	Help     string
	Options  []selOpt // makes a string render as a select
	Hz       bool     // uint32 → Hz widget (MHz/Hz)
	FreqList bool     // []uint32 → frequency-list widget
	Hidden   bool
}

// metaFor returns the metadata for structName.field (zero value if none),
// projected from the shared configbuilder registry so the TUI and web builder
// stay identical.
func metaFor(structName, field string) fieldMeta {
	fm := configbuilder.FieldMetaFor(structName, field)
	out := fieldMeta{Label: fm.Label, Help: fm.Help, Hz: fm.Hz, FreqList: fm.FreqList}
	for _, o := range fm.Options {
		out.Options = append(out.Options, selOpt{Value: o.Value, Label: o.Label})
	}
	return out
}

// isAdvanced reports whether a field is in the AdvancedJSON long-tail.
func isAdvanced(structName, field string) bool {
	return configbuilder.IsAdvanced(structName, field)
}

// humanize turns a Go field name into a label ("CallTimeoutMs" → "Call timeout
// ms", "HTTPAddr" → "HTTP addr").
func humanize(name string) string {
	var b strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			// Insert a space at a lower→upper boundary, or at the end of an
			// acronym run (upper followed by lower).
			if !unicode.IsUpper(prev) || (next != 0 && unicode.IsLower(next)) {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	out := b.String()
	// Lower-case the tail words but keep the first character as-is.
	return out
}

// labelFor resolves a field's display label (metadata override or humanized).
func labelFor(structName string, f reflect.StructField) string {
	if m := metaFor(structName, f.Name); m.Label != "" {
		return m.Label
	}
	return humanize(f.Name)
}
