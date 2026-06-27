package motorola

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/gauge"
)

func TestLUTIsBijection(t *testing.T) {
	t.Parallel()
	tab := Recovered()
	var seen [256]bool
	for i := 0; i < 256; i++ {
		v := tab.Fwd[i]
		if seen[v] {
			t.Fatalf("LUT not a bijection at %d", i)
		}
		seen[v] = true
		if tab.Inv[v] != uint8(i) {
			t.Fatalf("inverse mismatch at %d", i)
		}
	}
}

func TestStripCRC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   []byte
		want []byte
		ok   bool
	}{
		{nil, nil, false},
		{[]byte{0x01}, nil, false},
		{[]byte{0x01, 0x02}, []byte{}, true},
		{[]byte{1, 2, 3, 4}, []byte{1, 2}, true},
	}
	for _, c := range cases {
		got, ok := StripCRC(c.in)
		if ok != c.ok || (ok && !bytes.Equal(got, c.want)) {
			t.Fatalf("StripCRC(%v) = %v,%v want %v,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestDecodeCharRoundTrip(t *testing.T) {
	t.Parallel()
	tab := Recovered()
	for _, char := range []byte("AZ09 -") {
		for _, hodd := range []uint8{0, 1, 42, 200, 255} {
			for _, modd := range []uint8{1, 3, 127, 255} { // odd multipliers
				eo := tab.EncodeOdd(hodd, modd, char)
				if got := tab.DecodeChar(hodd, modd, eo); got != char {
					t.Fatalf("round trip char=%q hodd=%d modd=%d eo=%d -> %q", char, hodd, modd, eo, got)
				}
			}
		}
	}
}

func TestCandidatesContainTrueState(t *testing.T) {
	t.Parallel()
	tab := Recovered()
	char, hodd, modd := byte('R'), uint8(123), uint8(45)
	eo := tab.EncodeOdd(hodd, modd, char)
	cands := tab.CandidatesForChar(char, eo)
	found := false
	for _, s := range cands {
		if s.Hi == hodd && s.Mul == modd {
			found = true
		}
	}
	if !found {
		t.Fatalf("true state (%d,%d) not among %d candidates", hodd, modd, len(cands))
	}
	if len(cands) != 128 {
		t.Fatalf("expected 128 candidates, got %d", len(cands))
	}
}

func TestGaugeMergedIndexCleanUnderIdentity(t *testing.T) {
	t.Parallel()
	// Plant observations where, in identity coordinates, Hodd = HPrev ^ HCur.
	var obs []componentObs
	for hp := 0; hp < 32; hp++ {
		for hc := 0; hc < 8; hc++ {
			obs = append(obs, componentObs{
				HPrev: uint8(hp * 7),
				HCur:  uint8(hc * 31),
				Hodd:  uint8(hp*7) ^ uint8(hc*31),
				Modd:  1,
			})
		}
	}
	if c := gaugeMergedIndexConflicts(gauge.Gauge{Mg: 1, Cg: 0}, obs); c != 0 {
		t.Fatalf("identity gauge conflicts = %d, want 0", c)
	}
	// The full sweep must therefore find a zero-conflict gauge.
	best := len(obs) + 1
	for _, g := range gauge.All() {
		if c := gaugeMergedIndexConflicts(g, obs); c < best {
			best = c
		}
	}
	if best != 0 {
		t.Fatalf("sweep best conflicts = %d, want 0", best)
	}
}

// synthCSV writes a synthetic corpus whose high bytes are fixed per position
// (so position-k contexts repeat across aliases and the cell solver can pin
// them) and whose odd bytes encode each alias character under a planted
// context-deterministic state. Returns the CSV path.
func synthCSV(t *testing.T) string {
	t.Helper()
	tab := Recovered()
	g := func(k int) uint8 { return uint8(40 + 3*k) } // fixed high byte per position
	plant := func(hp, hc uint8) gauge.State {
		return gauge.State{Hi: hp ^ hc ^ 0x3C, Mul: (hp*2 + hc*4) | 1}
	}
	aliases := []string{"RADIO-1", "UNIT-72", "ALPHA-9", "BRAVO-5", "DELTA-3", "ECHO-88"}

	var buf bytes.Buffer
	buf.WriteString("rid,talkgroup,encoded_hex,alias\n")
	for i, a := range aliases {
		n := len(a)
		enc := make([]byte, 2*n+2)
		for k := 0; k < n; k++ {
			hc := g(k)
			var hp uint8
			if k == 0 {
				hp = uint8(n)
			} else {
				hp = g(k - 1)
			}
			st := plant(hp, hc)
			enc[2*k] = tab.EncodeEven(hc)
			enc[2*k+1] = tab.EncodeOdd(st.Hi, st.Mul, a[k])
		}
		enc[2*n], enc[2*n+1] = 0xAB, 0xCD // CRC placeholder, stripped on load
		fmt.Fprintf(&buf, "%d,100,%s,%s\n", 3000+i, hex.EncodeToString(enc), a)
	}
	path := filepath.Join(t.TempDir(), "synth.csv")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadCSVAndHighBytes(t *testing.T) {
	t.Parallel()
	tab := Recovered()
	rows, err := LoadCSV(synthCSV(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("loaded %d rows, want 6", len(rows))
	}
	// High bytes must match the planted fixed-per-position sequence.
	for _, r := range rows {
		h := tab.HighBytes(r.Enc)
		for k := range h {
			if h[k] != uint8(40+3*k) {
				t.Fatalf("row %q H[%d]=%d, want %d", r.Alias, k, h[k], 40+3*k)
			}
		}
	}
}

func TestCellSolverPinsAndIsMonotone(t *testing.T) {
	t.Parallel()
	tab := Recovered()
	rows, err := LoadCSV(synthCSV(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	ingest := func(ck *cellCheckpoint) {
		for _, row := range rows {
			for _, o := range tab.CharObservations(row) {
				key := ctxKey(o.Ctx)
				cand := candidateSet(tab, o)
				if prev, ok := ck.Candidates[key]; ok {
					ck.Candidates[key] = intersect(prev, cand)
				} else {
					ck.Candidates[key] = cand
				}
			}
		}
	}

	ck := &cellCheckpoint{Version: 1, Candidates: map[string][]uint16{}}
	ingest(ck)
	sizes1 := map[string]int{}
	pinned1 := 0
	for k, c := range ck.Candidates {
		sizes1[k] = len(c)
		if len(c) == 1 {
			pinned1++
		}
	}
	if pinned1 == 0 {
		t.Fatal("expected at least one pinned context after first ingest")
	}

	// Second ingest of the same data must not grow any candidate set
	// (monotone) and must not reduce the pinned count.
	ingest(ck)
	pinned2 := 0
	for k, c := range ck.Candidates {
		if len(c) > sizes1[k] {
			t.Fatalf("context %s grew %d -> %d (not monotone)", k, sizes1[k], len(c))
		}
		if len(c) == 1 {
			pinned2++
		}
	}
	if pinned2 < pinned1 {
		t.Fatalf("pinned dropped %d -> %d", pinned1, pinned2)
	}
}

func TestModesRunEndToEnd(t *testing.T) {
	t.Parallel()
	csv := synthCSV(t)
	out := t.TempDir()
	tool := aliasTool{}
	for _, m := range tool.Modes() {
		m := m
		t.Run(m.Name(), func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			env := cryptolab.Env{OutDir: out, Stdout: &buf, Format: cryptolab.FormatText}
			res, err := m.Run(context.Background(), []string{"-csv", csv}, env)
			if err != nil {
				t.Fatalf("%s: %v", m.Name(), err)
			}
			if res == nil || res.Mode != m.Name() {
				t.Fatalf("%s: bad result %+v", m.Name(), res)
			}
		})
	}
}

func TestAliasToolRegistered(t *testing.T) {
	t.Parallel()
	tool, ok := cryptolab.Lookup("alias")
	if !ok {
		t.Fatal("alias tool not registered")
	}
	if len(tool.Modes()) != 4 {
		t.Fatalf("alias has %d modes, want 4", len(tool.Modes()))
	}
}
