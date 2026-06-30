package webserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
)

func TestRecipeOpsEndpoint(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/cryptolab/recipe/ops")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var specs []struct {
		Name      string `json:"name"`
		Transform bool   `json:"transform"`
		Params    []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		} `json:"params"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&specs); err != nil {
		t.Fatal(err)
	}
	if len(specs) < 8 {
		t.Fatalf("expected the op catalogue, got %d", len(specs))
	}
	// xor must expose a hex key param.
	var ok bool
	for _, s := range specs {
		if s.Name == "xor" {
			for _, p := range s.Params {
				if p.Name == "key" && p.Kind == "hex" {
					ok = true
				}
			}
		}
	}
	if !ok {
		t.Fatal("xor op should expose a hex `key` param")
	}
}

func TestRecipeEndpointDecrypts(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	pt := []byte("DISPATCH UNIT SEVEN TO MAIN STREET NOW")
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
	id := uploadFile(t, ts.URL, "enc.bin", ct)

	body, _ := json.Marshal(map[string]any{
		"input_id": id,
		"steps": []map[string]any{
			{"op": "adp-decrypt", "params": map[string]any{"key": "abcdef0123", "mi": "090807060504030201"}},
			{"op": "stats"},
		},
	})
	resp, err := http.Post(ts.URL+"/api/v1/cryptolab/recipe", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("recipe status %d: %s", resp.StatusCode, b)
	}
	var out struct {
		FinalASCII string `json:"final_ascii"`
		Steps      []any  `json:"steps"`
		Error      string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Error != "" {
		t.Fatalf("recipe error: %s", out.Error)
	}
	if out.FinalASCII != string(pt) {
		t.Fatalf("final_ascii = %q, want %q", out.FinalASCII, pt)
	}
	if len(out.Steps) != 2 {
		t.Fatalf("want 2 step results, got %d", len(out.Steps))
	}
}
