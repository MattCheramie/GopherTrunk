package cryptolab_test

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"

	// Register all tools and subjects so Schema() reflects the full set.
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/subjects/motorola"
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/subjects/nxdn"
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/tools"
)

func TestSchemaCoversEveryRegisteredMode(t *testing.T) {
	t.Parallel()
	schema := cryptolab.Schema()
	if len(schema) == 0 {
		t.Fatal("Schema() is empty — are tools registered?")
	}
	// Every registered tool/mode must appear, and every mode must declare at
	// least one input parameter (its primary file), or the web UI can't run it.
	for _, t2 := range cryptolab.Tools() {
		for _, m := range t2.Modes() {
			found := false
			for _, ms := range schema {
				if ms.Tool == t2.Name() && ms.Mode == m.Name() {
					found = true
					if len(ms.Params) == 0 {
						t.Errorf("mode %s/%s has no params in the schema", t2.Name(), m.Name())
					}
					seen := map[string]bool{}
					for _, p := range ms.Params {
						if p.Name == "" || p.Kind == "" {
							t.Errorf("%s/%s: param with empty name/kind", t2.Name(), m.Name())
						}
						if seen[p.Name] {
							t.Errorf("%s/%s: duplicate param %q", t2.Name(), m.Name(), p.Name)
						}
						seen[p.Name] = true
					}
				}
			}
			if !found {
				t.Errorf("registered mode %s/%s missing from Schema()", t2.Name(), m.Name())
			}
		}
	}
}
