package tetra

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// newSysInfoLogCC builds a ControlChannel whose log output is captured.
func newSysInfoLogCC(t *testing.T) (*ControlChannel, *bytes.Buffer) {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	var buf bytes.Buffer
	cc := New(Options{
		Bus:        bus,
		Log:        slog.New(slog.NewTextHandler(&buf, nil)),
		SystemName: "Sys",
	})
	return cc, &buf
}

func countSysInfoLines(buf *bytes.Buffer) int {
	return strings.Count(buf.String(), "tetra: SYSINFO cell parameters")
}

// TestSysInfoCellParamsNotLoggedOnCCKHyperframeAlternation is the #1350
// regression. A cell with air-interface encryption alternates the 16-bit
// SYSINFO field between the CCK identifier (flag set) and the hyperframe
// counter (flag clear). The change gate compared the alternating pair as a
// change every time, so the INFO line was written on EVERY SYSINFO (the
// reporter's 1.18 M lines / 424 MB in 8.7 h, two variants per cell — one with
// cck_id, one without — of almost equal count). Only a first decode, a real
// parameter change, or a real CCK id change may log.
func TestSysInfoCellParamsNotLoggedOnCCKHyperframeAlternation(t *testing.T) {
	cc, buf := newSysInfoLogCC(t)
	base := SysInfoExt{SCCHInUse: 1, LocationArea: 31, BSServiceDetails: 0x0f0}
	cck := base
	cck.CCKValid, cck.CounterOrCCK = true, 0x1234
	for i := 0; i < 50; i++ {
		hf := base
		hf.CounterOrCCK = uint16(1000 + i) // hyperframe advances
		cc.learnSysInfoExt(cck)
		cc.learnSysInfoExt(hf)
	}
	if n := countSysInfoLines(buf); n != 1 {
		t.Fatalf("SYSINFO cell parameters logged %d times over 100 alternating CCK/hyperframe broadcasts, want 1", n)
	}
	// The one line carries the CCK id.
	if !strings.Contains(buf.String(), "cck_id=0x1234") {
		t.Errorf("the logged line lost the CCK id:\n%s", buf.String())
	}

	// A hyperframe-only broadcast arriving FIRST must not hide a later CCK id:
	// learning the CCK for the first time is new information and logs.
	cc2, buf2 := newSysInfoLogCC(t)
	cc2.learnSysInfoExt(base)
	cc2.learnSysInfoExt(cck)
	cc2.learnSysInfoExt(base)
	cc2.learnSysInfoExt(cck)
	if n := countSysInfoLines(buf2); n != 2 {
		t.Fatalf("hyperframe-first sequence logged %d times, want 2 (first decode + first CCK id)", n)
	}
}

// TestSysInfoCellParamsLogsRealChanges: the gate must still let real changes
// through — a CCK id rotation (re-keying) and a cell-parameter change.
func TestSysInfoCellParamsLogsRealChanges(t *testing.T) {
	cc, buf := newSysInfoLogCC(t)
	base := SysInfoExt{SCCHInUse: 1, LocationArea: 31}
	cck := base
	cck.CCKValid, cck.CounterOrCCK = true, 0x1234
	hf := base
	hf.CounterOrCCK = 77

	cc.learnSysInfoExt(cck) // 1: first decode
	cc.learnSysInfoExt(hf)
	rekey := cck
	rekey.CounterOrCCK = 0x1235
	cc.learnSysInfoExt(rekey) // 2: CCK id changed
	cc.learnSysInfoExt(hf)
	cc.learnSysInfoExt(rekey)
	moved := hf
	moved.SCCHInUse = 2
	cc.learnSysInfoExt(moved) // 3: SCCH allocation changed
	cc.learnSysInfoExt(rekey) // 4: SCCH back to 1
	if n := countSysInfoLines(buf); n != 4 {
		t.Fatalf("logged %d times, want 4 (first decode, CCK rotation, SCCH change, SCCH change back):\n%s", n, buf.String())
	}
	if !strings.Contains(buf.String(), "cck_id=0x1235") {
		t.Errorf("the CCK rotation line did not carry the new CCK id")
	}
}
