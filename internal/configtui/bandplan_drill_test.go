package configtui

import (
	"strings"
	"testing"
)

// selectRow moves the form cursor to the row editing field and activates it.
func selectRow(t *testing.T, m Model, field string) Model {
	t.Helper()
	rows := structRows(structNameOf(m.cur()), m.cur())
	idx := -1
	for i, r := range rows {
		if r.field == field {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("no %s row in %s", field, structNameOf(m.cur()))
	}
	for m.cursor > 0 {
		m = send(m, "up")
	}
	for range idx {
		m = send(m, "down")
	}
	return send(m, "enter")
}

// systemWithBandPlan opens Trunking, adds a system, drills into it and then
// into its band plan (a pointer-to-struct field the drill allocates).
func systemWithBandPlan(t *testing.T, plan string) Model {
	t.Helper()
	m := newTestModel()
	m = gotoSection(m, "trunking")
	m = send(m, "enter", "a", "enter") // Systems list, add one, open it
	m = selectRow(t, m, plan)
	_ = m.View()
	return m
}

// The NXDN band plan's linear and table modes sit behind a pointer
// (SystemConfig.NXDNBandPlan *NXDNBandPlanConfig). The drill path's resolve
// walked FieldByName straight onto that pointer, so opening either mode
// panicked ("call of reflect.Value.FieldByName on ptr Value") and the TUI
// crashed — the #1261 report: "There is no way to enter the logical channel
// number table … the TUI app crashes when you try". DMR's band plan has the
// same shape.
func TestBandPlanLinearEditableInTUI(t *testing.T) {
	m := systemWithBandPlan(t, "NXDNBandPlan")
	m = selectRow(t, m, "Linear")
	_ = m.View()
	m = selectRow(t, m, "Offset")
	m = send(m, "backspace", "1", "enter")
	_ = m.View()
	lin := m.cfg.Trunking.Systems[0].NXDNBandPlan.Linear
	if lin == nil || lin.Offset != 1 {
		t.Fatalf("linear plan after edit: %+v", lin)
	}

	m = systemWithBandPlan(t, "DMRBandPlan")
	m = selectRow(t, m, "Linear")
	_ = m.View()
	if m.cfg.Trunking.Systems[0].DMRBandPlan.Linear == nil {
		t.Fatal("DMR linear plan not allocated")
	}
}

func TestBandPlanTableEditableInTUI(t *testing.T) {
	m := systemWithBandPlan(t, "NXDNBandPlan")
	m = selectRow(t, m, "Table")
	m = send(m, "a", "enter") // add a channel row and open it
	_ = m.View()
	m = selectRow(t, m, "Channel")
	m = send(m, "backspace", "9", "enter")
	_ = m.View()
	tab := m.cfg.Trunking.Systems[0].NXDNBandPlan.Table
	if len(tab) != 1 || tab[0].Channel != 9 {
		t.Fatalf("table after edit: %+v", tab)
	}
	// The list shows rows by channel, not "item 1".
	m = send(m, "esc")
	if v := m.View(); !strings.Contains(v, "Channel 9") {
		t.Errorf("table list does not title the row by its channel:\n%s", v)
	}
}
