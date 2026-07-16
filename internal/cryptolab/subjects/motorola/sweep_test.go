package motorola

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/gauge"
)

// plantEvenHigh / plantOdd define a synthetic chosen-plaintext cipher whose
// per-position state depends only on the byte position (not the character), so
// the state at any swept position is constant across that sweep — exactly the
// property differential pinning relies on. Values are arbitrary but odd where
// an odd multiplier is required.
func plantEvenHigh(k int) uint8    { return uint8(50 + 7*k) }
func plantOddMul(k int) uint8      { return uint8(5+2*k) | 1 }
func plantOddHi(k int) uint8       { return uint8(80 + 13*k) }
func plantState(k int) gauge.State { return gauge.State{Hi: plantOddHi(k), Mul: plantOddMul(k)} }

// synthSweepCSV writes a chosen-plaintext corpus (pure cipher output, NO CRC)
// containing two single-position sweeps at length 6: position 0 and position 1,
// each varying the character over a distinct set while the rest stay 'A'.
func synthSweepCSV(t *testing.T) string {
	t.Helper()
	tab := Recovered()
	const n = 6
	encodeAlias := func(a string) string {
		enc := make([]byte, 2*n)
		for k := 0; k < n; k++ {
			enc[2*k] = tab.EncodeEven(plantEvenHigh(k))
			st := plantState(k)
			enc[2*k+1] = tab.EncodeOdd(st.Hi, st.Mul, a[k])
		}
		return hex.EncodeToString(enc)
	}
	var buf bytes.Buffer
	buf.WriteString("rid,talkgroup,encoded_hex,alias\n")
	id := 0
	emit := func(src, a string) {
		fmt.Fprintf(&buf, "%d,%s,%s,%s\n", id, src, encodeAlias(a), a)
		id++
	}
	for _, c := range "ABCDEFGH" { // sweep at position 0
		emit("s0", string(c)+"AAAAA")
	}
	for _, c := range "ABCDEFGH" { // sweep at position 1
		emit("s1", "A"+string(c)+"AAAA")
	}
	path := filepath.Join(t.TempDir(), "sweep.csv")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLoadChosenCSVKeepsAllBytes: LoadChosenCSV must NOT strip a trailing CRC —
// chosen-plaintext encoded_hex is pure cipher output, exactly 2·len(alias).
func TestLoadChosenCSVKeepsAllBytes(t *testing.T) {
	t.Parallel()
	rows, err := LoadChosenCSV(synthSweepCSV(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 16 {
		t.Fatalf("loaded %d rows, want 16", len(rows))
	}
	for _, r := range rows {
		if len(r.Enc) != 2*len(r.Alias) {
			t.Fatalf("alias %q: Enc=%d bytes, want %d (no CRC strip)", r.Alias, len(r.Enc), 2*len(r.Alias))
		}
	}
	// Contrast: LoadCSV would strip two bytes, shrinking Enc below 2·len.
	stripped, err := LoadCSV(synthSweepCSV(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stripped[0].Enc) != 2*len(stripped[0].Alias)-2 {
		t.Fatalf("LoadCSV Enc=%d, want CRC-stripped %d", len(stripped[0].Enc), 2*len(stripped[0].Alias)-2)
	}
}

// TestDetectSweepsAndPin: the two planted sweeps are detected, and differential
// pinning recovers the exact planted odd-position state (H and Modd).
func TestDetectSweepsAndPin(t *testing.T) {
	t.Parallel()
	tab := Recovered()
	rows, err := LoadChosenCSV(synthSweepCSV(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	sweeps := detectSweeps(rows, 3)
	if len(sweeps) != 2 {
		t.Fatalf("detected %d sweeps, want 2 (positions 0 and 1)", len(sweeps))
	}
	wantPos := map[int]bool{0: true, 1: true}
	for _, sw := range sweeps {
		if sw.Length != 6 || !wantPos[sw.CharPos] {
			t.Fatalf("unexpected sweep len=%d pos=%d", sw.Length, sw.CharPos)
		}
		delete(wantPos, sw.CharPos)
		ps, ok := pinSweep(tab, sw)
		if !ok {
			t.Fatalf("pinSweep failed for pos=%d", sw.CharPos)
		}
		if ps.Hodd != plantOddHi(sw.CharPos) || ps.Modd != plantOddMul(sw.CharPos) {
			t.Fatalf("pos=%d pinned H=%d Modd=%d, want H=%d Modd=%d",
				sw.CharPos, ps.Hodd, ps.Modd, plantOddHi(sw.CharPos), plantOddMul(sw.CharPos))
		}
		// L|1 must be the inverse of Modd and odd.
		if ps.LoddOr1&1 == 0 || ps.LoddOr1 != gauge.ModInv(ps.Modd) {
			t.Fatalf("pos=%d L|1=%d not a valid odd inverse of Modd=%d", sw.CharPos, ps.LoddOr1, ps.Modd)
		}
	}
}

// TestPinSweepRejectsMixedPrefix: a "sweep" whose rows do not actually share a
// fixed pre-char state must not pin (guards against a false positive).
func TestPinSweepRejectsMixedPrefix(t *testing.T) {
	t.Parallel()
	tab := Recovered()
	// Rows where the swept-byte value is not an affine function of the char
	// (odd byte scrambled) — pinning must reject.
	sw := Sweep{Length: 3, CharPos: 0, Rows: []Row{
		{Enc: []byte{tab.EncodeEven(10), 0x11, tab.EncodeEven(20), 0x22, tab.EncodeEven(30), 0x33}, Alias: "AAA"},
		{Enc: []byte{tab.EncodeEven(10), 0x99, tab.EncodeEven(20), 0x22, tab.EncodeEven(30), 0x33}, Alias: "BAA"},
		{Enc: []byte{tab.EncodeEven(10), 0x04, tab.EncodeEven(20), 0x22, tab.EncodeEven(30), 0x33}, Alias: "CAA"},
	}}
	if _, ok := pinSweep(tab, sw); ok {
		t.Fatal("pinSweep accepted a non-affine sweep; want reject")
	}
}

// TestDistinctDeltas exercises the two-state delta counter and its
// high-byte-difference detector.
func TestDistinctDeltas(t *testing.T) {
	t.Parallel()
	a := map[uint8]uint8{1: 10, 2: 20, 3: 30, 4: 40}
	// b = a + {61 for some, 200 for others}: two-value delta, one == want(61).
	b := map[uint8]uint8{1: 10 + 61, 2: 20 + 200, 3: 30 + 61, 4: 40 + 200}
	distinct, hasWant, common := distinctDeltas(a, b, 61)
	if common != 4 || distinct != 2 || !hasWant {
		t.Fatalf("distinctDeltas = distinct %d, hasWant %v, common %d; want 2,true,4", distinct, hasWant, common)
	}
	// A single-value delta not equal to want.
	c := map[uint8]uint8{1: 15, 2: 25, 3: 35, 4: 45}
	if d, has, _ := distinctDeltas(a, c, 61); d != 1 || has {
		t.Fatalf("uniform delta: distinct %d hasWant %v, want 1,false", d, has)
	}
}

// TestSweepModeRuns drives the full mode and checks the headline fields: two
// states pinned, and the low-byte observability floor reported (the sweeps are
// non-consecutive, so zero L_in->L_out transitions).
func TestSweepModeRuns(t *testing.T) {
	t.Parallel()
	env := cryptolab.Env{Stdout: io.Discard}
	res, err := sweepMode{}.Run(context.Background(), []string{"-csv", synthSweepCSV(t)}, env)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]any{}
	for _, f := range res.Fields {
		fields[f.Key] = f.Value
	}
	if fields["states_pinned"] != 2 {
		t.Fatalf("states_pinned = %v, want 2", fields["states_pinned"])
	}
	if fields["low_byte_transitions_observed"] != 0 {
		t.Fatalf("low_byte_transitions_observed = %v, want 0", fields["low_byte_transitions_observed"])
	}
	var sawFloor bool
	for _, n := range res.Notes {
		if bytes.Contains([]byte(n), []byte("observability floor")) {
			sawFloor = true
		}
	}
	if !sawFloor {
		t.Fatalf("expected the low-byte observability-floor note; notes=%v", res.Notes)
	}
}
