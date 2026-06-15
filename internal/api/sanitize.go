package api

import (
	"encoding"
	"encoding/json"
	"math"
	"reflect"
)

// finiteOr0 returns f unless it is NaN or ±Inf, in which case it returns 0.
// encoding/json cannot represent non-finite floats, so a stray Inf/NaN from a
// divide-by-zero or log-of-zero on a dead/marginal carrier would otherwise fail
// json.Marshal and break the event stream the moment it was encoded (issue #648).
// Mirrors hunt.finiteOr0; kept local so package api need not depend on hunt.
func finiteOr0(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

var (
	jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

// scrubNonFinite returns a JSON-safe copy of v in which every float64/float32
// that is NaN or ±Inf is replaced by 0. It is the categorical wire-boundary net
// for issue #648: the live event stream and the siglab REST responses carry
// floats computed across many protocols (SNR, EVM, confidence, dB power, …), any
// of which can go non-finite on a marginal carrier, and encoding/json rejects
// those — tearing down the WebSocket the moment one reaches the encoder.
//
// It copies as it descends, so the caller's value — which for a passed-through
// event payload is shared with every other bus subscriber, the NDJSON sink and
// the canonical store — is never mutated. On any unexpected reflection panic it
// falls back to returning v unchanged; the marshal-first guard at the WS/SSE
// boundary then logs and skips that one frame rather than crashing the stream.
func scrubNonFinite(v any) (out any) {
	defer func() {
		if recover() != nil {
			out = v
		}
	}()
	if v == nil {
		return nil
	}
	return scrubValue(reflect.ValueOf(v)).Interface()
}

// scrubValue is the reflection core. It returns a fresh value of the same type
// as rv with non-finite floats zeroed, allocating new containers as it descends
// so it never writes through to rv.
func scrubValue(rv reflect.Value) reflect.Value {
	t := rv.Type()

	// Types that marshal themselves (time.Time, etc.) or are non-float scalars
	// cannot expose a JSON-visible non-finite float — return them untouched and
	// skip the descent. This keeps the common int/string-only events cheap.
	if t.Implements(jsonMarshalerType) || t.Implements(textMarshalerType) ||
		reflect.PointerTo(t).Implements(jsonMarshalerType) ||
		reflect.PointerTo(t).Implements(textMarshalerType) {
		return rv
	}

	switch rv.Kind() {
	case reflect.Float64, reflect.Float32:
		out := reflect.New(t).Elem()
		out.SetFloat(finiteOr0(rv.Float()))
		return out

	case reflect.Pointer:
		if rv.IsNil() {
			return rv
		}
		out := reflect.New(t.Elem())
		out.Elem().Set(scrubValue(rv.Elem()))
		return out

	case reflect.Interface:
		if rv.IsNil() {
			return rv
		}
		out := reflect.New(t).Elem()
		out.Set(scrubValue(rv.Elem()))
		return out

	case reflect.Struct:
		out := reflect.New(t).Elem()
		out.Set(rv) // verbatim copy first (incl. unexported fields)
		scrubFieldsInPlace(out)
		return out

	case reflect.Slice:
		if rv.IsNil() {
			return rv
		}
		out := reflect.MakeSlice(t, rv.Len(), rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out.Index(i).Set(scrubValue(rv.Index(i)))
		}
		return out

	case reflect.Array:
		out := reflect.New(t).Elem()
		for i := 0; i < rv.Len(); i++ {
			out.Index(i).Set(scrubValue(rv.Index(i)))
		}
		return out

	case reflect.Map:
		if rv.IsNil() {
			return rv
		}
		out := reflect.MakeMapWithSize(t, rv.Len())
		for iter := rv.MapRange(); iter.Next(); {
			out.SetMapIndex(iter.Key(), scrubValue(iter.Value()))
		}
		return out

	default:
		return rv
	}
}

// scrubFieldsInPlace zeroes non-finite floats in the settable fields of the
// addressable struct value sv (a freshly-allocated copy — never the caller's
// original). It is split out so it can also reach the promoted exported fields
// of an embedded unexported struct, which encoding/json marshals even though the
// embedded field itself is not directly settable.
func scrubFieldsInPlace(sv reflect.Value) {
	t := sv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := sv.Field(i)
		if f.CanSet() {
			f.Set(scrubValue(f))
			continue
		}
		// Unexported field. If it is an embedded value struct, json promotes its
		// exported fields — its exported sub-fields are settable through f even
		// though f itself is not — so descend to scrub them. Plain unexported
		// fields (and embedded unexported pointers) are left as the verbatim copy:
		// json ignores the former, and the marshal-first guard covers the latter
		// exotic shared-pointer case.
		if t.Field(i).Anonymous && f.Kind() == reflect.Struct {
			scrubFieldsInPlace(f)
		}
	}
}
