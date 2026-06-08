package configtui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

// collectConfigFields walks every config-package struct reachable from
// config.Config and returns structName -> set of exported field names.
func collectConfigFields() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	seen := map[reflect.Type]bool{}
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice ||
			t.Kind() == reflect.Array || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || !strings.HasSuffix(t.PkgPath(), "internal/config") {
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

// TestFieldMetaKeysExist fails if the shared field-metadata registry references
// a config field that no longer exists (renamed/removed) — keeping the builders'
// polish in sync with the schema.
func TestFieldMetaKeysExist(t *testing.T) {
	fields := collectConfigFields()
	for key := range configbuilder.FieldMetas() {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 || !fields[parts[0]][parts[1]] {
			t.Errorf("FieldMetas key %q does not match any config field", key)
		}
	}
}

// TestAdvancedFieldsExist fails if the AdvancedFields allow-list references a
// field that no longer exists.
func TestAdvancedFieldsExist(t *testing.T) {
	fields := collectConfigFields()
	for structName, fs := range configbuilder.AdvancedFields {
		for _, f := range fs {
			if !fields[structName][f] {
				t.Errorf("AdvancedFields[%s] references missing field %q", structName, f)
			}
		}
	}
}

// TestSectionsCoverConfig fails if a top-level config.Config section is missing
// from configbuilder.Sections() (so a new section forces a nav entry in both
// the TUI and the web) — or if a Section names a field that doesn't exist.
func TestSectionsCoverConfig(t *testing.T) {
	fields := collectConfigFields()
	top := fields["Config"]
	have := map[string]bool{}
	for _, s := range configbuilder.Sections() {
		have[s.CfgField] = true
		if !top[s.CfgField] {
			t.Errorf("Sections() lists %q which is not a config.Config field", s.CfgField)
		}
	}
	for f := range top {
		if !have[f] {
			t.Errorf("config.Config.%s has no Section entry (add it to configbuilder.Sections so the builders show it)", f)
		}
	}
}

// TestReflectionReachesEverySection sanity-checks that the reflection form
// engine produces rows (or a list/map view) for every section's struct — i.e.
// no section is silently unrenderable.
func TestReflectionReachesEverySection(t *testing.T) {
	cfg := config.Default()
	root := reflect.ValueOf(&cfg).Elem()
	for _, s := range configbuilder.Sections() {
		v := root.FieldByName(s.CfgField)
		if !v.IsValid() {
			t.Errorf("section %s: no config field %s", s.Key, s.CfgField)
			continue
		}
		if v.Kind() == reflect.Struct {
			_ = structRows(v.Type().Name(), v) // must not panic
		}
	}
}

// TestEveryLeafEditable is the load-bearing dual-editor guard: it walks every
// field of every config-package struct reachable from config.Config and
// asserts the reflection form engine assigns a HANDLED widget kind (never
// kindUnsupported). A new field of a kind the TUI can't edit (e.g.
// map[string]string, []int, time.Duration, a foreign struct) fails here — so a
// config-schema change cannot land without the TUI being taught to edit it
// (the web side is guarded by configbuilder.TestConfigSchemaCoveredByWebBuilder).
func TestEveryLeafEditable(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(t reflect.Type)
	var bad []string
	walk = func(rt reflect.Type) {
		for rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}
		if rt.Kind() != reflect.Struct || !strings.HasSuffix(rt.PkgPath(), "internal/config") {
			return
		}
		if seen[rt] {
			return
		}
		seen[rt] = true
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			m := metaFor(rt.Name(), f.Name)
			if m.Hidden {
				continue
			}
			if kindOf(f, m) == kindUnsupported {
				bad = append(bad, rt.Name()+"."+f.Name+" ("+f.Type.String()+")")
			}
			// Recurse into nested config structs (incl. through slices/ptrs/maps).
			ft := f.Type
			for ft.Kind() == reflect.Ptr || ft.Kind() == reflect.Slice ||
				ft.Kind() == reflect.Array || ft.Kind() == reflect.Map {
				ft = ft.Elem()
			}
			walk(ft)
		}
	}
	walk(reflect.TypeOf(config.Config{}))
	if len(bad) > 0 {
		t.Errorf("config fields the TUI cannot edit (add a widget kind in kindOf, or metadata, for each):\n  %s",
			strings.Join(bad, "\n  "))
	}
}
