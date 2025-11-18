package graphql

import (
	"fmt"
	"io"
	"reflect"

	"github.com/llehouerou/gqlclient/internal/reflectutil"
)

// writeSliceQuery writes a minified query for a slice type to w.
func writeSliceQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
) error {
	if t.Elem().Kind() != reflect.Array {
		err := writeQuery(w, t.Elem(), reflectutil.IndexSafe(v, 0), false)
		if err != nil {
			return fmt.Errorf(
				"failed to write query for slice item `%v`: %w",
				t,
				err,
			)
		}
		return nil
	}
	// handle [][2]any like an ordered map
	return writeOrderedMapQuery(w, t, v)
}

// writeOrderedMapQuery writes a minified query for [][2]any pattern to w.
func writeOrderedMapQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
) error {
	if t.Elem().Len() != 2 {
		return fmt.Errorf("only arrays of len 2 are supported, got %v", t.Elem())
	}
	sliceOfPairs := v
	_, _ = io.WriteString(w, "{")
	for i := 0; i < sliceOfPairs.Len(); i++ {
		pair := sliceOfPairs.Index(i)
		// it.Value() returns any, so we need to use reflect.ValueOf
		// to cast it away
		key, val := pair.Index(0), reflect.ValueOf(pair.Index(1).Interface())
		keyString, ok := key.Interface().(string)
		if !ok {
			return fmt.Errorf("expected pair (string, %v), got (%v, %v)",
				val.Type(), key.Type(), val.Type())
		}
		_, _ = io.WriteString(w, keyString)
		err := writeQuery(w, val.Type(), val, false)
		if err != nil {
			return fmt.Errorf(
				"failed to write query for pair[1] `%v`: %w",
				val.Type(),
				err,
			)
		}
	}
	_, _ = io.WriteString(w, "}")
	return nil
}

// writeInterfaceQuery writes a minified query for an interface type to w.
func writeInterfaceQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
	inline bool,
) error {
	val := reflect.ValueOf(v.Interface())
	if !val.IsValid() {
		return nil
	}
	// Check if the interface contains a nil pointer
	kind := val.Kind()
	if (kind == reflect.Ptr || kind == reflect.Interface || kind == reflect.Slice ||
		kind == reflect.Map || kind == reflect.Chan || kind == reflect.Func) &&
		val.IsNil() {
		return nil
	}
	err := writeQuery(w, val.Type(), val, inline)
	if err != nil {
		return fmt.Errorf("failed to write query for interface `%v`: %w", t, err)
	}
	return nil
}
