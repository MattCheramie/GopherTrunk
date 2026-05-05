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
