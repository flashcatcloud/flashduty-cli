package output

import (
	"reflect"
	"time"
)

// instantLike mirrors go-flashduty's Timestamp/TimestampMilli by method set so
// the JSON printer can recognise SDK timestamp fields without importing the
// SDK. The Time method excludes time.Time itself, whose zero value already
// marshals as a (consistently typed) RFC3339 string.
type instantLike interface {
	Time() time.Time
	Unix() int64
	IsZero() bool
}

var (
	instantLikeType = reflect.TypeOf((*instantLike)(nil)).Elem()
	anyType         = reflect.TypeOf((*any)(nil)).Elem()
)

// NullUnsetInstants returns a copy of v in which every unset (zero) SDK
// timestamp — go-flashduty Timestamp/TimestampMilli — is replaced by nil, so
// JSON output renders it as null instead of the bare integer 0.
//
// The SDK's Timestamp.MarshalJSON emits a quoted RFC3339 string for a set
// value but the bare integer 0 for the unset sentinel, so one field switches
// JSON type depending on record state (e.g. an alert's end_time is 0 while the
// alert is active but "2026-05-28T08:00:00Z" once recovered). That breaks jq
// arithmetic over mixed-state results. Set values keep their original typed
// value, so their custom MarshalJSON (and byte shape) is untouched;
// non-timestamp zero integers are untouched too.
//
// The transform rebuilds only the subtrees that contain a timestamp, via
// reflect.StructOf on the same field names/tags/order, so the encoded output
// keeps the original struct field order. TOON and table output are unaffected
// (they render the sentinel through String(), already always a string).
func NullUnsetInstants(v any) any {
	if v == nil {
		return nil
	}
	t := &instantTransform{memo: map[reflect.Type]xformed{}, active: map[reflect.Type]bool{}}
	out := t.value(reflect.ValueOf(v))
	if !out.IsValid() {
		return nil
	}
	return out.Interface()
}

// xformed is the memoized result of transforming one type: the replacement
// type (rt itself when nothing below it contains an instant) plus a changed
// flag. The flag — not type inequality — drives descent: an interface{} keeps
// its type yet its dynamic value may still hold an instant.
type xformed struct {
	typ     reflect.Type
	changed bool
}

// instantTransform carries the type memo across one NullUnsetInstants walk.
// active guards against self-referential types, which reflect.StructOf cannot
// rebuild — a cycle leaves that subtree unchanged.
type instantTransform struct {
	memo   map[reflect.Type]xformed
	active map[reflect.Type]bool
}

// value returns rv with unset instants nulled, assignable to the slot produced
// by xformType(rv.Type()).
func (t *instantTransform) value(rv reflect.Value) reflect.Value {
	xf := t.xformType(rv.Type())
	if !xf.changed {
		return rv
	}

	// Instants are int64-kind named types (and pointers to them), so this check
	// must run before the kind switch, not under the struct case.
	if isInstant(rv.Type()) {
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			return rv // already null on the wire
		}
		if rv.Interface().(instantLike).IsZero() {
			return reflect.Zero(anyType)
		}
		return rv
	}

	switch rv.Kind() {
	case reflect.Interface:
		if rv.IsNil() {
			return rv
		}
		inner := t.value(rv.Elem())
		out := reflect.New(rv.Type()).Elem()
		out.Set(inner)
		return out
	case reflect.Ptr:
		if rv.IsNil() {
			return rv
		}
		elem := t.value(rv.Elem())
		out := reflect.New(elem.Type())
		out.Elem().Set(elem)
		return out
	case reflect.Struct:
		out := reflect.New(xf.typ).Elem()
		for i := 0; i < rv.NumField(); i++ {
			out.Field(i).Set(t.value(rv.Field(i)))
		}
		return out
	case reflect.Slice:
		out := reflect.MakeSlice(xf.typ, rv.Len(), rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out.Index(i).Set(t.value(rv.Index(i)))
		}
		return out
	case reflect.Array:
		out := reflect.New(xf.typ).Elem()
		for i := 0; i < rv.Len(); i++ {
			out.Index(i).Set(t.value(rv.Index(i)))
		}
		return out
	case reflect.Map:
		out := reflect.MakeMapWithSize(xf.typ, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), t.value(iter.Value()))
		}
		return out
	default:
		return rv
	}
}

// xformType returns rt's replacement type and whether anything at or below rt
// may hold an instant.
func (t *instantTransform) xformType(rt reflect.Type) xformed {
	if xf, ok := t.memo[rt]; ok {
		return xf
	}
	if t.active[rt] {
		return xformed{rt, false}
	}

	xf := xformed{rt, false}
	switch {
	case isInstant(rt):
		// Widened to any so an unset value can become nil (JSON null); a set
		// value keeps the original concrete type and its MarshalJSON.
		xf = xformed{anyType, true}
	case rt.Kind() == reflect.Interface:
		// Dynamic values behind an interface may hold an instant even though
		// the static type stays the same.
		if rt.NumMethod() == 0 {
			xf = xformed{rt, true}
		}
	case rt.Kind() == reflect.Ptr:
		if elem := t.xformType(rt.Elem()); elem.changed {
			xf = xformed{anyType, true}
		}
	case rt.Kind() == reflect.Struct:
		xf = t.xformStruct(rt)
	case rt.Kind() == reflect.Slice:
		if elem := t.xformType(rt.Elem()); elem.changed {
			xf = xformed{reflect.SliceOf(elem.typ), true}
		}
	case rt.Kind() == reflect.Array:
		if elem := t.xformType(rt.Elem()); elem.changed {
			xf = xformed{reflect.ArrayOf(rt.Len(), elem.typ), true}
		}
	case rt.Kind() == reflect.Map:
		if val := t.xformType(rt.Elem()); val.changed {
			xf = xformed{reflect.MapOf(rt.Key(), val.typ), true}
		}
	}
	t.memo[rt] = xf
	return xf
}

// xformStruct rebuilds a struct type with instant(-holding) fields widened to
// any. It reports no change when no field needs rewriting, or when the struct
// is not safely rebuildable (unexported fields, or an embedded field whose
// type changed — reflect.StructOf cannot keep an unnamed rebuilt type
// anonymous).
func (t *instantTransform) xformStruct(rt reflect.Type) xformed {
	t.active[rt] = true
	defer delete(t.active, rt)

	fields := make([]reflect.StructField, rt.NumField())
	changed := false
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" { // unexported: leave the whole struct alone
			return xformed{rt, false}
		}
		fx := t.xformType(f.Type)
		if fx.changed {
			if f.Anonymous {
				return xformed{rt, false}
			}
			changed = true
			f.Type = fx.typ
		}
		fields[i] = f
	}
	if !changed {
		return xformed{rt, false}
	}
	return xformed{reflect.StructOf(fields), true}
}

func isInstant(rt reflect.Type) bool {
	return rt.Implements(instantLikeType)
}
