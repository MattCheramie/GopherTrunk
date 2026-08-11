package configtui

import (
	"encoding/json"
	"reflect"
)

// rowKind classifies how a struct field is edited in the form.
type rowKind int

const (
	kindText rowKind = iota
	kindSelect
	kindNumber
	kindHz
	kindBool
	kindBoolPtr   // *bool tri-state (default/true/false)
	kindNumberPtr // *int/*uint tri-state (unset/value), e.g. *uint8 color_code
	kindStringList
	kindFreqList
	kindList        // []struct → drill into a list view
	kindStruct      // nested struct → drill into a sub-form
	kindPtr         // *struct → enable/clear + drill
	kindMap         // map[string]bool → drill into a toggle list
	kindUnsupported // a kind the engine can't edit (guarded by a test)
)

// formRow describes one editable row in a struct form view.
type formRow struct {
	field    string
	label    string
	help     string
	kind     rowKind
	advanced bool
	options  []selOpt
}

// pathStep is one hop into config.Config: a struct field by name, or a slice
// index (when Field == "").
type pathStep struct {
	Field string
	Index int
}

// resolve walks from an addressable root following path, returning the
// addressable value at the end. Slices/structs encountered through addressable
// parents stay addressable, so callers can Set scalars in place.
func resolve(root reflect.Value, path []pathStep) reflect.Value {
	v := root
	for _, s := range path {
		if s.Field != "" {
			v = v.FieldByName(s.Field)
		} else {
			v = v.Index(s.Index)
		}
	}
	return v
}

// structRows enumerates the editable rows of a struct value, non-advanced
// fields first then advanced ones (mirroring the web's "Advanced" grouping).
func structRows(structName string, v reflect.Value) []formRow {
	t := v.Type()
	var normal, adv []formRow
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		m := metaFor(structName, f.Name)
		if m.Hidden {
			continue
		}
		r := formRow{
			field:    f.Name,
			label:    labelFor(structName, f),
			help:     m.Help,
			advanced: isAdvanced(structName, f.Name),
			options:  m.Options,
			kind:     kindOf(f, m),
		}
		if r.advanced {
			adv = append(adv, r)
		} else {
			normal = append(normal, r)
		}
	}
	return append(normal, adv...)
}

func kindOf(f reflect.StructField, m fieldMeta) rowKind {
	ft := f.Type
	switch ft.Kind() {
	case reflect.String:
		if len(m.Options) > 0 {
			return kindSelect
		}
		return kindText
	case reflect.Bool:
		return kindBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		if m.Hz {
			return kindHz
		}
		if len(m.Options) > 0 {
			return kindSelect // e.g. BaudHz rendered as a choice
		}
		return kindNumber
	case reflect.Slice:
		if m.FreqList {
			return kindFreqList
		}
		el := ft.Elem()
		for el.Kind() == reflect.Ptr {
			el = el.Elem()
		}
		if el.Kind() == reflect.Struct {
			return kindList
		}
		if el.Kind() == reflect.String {
			return kindStringList
		}
		return kindUnsupported // e.g. []int — would corrupt on commit
	case reflect.Ptr:
		if ft.Elem().Kind() == reflect.Bool {
			return kindBoolPtr
		}
		switch ft.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return kindNumberPtr // e.g. *uint8 color_code (nil = unset)
		}
		if ft.Elem().Kind() == reflect.Struct {
			return kindPtr
		}
		return kindUnsupported
	case reflect.Struct:
		return kindStruct
	case reflect.Map:
		// Only map[string]bool (web.tabs) has a real editor today.
		if ft.Key().Kind() == reflect.String && ft.Elem().Kind() == reflect.Bool {
			return kindMap
		}
		return kindUnsupported
	}
	return kindUnsupported
}

// structNameOf returns the element struct type name of a value (deref ptr).
func structNameOf(v reflect.Value) string {
	t := v.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// ---- slice mutation helpers (operate on an addressable slice Value) --------

// appendItem grows the slice by one zero element, applies any new-item
// defaults, and returns the new length.
func appendItem(slice reflect.Value) int {
	elem := reflect.New(slice.Type().Elem()).Elem()
	applyNewDefaults(elem)
	slice.Set(reflect.Append(slice, elem))
	return slice.Len()
}

func removeItem(slice reflect.Value, i int) {
	n := slice.Len()
	if i < 0 || i >= n {
		return
	}
	reflect.Copy(slice.Slice(i, n), slice.Slice(i+1, n))
	slice.Set(slice.Slice(0, n-1))
}

func dupItem(slice reflect.Value, i int) {
	if i < 0 || i >= slice.Len() {
		return
	}
	cp := reflect.New(slice.Type().Elem())
	// deep copy via JSON round-trip (config structs are JSON-able)
	b, err := json.Marshal(slice.Index(i).Interface())
	if err == nil {
		_ = json.Unmarshal(b, cp.Interface())
	}
	out := reflect.MakeSlice(slice.Type(), 0, slice.Len()+1)
	out = reflect.AppendSlice(out, slice.Slice(0, i+1))
	out = reflect.Append(out, cp.Elem())
	if i+1 < slice.Len() {
		out = reflect.AppendSlice(out, slice.Slice(i+1, slice.Len()))
	}
	slice.Set(out)
}

func moveItem(slice reflect.Value, i, delta int) {
	j := i + delta
	if i < 0 || i >= slice.Len() || j < 0 || j >= slice.Len() {
		return
	}
	a, b := slice.Index(i), slice.Index(j)
	tmp := reflect.New(slice.Type().Elem()).Elem()
	tmp.Set(a)
	a.Set(b)
	b.Set(tmp)
}

// applyNewDefaults seeds a freshly-appended struct element with the same
// defaults the web's makeNew uses, so a new item is valid-ish out of the box.
func applyNewDefaults(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}
	set := func(name string, val any) {
		f := v.FieldByName(name)
		if f.IsValid() && f.CanSet() {
			rv := reflect.ValueOf(val)
			if rv.Type().ConvertibleTo(f.Type()) {
				f.Set(rv.Convert(f.Type()))
			}
		}
	}
	switch v.Type().Name() {
	case "PagingPOCSAGConfig", "PagingWidebandChannel":
		set("BaudHz", uint32(1200))
	}
	switch v.Type().Name() {
	case "PagingWidebandChannel":
		set("Protocol", "pocsag")
	case "EncryptionKeyConfig":
		set("Algorithm", "rc4")
	case "ConvChannelConfig":
		set("Mode", "nfm")
	case "DeviceConfig":
		set("Gain", "auto")
	}
}
