package mbelib

import (
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestStubBuildLeavesRegistryUntouched documents what the default
// (no-tag) build looks like to callers: the package compiles, init
// is a no-op, and the `imbe` / `ambe2` factories aren't registered.
//
// When built with `-tags mbelib && cgo`, the wrapper init() registers
// both factories; that path isn't exercised by CI but is covered by
// the build constraint in cgo_mbelib.go.
func TestStubBuildLeavesRegistryUntouched(t *testing.T) {
	// In the default build the stub is in effect. Asking the registry
	// for `imbe` / `ambe2` should fail with the package's standard
	// "unknown vocoder" wording. With -tags mbelib, both lookups
	// succeed — both branches are valid.
	for _, name := range []string{"imbe", "ambe2"} {
		v, err := voice.DefaultRegistry.New(name)
		if err == nil {
			// mbelib build path: must produce a working Vocoder.
			if v == nil {
				t.Errorf("New(%q) returned (nil, nil)", name)
				continue
			}
			if v.Name() != name {
				t.Errorf("vocoder name = %q, want %q", v.Name(), name)
			}
			_ = v.Close()
			continue
		}
		// Stub build path: the error message comes from the
		// registry itself.
		if !strings.Contains(err.Error(), "unknown vocoder") {
			t.Errorf("New(%q) error = %q; want \"unknown vocoder\" text", name, err)
		}
	}
}

// TestPackageImports records the compile-time import dependency on
// internal/voice. Without this reference Go would happily drop the
// import in the stub build (since neither file uses it), and the
// package's "always-built" surface would silently lose its connection
// to the registry. This test compiles in both build flavors.
func TestPackageImports(t *testing.T) {
	// Touch the registry through the public surface to make sure
	// the import path stays alive.
	if voice.DefaultRegistry == nil {
		t.Fatal("voice.DefaultRegistry is nil")
	}
}
