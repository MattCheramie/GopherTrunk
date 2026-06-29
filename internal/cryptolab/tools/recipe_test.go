package tools

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
)

func TestRecipeToolRegistered(t *testing.T) {
	t.Parallel()
	if _, ok := cryptolab.Lookup("recipe"); !ok {
		t.Fatal("recipe tool not registered")
	}
}

func TestRecipeListOps(t *testing.T) {
	t.Parallel()
	res := runMode(t, "recipe", "run", []string{"-list"})
	if len(res.Findings) == 0 {
		t.Fatalf("-list returned no ops: %+v", res)
	}
}

// TestRecipeRunDecryptsADP drives the tool end-to-end: an ADP-encrypted input
// plus a two-step recipe (adp-decrypt → stats) must recover the plaintext.
func TestRecipeRunDecryptsADP(t *testing.T) {
	t.Parallel()
	pt := []byte("UNITS RESPOND CODE THREE NOW")
	key := []byte{0xAB, 0xCD, 0xEF, 0x01, 0x23}
	mi := []byte{9, 8, 7, 6, 5, 4, 3, 2, 1}
	ks, err := p25crypto.Keystream(p25crypto.AlgADP, key, mi, len(pt))
	if err != nil {
		t.Fatal(err)
	}
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ ks[i]
	}
	inFile := writeFile(t, "ct.bin", ct)
	recipeJSON := fmt.Sprintf(
		`[{"op":"adp-decrypt","key":"%s","mi":"%s"},{"op":"stats"}]`,
		hex.EncodeToString(key), hex.EncodeToString(mi))
	recipeFile := writeFile(t, "recipe.json", []byte(recipeJSON))

	res := runMode(t, "recipe", "run", []string{"-in", inFile, "-recipe", recipeFile})
	if got := fieldString(t, res, "final_ascii"); got != string(pt) {
		t.Fatalf("final_ascii = %q, want %q", got, pt)
	}
}

// TestRecipeYAMLStepsForm verifies the object-with-steps YAML form parses.
func TestRecipeYAMLStepsForm(t *testing.T) {
	t.Parallel()
	inFile := writeFile(t, "in.bin", []byte{0x00, 0xFF})
	recipeYAML := "steps:\n  - op: not\n  - op: not\n"
	recipeFile := writeFile(t, "recipe.yaml", []byte(recipeYAML))
	res := runMode(t, "recipe", "run", []string{"-in", inFile, "-recipe", recipeFile})
	if got := fieldString(t, res, "final_hex"); got != "00ff" {
		t.Fatalf("double-NOT via YAML steps form = %q, want 00ff", got)
	}
}
