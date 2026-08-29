package motorola

import "testing"

// Frequencies cross-checked against trunk-recorder
// SmartnetParser::get_freq (800 standard/rebanded/splinter, 900).
func TestBandPlan800Standard(t *testing.T) {
	plan, ok := ParseBandPlan("")
	if !ok || plan.Name() != "800_standard" {
		t.Fatalf("default plan = %s, ok=%v", plan.Name(), ok)
	}
	cases := []struct {
		ch   uint16
		want uint32
	}{
		{0x000, 851_012_500},
		// Issue #1143 reporter's control channel.
		{0x08E, 854_562_500},
		{0x2CF, 868_987_500}, // 851.0125 + 0.025*719, top of the low segment
		{0x2D0, 866_000_000},
		{0x2F7, 866_975_000},
		{0x32F, 867_000_000},
		{0x3BE, 868_975_000},
		{0x3C1, 867_425_000},
	}
	for _, c := range cases {
		got, ok := plan.Frequency(c.ch)
		if !ok {
			t.Errorf("Frequency(%#x) not a channel", c.ch)
			continue
		}
		if got != c.want {
			t.Errorf("Frequency(%#x) = %d, want %d", c.ch, got, c.want)
		}
	}
	for _, ch := range []uint16{0x2F8, 0x308, 0x30B, 0x320, 0x3FF} {
		if plan.IsChannel(ch) {
			t.Errorf("IsChannel(%#x) = true; command values must not read as channels", ch)
		}
	}
}

func TestBandPlan800Rebanded(t *testing.T) {
	plan, _ := ParseBandPlan("800_rebanded")
	if hz, ok := plan.Frequency(0x000); !ok || hz != 851_012_500 {
		t.Errorf("rebanded ch 0 = %d, %v", hz, ok)
	}
	if hz, ok := plan.Frequency(0x1B8); !ok || hz != 851_025_000 {
		t.Errorf("rebanded ch 0x1B8 = %d, %v; want 851025000", hz, ok)
	}
	if plan.IsChannel(0x230) {
		t.Error("rebanded 0x230 should not be a channel")
	}
	if hz, ok := plan.Frequency(0x2D0); !ok || hz != 866_000_000 {
		t.Errorf("rebanded shared segment 0x2D0 = %d, %v", hz, ok)
	}
}

func TestBandPlan800Splinter(t *testing.T) {
	plan, _ := ParseBandPlan("800_splinter")
	if hz, ok := plan.Frequency(0x000); !ok || hz != 851_000_000 {
		t.Errorf("splinter ch 0 = %d, %v; want 851000000", hz, ok)
	}
	if hz, ok := plan.Frequency(0x258); !ok || hz != 866_012_500 {
		t.Errorf("splinter ch 0x258 = %d, %v; want 866012500", hz, ok)
	}
}

func TestBandPlan900(t *testing.T) {
	plan, ok := ParseBandPlan("900")
	if !ok {
		t.Fatal("900 plan not recognised")
	}
	if hz, ok := plan.Frequency(0x000); !ok || hz != 935_012_500 {
		t.Errorf("900 ch 0 = %d, %v", hz, ok)
	}
	if hz, ok := plan.Frequency(0x1DE); !ok || hz != 935_012_500+12_500*0x1DE {
		t.Errorf("900 ch 0x1DE = %d, %v", hz, ok)
	}
	if plan.IsChannel(0x1DF) {
		t.Error("900 plan 0x1DF should not be a channel")
	}
}

func TestParseBandPlanUnknownFallsBack(t *testing.T) {
	plan, ok := ParseBandPlan("obt_custom")
	if ok {
		t.Error("unknown plan name reported ok")
	}
	if plan.Name() != "800_standard" {
		t.Errorf("fallback plan = %s", plan.Name())
	}
}
