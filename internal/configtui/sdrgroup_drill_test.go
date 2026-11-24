package configtui

import "testing"

// TestSDRGroupEditableInTUI drives the sdr.groups drill path with keystrokes
// (TestEveryLeafEditable classifies kinds but never walks a path): add a
// group, name it, add a channel inside it and set its frequency.
func TestSDRGroupEditableInTUI(t *testing.T) {
	m := newTestModel()
	m = gotoSection(m, "sdr")
	m = selectRow(t, m, "Groups")
	m = send(m, "a", "enter") // add a group and open it
	_ = m.View()
	m = selectRow(t, m, "Name")
	m = send(m, "s", "o", "u", "t", "h", "enter")
	_ = m.View()
	m = selectRow(t, m, "Channels")
	m = send(m, "a", "enter") // add a channel and open it
	_ = m.View()
	m = selectRow(t, m, "System")
	m = send(m, "P", "enter")
	_ = m.View()
	groups := m.cfg.SDR.Groups
	if len(groups) != 1 || groups[0].Name != "south" {
		t.Fatalf("groups after edit: %+v", groups)
	}
	if chs := groups[0].Channels; len(chs) != 1 || chs[0].System != "P" {
		t.Fatalf("group channels after edit: %+v", chs)
	}
}
