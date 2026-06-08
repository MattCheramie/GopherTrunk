package api

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestTrunkingConfigFieldsSurfacedInBuilder is a guardrail against config
// fields silently becoming uneditable. Every exported field of
// config.TrunkingConfig must have a touchpoint in the web Config Builder:
// it must appear in the builder's TypeScript type mirror (api/types.ts)
// AND have a control in the Trunking section UI (sections/Trunking.tsx).
//
// The builder seeds the full config from GET /api/v1/config/defaults and
// round-trips sections it has no bespoke editor for untouched, so this
// check is scoped to TrunkingConfig — the bespoke-editor section the voice
// recording settings (voice_hangtime_ms, voice_call_grouping) live in.
// When a future change adds a field to TrunkingConfig, this test fails
// until that field is given a real editor surface in the builder. Update
// web/configbuilder/src/api/types.ts and
// web/configbuilder/src/sections/Trunking.tsx to make it pass.
func TestTrunkingConfigFieldsSurfacedInBuilder(t *testing.T) {
	t.Parallel()

	// Test runs with CWD = the package dir (internal/api); the repo root
	// is two levels up.
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(b)
	}

	const (
		typesRel   = "web/configbuilder/src/api/types.ts"
		sectionRel = "web/configbuilder/src/sections/Trunking.tsx"
	)
	types := read(typesRel)
	section := read(sectionRel)

	rt := reflect.TypeOf(config.TrunkingConfig{})
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		// The builder binds controls to Go field names (see the header
		// comment in types.ts), so we look for the exported field name.
		if !strings.Contains(types, f.Name) {
			t.Errorf("config.TrunkingConfig.%s is not mirrored in %s; add it to the TrunkingConfig interface", f.Name, typesRel)
		}
		if !strings.Contains(section, f.Name) {
			t.Errorf("config.TrunkingConfig.%s has no editor control in %s; add a touchpoint so operators can edit it", f.Name, sectionRel)
		}
	}
}
