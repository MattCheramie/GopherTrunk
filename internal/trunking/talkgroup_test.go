package trunking

import (
	"strings"
	"testing"
)

func TestTalkgroupDBLookup(t *testing.T) {
	d := NewTalkgroupDB()
	d.Add(&TalkGroup{ID: 1234, AlphaTag: "FIRE-DISP", Priority: 2})
	if tg := d.Lookup(1234); tg == nil || tg.AlphaTag != "FIRE-DISP" {
		t.Errorf("Lookup(1234) = %+v", tg)
	}
	if tg := d.Lookup(9999); tg != nil {
		t.Errorf("Lookup(9999) should be nil, got %+v", tg)
	}
	if got := d.Len(); got != 1 {
		t.Errorf("Len = %d, want 1", got)
	}
}

func TestLoadCSVTrunkRecorderFormat(t *testing.T) {
	csv := `Decimal,Hex,Mode,Alpha Tag,Description,Tag,Group,Priority,Lockout
1234,4D2,D,FIRE-DISP,Fire Dispatch,Fire,Multi,2,
5678,162E,D,LOCK-ME,Test,Misc,Misc,L,
9000,2328,D,EXPLICIT,Explicit lockout,Misc,Misc,3,Y
`
	d := NewTalkgroupDB()
	n, err := d.LoadCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if n != 3 {
		t.Errorf("loaded %d, want 3", n)
	}
	fire := d.Lookup(1234)
	if fire == nil {
		t.Fatal("FIRE-DISP not loaded")
	}
	if fire.AlphaTag != "FIRE-DISP" || fire.Priority != 2 || fire.Tag != "Fire" {
		t.Errorf("FIRE-DISP = %+v", fire)
	}

	lk := d.Lookup(5678)
	if lk == nil || !lk.Lockout {
		t.Errorf("LOCK-ME priority=L should set Lockout: %+v", lk)
	}

	ex := d.Lookup(9000)
	if ex == nil || !ex.Lockout || ex.Priority != 3 {
		t.Errorf("EXPLICIT = %+v", ex)
	}
}

func TestLoadCSVStreamRecordMuteIcon(t *testing.T) {
	csv := `Decimal,Alpha Tag,Stream,Record,Mute,Icon
1,DEFAULTS,,,,
2,NOSTREAM,no,,,
3,NORECORD,,false,,
4,MUTED,,,yes,
5,ICONED,,,,fire-truck
`
	d := NewTalkgroupDB()
	if _, err := d.LoadCSV(strings.NewReader(csv)); err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	// Defaults: stream + record on, mute off, no icon.
	def := d.Lookup(1)
	if def == nil || !def.Stream || !def.Record || def.Mute || def.Icon != "" {
		t.Errorf("DEFAULTS = %+v", def)
	}
	if tg := d.Lookup(2); tg == nil || tg.Stream {
		t.Errorf("NOSTREAM should have Stream=false: %+v", tg)
	}
	if tg := d.Lookup(3); tg == nil || tg.Record {
		t.Errorf("NORECORD should have Record=false: %+v", tg)
	}
	if tg := d.Lookup(4); tg == nil || !tg.Mute {
		t.Errorf("MUTED should have Mute=true: %+v", tg)
	}
	if tg := d.Lookup(5); tg == nil || tg.Icon != "fire-truck" {
		t.Errorf("ICONED should have Icon=fire-truck: %+v", tg)
	}
}

func TestLoadCSVRequiresDecimal(t *testing.T) {
	csv := `Hex,Alpha Tag
4D2,FIRE
`
	d := NewTalkgroupDB()
	_, err := d.LoadCSV(strings.NewReader(csv))
	if err == nil {
		t.Error("expected error for missing Decimal column")
	}
}

func TestLoadJSON(t *testing.T) {
	js := `[
  {"id": 100, "alpha_tag": "OPS-1", "priority": 1},
  {"id": 200, "alpha_tag": "OPS-2", "priority": 5, "lockout": true}
]`
	d := NewTalkgroupDB()
	n, err := d.LoadJSON(strings.NewReader(js))
	if err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if n != 2 {
		t.Errorf("loaded %d, want 2", n)
	}
	if tg := d.Lookup(200); tg == nil || !tg.Lockout {
		t.Errorf("OPS-2 lockout flag missing: %+v", tg)
	}
}

func TestLoadCSVDefaultsScanTrue(t *testing.T) {
	// A CSV without a Scan column must leave every TG with Scan=true
	// so legacy configs keep their "follow every grant" behavior.
	csv := `Decimal,Alpha Tag
1,OPS-A
2,OPS-B
`
	d := NewTalkgroupDB()
	if _, err := d.LoadCSV(strings.NewReader(csv)); err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	for _, id := range []uint32{1, 2} {
		tg := d.Lookup(id)
		if tg == nil || !tg.Scan {
			t.Errorf("TG %d Scan = false, want true (default for missing column)", id)
		}
	}
}

func TestLoadCSVScanColumnOverrides(t *testing.T) {
	csv := `Decimal,Alpha Tag,Scan
1,OPS-A,Y
2,OPS-B,n
3,OPS-C,
`
	d := NewTalkgroupDB()
	if _, err := d.LoadCSV(strings.NewReader(csv)); err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if !d.Lookup(1).Scan {
		t.Errorf("TG 1 Scan = false, want true (explicit Y)")
	}
	if d.Lookup(2).Scan {
		t.Errorf("TG 2 Scan = true, want false (explicit n)")
	}
	if !d.Lookup(3).Scan {
		t.Errorf("TG 3 Scan = false, want true (empty cell falls back to default)")
	}
}

func TestLoadJSONDefaultsScanWhenAbsent(t *testing.T) {
	// JSON records without "scan" resolve to true; explicit
	// "scan":false carries through.
	js := `[
  {"id": 1, "alpha_tag": "A"},
  {"id": 2, "alpha_tag": "B", "scan": false},
  {"id": 3, "alpha_tag": "C", "scan": true}
]`
	d := NewTalkgroupDB()
	if _, err := d.LoadJSON(strings.NewReader(js)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if !d.Lookup(1).Scan {
		t.Errorf("TG 1 Scan = false, want true (no scan key)")
	}
	if d.Lookup(2).Scan {
		t.Errorf("TG 2 Scan = true, want false (explicit false)")
	}
	if !d.Lookup(3).Scan {
		t.Errorf("TG 3 Scan = false, want true (explicit true)")
	}
}
