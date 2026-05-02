package decode

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

// Fast-path unmarshaling: lift simple json.Decoder.Token() values into
// Go scalar fields without going through the json.Marshal +
// json.Unmarshal pingpong used by the reference path.
//
// This eliminates ~30x allocation amplification on typed payloads while
// preserving correctness for types that need encoding/json semantics
// (json.Unmarshaler implementations like time.Time, interface targets,
// json.RawMessage transcoding for non-RawMessage tokens, etc.) — those
// fall through to slowUnmarshalValue.

var (
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	jsonNumberType      = reflect.TypeOf(json.Number(""))
)

// jsonUnmarshalerCache memoizes whether a given target type, or a
// pointer to it, implements json.Unmarshaler. Result is constant per
// type and the check sits on every scalar decode, so caching the
// reflect.Type.Implements call is worth it.
var jsonUnmarshalerCache sync.Map // map[reflect.Type]bool

func implementsJSONUnmarshaler(t reflect.Type) bool {
	if cached, ok := jsonUnmarshalerCache.Load(t); ok {
		return cached.(bool) //nolint:errcheck // jsonUnmarshalerCache only ever stores bool
	}
	has := t.Implements(jsonUnmarshalerType) ||
		reflect.PointerTo(t).Implements(jsonUnmarshalerType)
	jsonUnmarshalerCache.Store(t, has)
	return has
}

// fastUnmarshal attempts to set v from a json.Decoder.Token() value
// without round-tripping through encoding/json. Return values:
//
//	(true,  nil)   — fast path handled the value.
//	(true,  err)   — fast path attempted and produced a real error.
//	                 The caller must NOT fall back, since the slow path
//	                 would surface the same error with extra work.
//	(false, nil)   — fast path declined; caller should use slow path.
func fastUnmarshal(value any, v reflect.Value) (bool, error) {
	targetType := v.Type()
	kind := targetType.Kind()

	// JSON null zeroes the target. Defer interfaces to the slow path
	// so encoding/json handles their semantics.
	if value == nil {
		if kind == reflect.Interface {
			return false, nil
		}
		v.Set(reflect.Zero(targetType))
		return true, nil
	}

	// Pointer target: allocate elem, recurse, swap in the new pointer.
	if kind == reflect.Ptr {
		elemType := targetType.Elem()
		if elemType.Kind() == reflect.Interface ||
			implementsJSONUnmarshaler(targetType) ||
			implementsJSONUnmarshaler(elemType) {
			return false, nil
		}
		newPtr := reflect.New(elemType)
		ok, err := fastUnmarshal(value, newPtr.Elem())
		if err != nil {
			return true, err
		}
		if !ok {
			return false, nil
		}
		v.Set(newPtr)
		return true, nil
	}

	// Interface targets need encoding/json's interface{} mapping.
	if kind == reflect.Interface {
		return false, nil
	}
	// Custom UnmarshalJSON wins over kind-based fast paths.
	if implementsJSONUnmarshaler(targetType) {
		return false, nil
	}

	switch tok := value.(type) {
	case string:
		if kind != reflect.String {
			return false, nil
		}
		v.SetString(tok)
		return true, nil

	case bool:
		if kind != reflect.Bool {
			return false, nil
		}
		v.SetBool(tok)
		return true, nil

	case json.Number:
		// json.Number target keeps the raw textual representation.
		if targetType == jsonNumberType {
			v.SetString(string(tok))
			return true, nil
		}
		switch kind {
		case reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			n, err := strconv.ParseInt(string(tok), 10, 64)
			if err != nil {
				//nolint:nilerr // not handled here; caller falls back to slow path
				return false, nil
			}
			if v.OverflowInt(n) {
				return true, fmt.Errorf(
					"cannot unmarshal %s into Go value of type %s",
					tok,
					targetType,
				)
			}
			v.SetInt(n)
			return true, nil
		case reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64:
			u, err := strconv.ParseUint(string(tok), 10, 64)
			if err != nil {
				//nolint:nilerr // not handled here; caller falls back to slow path
				return false, nil
			}
			if v.OverflowUint(u) {
				return true, fmt.Errorf(
					"cannot unmarshal %s into Go value of type %s",
					tok,
					targetType,
				)
			}
			v.SetUint(u)
			return true, nil
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(string(tok), 64)
			if err != nil {
				//nolint:nilerr // not handled here; caller falls back to slow path
				return false, nil
			}
			if v.OverflowFloat(f) {
				return true, fmt.Errorf(
					"cannot unmarshal %s into Go value of type %s",
					tok,
					targetType,
				)
			}
			v.SetFloat(f)
			return true, nil
		}
		return false, nil
	}

	return false, nil
}
