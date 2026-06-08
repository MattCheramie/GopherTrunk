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

// TestMetaTableFieldsExist fails if a metadata override references a config
// field that no longer exists (renamed/removed) — keeping the TUI's polish in
// sync with the schema.
func TestMetaTableFieldsExist(t *testing.T) {
	fields := collectConfigFields()
	for key := range metaTable {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 || !fields[parts[0]][parts[1]] {
			t.Errorf("metaTable key %q does not match any config field", key)
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
