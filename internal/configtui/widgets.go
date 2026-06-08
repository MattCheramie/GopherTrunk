package configtui

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
)

// displayValue renders a field's current value for the form row.
func displayValue(fv reflect.Value, r formRow) string {
	switch r.kind {
	case kindSelect:
		cur := fmt.Sprint(scalarString(fv))
		for _, o := range r.options {
			if o.Value == cur {
				return o.Label
			}
		}
		if cur == "" {
			return "(unset)"
		}
		return cur
	case kindText:
		s := fv.String()
		if s == "" {
			return "(empty)"
		}
		return s
	case kindNumber:
		return scalarString(fv)
	case kindHz:
		return configbuilder.HzToDisplay(uint32(fv.Uint()))
	case kindBool:
		if fv.Bool() {
			return "[x] yes"
		}
		return "[ ] no"
	case kindBoolPtr:
		if fv.IsNil() {
			return "(default)"
		}
		if fv.Elem().Bool() {
			return "yes"
		}
		return "no"
	case kindStringList:
		n := fv.Len()
		if n == 0 {
			return "(none)"
		}
		return strings.Join(stringSlice(fv), ", ")
	case kindFreqList:
		n := fv.Len()
		if n == 0 {
			return "(none)"
		}
		return fmt.Sprintf("%d freq(s)", n)
	case kindList:
		return fmt.Sprintf("%d item(s)  →", fv.Len())
	case kindStruct:
		return "edit  →"
	case kindPtr:
		if fv.IsNil() {
			return "(none)"
		}
		return "set  →"
	case kindMap:
		return fmt.Sprintf("%d set  →", nonDefaultMapCount(fv))
	}
	return ""
}

// editText returns the raw text to seed an inline editor for a row.
func editText(fv reflect.Value, r formRow) string {
	switch r.kind {
	case kindHz:
		if fv.Uint() == 0 {
			return ""
		}
		return configbuilder.HzToDisplay(uint32(fv.Uint()))
	case kindFreqList:
		parts := make([]string, fv.Len())
		for i := 0; i < fv.Len(); i++ {
			parts[i] = configbuilder.HzToDisplay(uint32(fv.Index(i).Uint()))
		}
		return strings.Join(parts, ", ")
	case kindStringList:
		return strings.Join(stringSlice(fv), ", ")
	case kindNumber:
		return scalarString(fv)
	default:
		return fv.String()
	}
}

// commitText applies an edited string to the field value.
func commitText(fv reflect.Value, r formRow, text string) error {
	switch r.kind {
	case kindText, kindSelect:
		fv.SetString(text)
	case kindHz:
		fv.SetUint(uint64(configbuilder.ParseHz(text)))
	case kindFreqList:
		hz := configbuilder.ParseFreqList(text)
		out := reflect.MakeSlice(fv.Type(), len(hz), len(hz))
		for i, h := range hz {
			out.Index(i).SetUint(uint64(h))
		}
		fv.Set(out)
	case kindStringList:
		parts := []string{}
		for _, p := range strings.Split(text, ",") {
			if t := strings.TrimSpace(p); t != "" {
				parts = append(parts, t)
			}
		}
		out := reflect.MakeSlice(fv.Type(), len(parts), len(parts))
		for i, p := range parts {
			out.Index(i).SetString(p)
		}
		if len(parts) == 0 {
			fv.Set(reflect.Zero(fv.Type())) // nil so YAML omits empty list
		} else {
			fv.Set(out)
		}
	case kindNumber:
		return setNumber(fv, text)
	}
	return nil
}

func setNumber(fv reflect.Value, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		fv.Set(reflect.Zero(fv.Type()))
		return nil
	}
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(text, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	}
	return nil
}

// cycleSelect advances a select field to the next/previous option.
func cycleSelect(fv reflect.Value, r formRow, dir int) {
	if len(r.options) == 0 {
		return
	}
	cur := scalarString(fv)
	idx := 0
	for i, o := range r.options {
		if o.Value == cur {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(r.options)) % len(r.options)
	val := r.options[idx].Value
	if fv.Kind() == reflect.String {
		fv.SetString(val)
	} else {
		_ = setNumber(fv, val)
	}
}

func toggleBool(fv reflect.Value) { fv.SetBool(!fv.Bool()) }

// cycleBoolPtr cycles a *bool through default(nil)/true/false.
func cycleBoolPtr(fv reflect.Value) {
	switch {
	case fv.IsNil():
		b := true
		fv.Set(reflect.ValueOf(&b))
	case fv.Elem().Bool():
		b := false
		fv.Set(reflect.ValueOf(&b))
	default:
		fv.Set(reflect.Zero(fv.Type()))
	}
}

// ---- small reflect helpers ----

func scalarString(fv reflect.Value) string {
	switch fv.Kind() {
	case reflect.String:
		return fv.String()
	case reflect.Bool:
		return strconv.FormatBool(fv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(fv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(fv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(fv.Float(), 'g', -1, 64)
	}
	return ""
}

func stringSlice(fv reflect.Value) []string {
	out := make([]string, fv.Len())
	for i := range out {
		out[i] = fv.Index(i).String()
	}
	return out
}

func nonDefaultMapCount(fv reflect.Value) int {
	if fv.IsNil() {
		return 0
	}
	return fv.Len()
}
