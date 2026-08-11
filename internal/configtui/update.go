package configtui

import (
	"reflect"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		// Modal owns all keys while open.
		if m.modal != nil {
			nm, cmd := m.modal.Update(msg, &m)
			m.modal = nm
			return m, cmd
		}
		// Inline editor owns keys while editing.
		if m.editing {
			return m.updateEditing(msg)
		}
		return m.updateKey(msg)
	}
	return m, nil
}

func (m Model) updateEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editing = false
		return m, nil
	case "enter":
		fv := m.cur().FieldByName(m.editRow.field)
		if err := commitText(fv, m.editRow, m.input.Value()); err != nil {
			m.pushErr(m.editRow.label + ": " + err.Error())
		} else {
			m.markDirty()
		}
		m.editing = false
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m.requestQuit()
	case "q":
		if m.focus == zoneSections {
			return m.requestQuit()
		}
	case "s", "ctrl+s":
		return m.startSave()
	case "n":
		return m.requestNew()
	case "o":
		m.modal = newOpenModal(&m)
		return m, nil
	case "v":
		m.revalidate()
		m.jumpFirstError()
		return m, nil
	case "y":
		m.modal = newYAMLModal(&m)
		return m, nil
	case "R":
		m.revert()
		return m, nil
	case "tab":
		if m.focus == zoneSections {
			m.focus = zoneForm
		} else {
			m.focus = zoneSections
		}
		return m, nil
	}
	if m.focus == zoneSections {
		return m.updateSections(msg)
	}
	return m.updateForm(msg)
}

func (m Model) updateSections(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(configbuilder.Sections())
	switch msg.String() {
	case "up", "k":
		if m.sectionIdx > 0 {
			m.sectionIdx--
			m.enterSection()
		}
	case "down", "j":
		if m.sectionIdx < n-1 {
			m.sectionIdx++
			m.enterSection()
		}
	case "enter", "right", "l":
		m.focus = zoneForm
		m.cursor = 0
	}
	return m, nil
}

func (m *Model) enterSection() {
	m.navpath = nil
	m.cursor = 0
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.cur().Kind() {
	case reflect.Slice:
		return m.updateList(msg)
	case reflect.Map:
		return m.updateMap(msg)
	default:
		return m.updateStructForm(msg)
	}
}

// updateStructForm and friends re-resolve the current value from the
// receiver (m.cur()) before mutating, so the reflect.Value points into the
// SAME config copy that this method returns — passing a Value bound to a
// caller's copy would silently drop edits.
func (m Model) updateStructForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := m.cur()
	rows := structRows(structNameOf(cur), cur)
	if m.cursor >= len(rows) {
		m.cursor = len(rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	// Per-system talkgroup editor.
	if structNameOf(cur) == "SystemConfig" && msg.String() == "t" {
		m.modal = newTalkgroupModal(&m)
		return m, nil
	}
	// RadioReference subscription check (uses the edited creds).
	if structNameOf(cur) == "RadioReferenceConfig" && msg.String() == "V" {
		m.modal = newRRVerifyModal(&m)
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(rows)-1 {
			m.cursor++
		}
	case "esc", "left", "h":
		return m.popOrSections()
	case "enter", "right", "l", " ":
		if len(rows) == 0 {
			return m, nil
		}
		return m.activateRow(rows[m.cursor], msg.String())
	}
	return m, nil
}

func (m Model) activateRow(r formRow, key string) (tea.Model, tea.Cmd) {
	fv := m.cur().FieldByName(r.field)
	switch r.kind {
	case kindText, kindNumber, kindNumberPtr, kindHz, kindFreqList, kindStringList:
		m.editing = true
		m.editRow = r
		m.input.SetValue(editText(fv, r))
		m.input.CursorEnd()
		m.input.Focus()
		return m, nil
	case kindSelect:
		cycleSelect(fv, r, 1)
		m.markDirty()
	case kindBool:
		toggleBool(fv)
		m.markDirty()
	case kindBoolPtr:
		cycleBoolPtr(fv)
		m.markDirty()
	case kindStruct:
		m.navpath = append(m.navpath, pathStep{Field: r.field})
		m.cursor = 0
	case kindMap:
		m.navpath = append(m.navpath, pathStep{Field: r.field})
		m.cursor = 0
	case kindPtr:
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
			m.markDirty()
		}
		m.navpath = append(m.navpath, pathStep{Field: r.field})
		m.cursor = 0
	case kindList:
		m.navpath = append(m.navpath, pathStep{Field: r.field})
		m.cursor = 0
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	slice := m.cur()
	isSystems := slice.Type().Elem().Name() == "SystemConfig"
	if isSystems {
		switch msg.String() {
		case "r":
			m.modal = newRRModal(&m)
			return m, nil
		case "i":
			m.modal = newImportModal(&m)
			return m, nil
		}
	}
	n := slice.Len()
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < n-1 {
			m.cursor++
		}
	case "a":
		appendItem(slice)
		m.cursor = slice.Len() - 1
		m.markDirty()
	case "d":
		if n > 0 {
			idx := m.cursor
			m.modal = newConfirmModal("Remove this item?", func(mm *Model) tea.Cmd {
				removeItem(mm.cur(), idx)
				mm.markDirty()
				return nil
			})
		}
	case "D":
		if n > 0 {
			dupItem(slice, m.cursor)
			m.markDirty()
		}
	case "J":
		if n > 0 {
			moveItem(slice, m.cursor, 1)
			if m.cursor < n-1 {
				m.cursor++
			}
			m.markDirty()
		}
	case "K":
		if n > 0 {
			moveItem(slice, m.cursor, -1)
			if m.cursor > 0 {
				m.cursor--
			}
			m.markDirty()
		}
	case "enter", "right", "l":
		if n > 0 {
			m.navpath = append(m.navpath, pathStep{Index: m.cursor})
			m.cursor = 0
		}
	case "esc", "left", "h":
		return m.popOrSections()
	}
	return m, nil
}

func (m Model) updateMap(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := sortedTabKeys()
	if m.cursor >= len(keys) {
		m.cursor = len(keys) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(keys)-1 {
			m.cursor++
		}
	case "enter", " ":
		// Web.Tabs: present(true)=shown. Toggle = delete (shown) / set false (hidden).
		fv := m.cur().FieldByName("Tabs")
		if fv.IsNil() {
			fv.Set(reflect.MakeMap(fv.Type()))
		}
		k := reflect.ValueOf(keys[m.cursor])
		if fv.MapIndex(k).IsValid() {
			fv.SetMapIndex(k, reflect.Value{}) // delete → shown (default)
		} else {
			fv.SetMapIndex(k, reflect.ValueOf(false)) // hidden
		}
		if fv.Len() == 0 {
			fv.Set(reflect.Zero(fv.Type()))
		}
		m.markDirty()
	case "esc", "left", "h":
		return m.popOrSections()
	}
	return m, nil
}

func (m Model) popOrSections() (tea.Model, tea.Cmd) {
	if len(m.navpath) > 0 {
		m.navpath = m.navpath[:len(m.navpath)-1]
		m.cursor = 0
		return m, nil
	}
	m.focus = zoneSections
	return m, nil
}

func (m *Model) jumpFirstError() {
	for i, s := range configbuilder.Sections() {
		if len(m.validation[s.Key]) > 0 {
			m.sectionIdx = i
			m.focus = zoneForm
			m.navpath = nil
			m.cursor = 0
			return
		}
	}
}
