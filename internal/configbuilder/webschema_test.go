package configbuilder

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// webRoundTripAllow lists config fields the web builder intentionally does NOT
// declare in web/configbuilder/src/api/types.ts; they round-trip via the
// TypeScript index signatures (`[k: string]: unknown`) on those interfaces and
// are reachable in the TUI's generic editor. Keyed by Go struct name. The
// AdvancedFields set (protocol knobs / device flags) is folded in too.
//
// Adding a field to a config struct WITHOUT giving it a typed entry in types.ts
// (a bespoke web editor) must be a deliberate choice recorded here.
var webRoundTripAllow = map[string][]string{
	// Rarely-tuned SDR watchdog interval; the web SDR section doesn't render
	// it (round-trips via the SDRConfig index signature). Editable in the TUI.
	"SDRConfig": {"WatchdogIntervalMs"},

	// The LoRa section round-trips through the web Config Builder's generic
	// index-signature editor; a bespoke typed editor in types.ts is a
	// follow-up. The fields remain fully editable in the TUI and via raw YAML.
	"Config":               {"LoRa"},
	"LoRaChannelConfig":    {"CenterHz", "Bandwidth", "Oversample", "SubChannels", "LoRaWANKeys"},
	"LoRaSubChannelConfig": {"OffsetHz", "SpreadingFactor", "SyncWord"},
	"LoRaWANKeyConfig":     {"DevAddr", "NwkSKey", "AppSKey"},

	// Per-system P25 site name catalogue (issue #698). Optional
	// presentation metadata surfaced via GET /api/v1/sites; it round-trips
	// through the SystemConfig index signature and the whole Sites array
	// (each a SiteConfig) is preserved as unknown JSON. Editable in the TUI
	// and raw YAML; a bespoke web editor is a follow-up.
	"SystemConfig": {"Sites"},
	"SiteConfig":   {"RFSS", "Site", "Name"},
}

// TestConfigSchemaCoveredByWebBuilder fails when a config.Config field has no
// counterpart in the web builder's TypeScript schema (types.ts) and is not on
// the round-trip allow-list — i.e. a config change that the web Config Builder
// would silently drop on save/load.
func TestConfigSchemaCoveredByWebBuilder(t *testing.T) {
	typesSrc := readWebTypes(t)

	allow := mergedWebAllow()
	fields := collectConfigStructFields()

	var missing []string
	for structName, fnames := range fields {
		for fname := range fnames {
			if allow[structName] != nil && allow[structName][fname] {
				continue
			}
			if !typesDeclaresField(typesSrc, fname) {
				missing = append(missing, structName+"."+fname)
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("config fields missing from web builder types.ts (add a typed field + editor, or allow-list as index-signature round-trip):\n  %s",
			strings.Join(missing, "\n  "))
	}
}

// mergedWebAllow folds AdvancedFields into webRoundTripAllow.
func mergedWebAllow() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	add := func(s, f string) {
		if out[s] == nil {
			out[s] = map[string]bool{}
		}
		out[s][f] = true
	}
	for s, fs := range AdvancedFields {
		for _, f := range fs {
			add(s, f)
		}
	}
	for s, fs := range webRoundTripAllow {
		for _, f := range fs {
			add(s, f)
		}
	}
	return out
}

// collectConfigStructFields walks every config-package struct reachable from
// config.Config and returns structName -> set of exported field names.
func collectConfigStructFields() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	seen := map[reflect.Type]bool{}
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice ||
			t.Kind() == reflect.Array || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		// Only descend into config-package structs (skip stdlib/foreign types).
		if !strings.HasSuffix(t.PkgPath(), "internal/config") {
			return
		}
		if seen[t] {
			return
		}
		seen[t] = true
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if out[t.Name()] == nil {
				out[t.Name()] = map[string]bool{}
			}
			out[t.Name()][f.Name] = true
			walk(f.Type)
		}
	}
	walk(reflect.TypeOf(config.Config{}))
	return out
}

var fieldDeclCache = map[string]*regexp.Regexp{}

// typesDeclaresField reports whether types.ts declares a property with the
// given Go field name (e.g. `Foo:` or `Foo?:` at the start of an indented
// line). Config field names are unique enough that presence anywhere is a
// sufficient coverage signal.
func typesDeclaresField(src, field string) bool {
	re := fieldDeclCache[field]
	if re == nil {
		re = regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(field) + `\??:`)
		fieldDeclCache[field] = re
	}
	return re.MatchString(src)
}

func readWebTypes(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/configbuilder/ -> repo root is ../../
	p := filepath.Join(filepath.Dir(thisFile), "..", "..",
		"web", "configbuilder", "src", "api", "types.ts")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read web types.ts (%s): %v", p, err)
	}
	return string(b)
}
