package graphql

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/llehouerou/gqlclient/internal/reflectutil"
)

// query uses writeQuery to recursively construct
// a minified query string from the provided struct v.
//
// E.g., struct{Foo Int, BarBaz *bool} -> "{foo,barBaz}".
func query(v any) (string, error) {
	var buf bytes.Buffer
	err := writeQuery(&buf, reflect.TypeOf(v), reflect.ValueOf(v), false)
	if err != nil {
		return "", fmt.Errorf("failed to write query: %w", err)
	}
	return buf.String(), nil
}

// writeQuery writes a minified query for t to buf.
// If inline is true, the struct fields of t are inlined into parent struct.
//
// This is the main orchestration function that dispatches to specialized
// handlers based on the type kind.
func writeQuery(
	buf *bytes.Buffer,
	t reflect.Type,
	v reflect.Value,
	inline bool,
) error {
	switch t.Kind() {
	case reflect.Interface:
		return writeInterfaceQuery(buf, t, v, inline)
	case reflect.Ptr:
		err := writeQuery(buf, t.Elem(), reflectutil.ElemSafe(v), false)
		if err != nil {
			return fmt.Errorf("failed to write query for ptr `%v`: %w", t, err)
		}
	case reflect.Struct:
		return writeStructQuery(buf, t, v, inline)
	case reflect.Slice:
		return writeSliceQuery(buf, t, v)
	case reflect.Map:
		return fmt.Errorf("type %v is not supported, use [][2]any instead", t)
	}
	return nil
}
