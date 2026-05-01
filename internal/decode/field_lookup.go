package decode

import (
	"reflect"

	"github.com/llehouerou/gqlclient/internal/tagparser"
)

// fieldByGraphQLName returns an exported struct field of struct v
// that matches GraphQL name, or invalid reflect.Value if none found.
//
// Field metadata is precomputed per reflect.Type and cached, so this is
// O(N_fields) string compares in the hot path (no reflection or tag
// parsing per call). See field_cache.go.
func fieldByGraphQLName(
	v reflect.Value,
	name string,
) (val reflect.Value, taggedAsScalar bool) {
	tbl := lookupFieldTable(v.Type())
	if i, scalar, ok := tbl.lookup(v, name); ok {
		return v.Field(i), scalar
	}
	return reflect.Value{}, false
}

// orderedMapValueByGraphQLName takes [][2]string, interprets it as an ordered map
// and returns value for corresponding key, or invalid reflect.Value if none found.
func orderedMapValueByGraphQLName(v reflect.Value, name string) reflect.Value {
	for i := range v.Len() {
		pair := v.Index(i)
		key := pair.Index(0).Interface().(string)
		if keyHasGraphQLName(key, name) {
			return pair.Index(1)
		}
	}
	return reflect.Value{}
}

// keyHasGraphQLName reports whether a tag value matches a GraphQL field name.
func keyHasGraphQLName(value, name string) bool {
	parsed, err := tagparser.ParseGraphQLTag(value)
	if err != nil {
		return false
	}
	if parsed.IsFragment {
		// GraphQL fragment. It doesn't have a name.
		return false
	}
	// When there's an alias, the response uses the alias name.
	// Otherwise, it uses the field name.
	if parsed.Alias != "" {
		return parsed.Alias == name
	}
	return parsed.FieldName == name
}
