package iio

import (
	"os"
	"testing"
)

func TestParseReconstructedPlutoContext(t *testing.T) {
	raw, err := os.ReadFile("testdata/plutosky_rev_c_context.xml")
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := ParseContext(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := ctx.Attrs["fw_version"]; got != "v0.38" {
		t.Errorf("fw_version = %q", got)
	}
	rx := ctx.FindDevice("cf-ad9361-lpc")
	if rx == nil || rx.ID != "iio:device3" {
		t.Fatalf("cf-ad9361-lpc = %+v", rx)
	}
	i := rx.FindChannel("voltage0", false)
	if i == nil || i.Scan == nil || i.Scan.Bits != 12 || i.Scan.Length != 16 || !i.Scan.Signed || i.Scan.BigEndian {
		t.Fatalf("voltage0 scan = %+v", i)
	}
	if got := rx.Mask(i, rx.FindChannel("voltage1", false)); got != "00000003" {
		t.Errorf("mask = %q, want 00000003", got)
	}
	phy := ctx.FindDevice("iio:device0")
	if phy == nil || phy.Name != "ad9361-phy" {
		t.Fatalf("lookup by id: %+v", phy)
	}
	// RX_LO by its label as well as its id.
	if phy.FindChannel("RX_LO", true) != phy.FindChannel("altvoltage0", true) {
		t.Error("RX_LO name and altvoltage0 id resolve differently")
	}
	// The PHY's voltage0 exists as both input and output.
	if phy.FindChannel("voltage0", false) == phy.FindChannel("voltage0", true) {
		t.Error("input and output voltage0 collapsed")
	}
}

// libiio numbers channels after sorting scan elements by index (context.c
// reorder_channels), and that number is the mask bit. A board that lists Q
// before I, or puts non-scan channels first, must still get the right bits.
func TestChannelNumbersFollowScanIndex(t *testing.T) {
	xml := []byte(`<context name="t"><device id="iio:device0" name="d">` +
		`<channel id="temp0" type="input"/>` +
		`<channel id="voltage1" type="input"><scan-element index="1" format="le:S12/16&gt;&gt;0"/></channel>` +
		`<channel id="voltage0" type="input"><scan-element index="0" format="le:S12/16&gt;&gt;0"/></channel>` +
		`</device></context>`)
	ctx, err := ParseContext(xml)
	if err != nil {
		t.Fatal(err)
	}
	d := ctx.Devices[0]
	want := map[string]int{"voltage0": 0, "voltage1": 1, "temp0": 2}
	for id, n := range want {
		if ch := d.FindChannel(id, false); ch == nil || ch.Number != n {
			t.Errorf("%s number = %+v, want %d", id, ch, n)
		}
	}
	if got := d.Mask(d.FindChannel("voltage1", false)); got != "00000002" {
		t.Errorf("mask = %q", got)
	}
}

func TestMaskSpansWordsMostSignificantFirst(t *testing.T) {
	d := &Device{}
	for i := 0; i < 40; i++ {
		d.Channels = append(d.Channels, &Channel{Number: i})
	}
	if d.MaskWords() != 2 {
		t.Fatalf("words = %d", d.MaskWords())
	}
	if got := d.Mask(d.Channels[0], d.Channels[33]); got != "0000000200000001" {
		t.Errorf("mask = %q", got)
	}
}

func TestParseScanFormat(t *testing.T) {
	cases := []struct {
		in                      string
		be, signed              bool
		bits, length, rep, shft int
	}{
		{"le:S12/16>>0", false, true, 12, 16, 1, 0},
		{"be:u24/32>>8", true, false, 24, 32, 1, 8},
		{"le:s16/16X2>>0", false, true, 16, 16, 2, 0},
	}
	for _, c := range cases {
		f, err := parseScanFormat("0", c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if f.BigEndian != c.be || f.Signed != c.signed || f.Bits != c.bits || f.Length != c.length || f.Repeat != c.rep || f.Shift != c.shft {
			t.Errorf("%s = %+v", c.in, f)
		}
	}
	for _, bad := range []string{"", "le", "le:S12", "xx:S12/16>>0", "le:S12/16"} {
		if _, err := parseScanFormat("0", bad); err == nil {
			t.Errorf("parseScanFormat(%q) = nil error", bad)
		}
	}
}

func TestParseRange(t *testing.T) {
	if lo, hi, ok := parseRange("[325000000 1 3800000000]"); !ok || lo != 325_000_000 || hi != 3_800_000_000 {
		t.Errorf("got %d %d %v", lo, hi, ok)
	}
	if _, _, ok := parseRange("garbage"); ok {
		t.Error("garbage parsed")
	}
}
