package configbuilder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestFieldHelpCoverage is a load-bearing dual-editor guard: every exported
// field of every config-package struct reachable from config.Config must have a
// non-empty Help in the shared FieldMeta registry. A new field fails here until
// help is authored — and because BOTH builders source their help from this
// registry, the schema change can't ship without updating both.
func TestFieldHelpCoverage(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var missing []string
	var walk func(rt reflect.Type)
	walk = func(rt reflect.Type) {
		for rt.Kind() == reflect.Ptr || rt.Kind() == reflect.Slice ||
			rt.Kind() == reflect.Array || rt.Kind() == reflect.Map {
			rt = rt.Elem()
		}
		if rt.Kind() != reflect.Struct || !strings.HasSuffix(rt.PkgPath(), "internal/config") {
			return
		}
		if seen[rt] {
			return
		}
		seen[rt] = true
		// The top-level Config struct's fields are the 23 sections; their help
		// is the section Instructions (checked by TestSectionMetaComplete), not
		// a per-field entry. Recurse into them but don't require a FieldMeta.
		isRoot := rt == reflect.TypeOf(config.Config{})
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			if !isRoot && FieldMetaFor(rt.Name(), f.Name).Help == "" {
				missing = append(missing, rt.Name()+"."+f.Name+" ("+f.Type.String()+")")
			}
			walk(f.Type)
		}
	}
	walk(reflect.TypeOf(config.Config{}))
	if len(missing) > 0 {
		t.Errorf("config fields missing Help in the shared FieldMeta registry (add an entry in fieldmeta.go for each — both Config Builders source help from it):\n  %s",
			strings.Join(missing, "\n  "))
	}
}

// TestSectionMetaComplete checks every Section has Instructions + a Docs link,
// so the shared section help both builders render is never blank.
func TestSectionMetaComplete(t *testing.T) {
	for _, s := range Sections() {
		if strings.TrimSpace(s.Instructions) == "" {
			t.Errorf("section %q has no Instructions", s.Key)
		}
		if strings.TrimSpace(s.DocURL) == "" || strings.TrimSpace(s.DocTitle) == "" {
			t.Errorf("section %q has no Docs link (DocTitle/DocURL)", s.Key)
		}
	}
}
