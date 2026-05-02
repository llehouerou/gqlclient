package graphql

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/llehouerou/gqlclient/internal/reflectutil"
)

// writeSliceQuery writes a minified query for a slice type to buf.
func writeSliceQuery(
	buf *bytes.Buffer,
	t reflect.Type,
	v reflect.Value,
) error {
	if t.Elem().Kind() != reflect.Array {
		err := writeQuery(buf, t.Elem(), reflectutil.IndexSafe(v, 0), false)
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
	return writeOrderedMapQuery(buf, t, v)
}

// writeOrderedMapQuery writes a minified query for [][2]any pattern to buf.
func writeOrderedMapQuery(
	buf *bytes.Buffer,
	t reflect.Type,
	v reflect.Value,
) error {
	if t.Elem().Len() != 2 {
		return fmt.Errorf("only arrays of len 2 are supported, got %v", t.Elem())
	}
	sliceOfPairs := v
	buf.WriteString("{")
	for i := range sliceOfPairs.Len() {
		pair := sliceOfPairs.Index(i)
		// it.Value() returns any, so we need to use reflect.ValueOf
		// to cast it away
		key, val := pair.Index(0), reflect.ValueOf(pair.Index(1).Interface())
		keyString, ok := key.Interface().(string)
		if !ok {
			return fmt.Errorf("expected pair (string, %v), got (%v, %v)",
				val.Type(), key.Type(), val.Type())
		}
		buf.WriteString(keyString)
		err := writeQuery(buf, val.Type(), val, false)
		if err != nil {
			return fmt.Errorf(
				"failed to write query for pair[1] `%v`: %w",
				val.Type(),
				err,
			)
		}
	}
	buf.WriteString("}")
	return nil
}

// writeInterfaceQuery writes a minified query for an interface type to buf.
func writeInterfaceQuery(
	buf *bytes.Buffer,
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
	err := writeQuery(buf, val.Type(), val, inline)
	if err != nil {
		return fmt.Errorf("failed to write query for interface `%v`: %w", t, err)
	}
	return nil
}
