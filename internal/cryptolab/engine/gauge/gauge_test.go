package gauge

import "testing"

func TestModInv(t *testing.T) {
	t.Parallel()
	for a := 1; a < 256; a += 2 {
		inv := ModInv(uint8(a))
		if uint8(a)*inv != 1 {
			t.Fatalf("ModInv(%d)=%d, product=%d, want 1", a, inv, uint8(a)*inv)
		}
	}
}

func TestModInvPanicsOnEven(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on even input")
		}
	}()
	ModInv(4)
}

func TestAllGaugesCount(t *testing.T) {
	t.Parallel()
	all := All()
	if len(all) != 32768 {
		t.Fatalf("len(All()) = %d, want 32768", len(all))
	}
	seen := map[Gauge]bool{}
	for _, g := range all {
		if g.Mg&1 == 0 {
			t.Fatalf("gauge with even Mg=%d", g.Mg)
		}
		if seen[g] {
			t.Fatalf("duplicate gauge %+v", g)
		}
		seen[g] = true
	}
}

func TestGaugeTransformRoundTrip(t *testing.T) {
	t.Parallel()
	states := []State{{0, 1}, {255, 3}, {128, 127}, {42, 17}, {7, 255}}
	for _, g := range All() {
		for _, s := range states {
			if got := g.FromTrue(g.ToTrue(s)); got != s {
				t.Fatalf("gauge %+v: FromTrue(ToTrue(%+v)) = %+v", g, s, got)
			}
		}
	}
}
