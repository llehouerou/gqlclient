package decode

import (
	"reflect"
	"strings"

	"github.com/llehouerou/gqlclient/internal/reflectutil"
	"github.com/llehouerou/gqlclient/internal/tagparser"
	"github.com/llehouerou/gqlclient/types"
)

// fieldByGraphQLName returns an exported struct field of struct v
// that matches GraphQL name, or invalid reflect.Value if none found.
func fieldByGraphQLName(
	v reflect.Value,
	name string,
) (val reflect.Value, taggedAsScalar bool) {
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).PkgPath != "" {
			// Skip unexported field.
			continue
		}
		if hasGraphQLName(v.Type().Field(i), v.Field(i), name) {
			return v.Field(i), hasScalarTag(v.Type().Field(i))
		}
	}
	return reflect.Value{}, false
}

// orderedMapValueByGraphQLName takes [][2]string, interprets it as an ordered map
// and returns value for corresponding key, or invalid reflect.Value if none found.
func orderedMapValueByGraphQLName(v reflect.Value, name string) reflect.Value {
	for i := 0; i < v.Len(); i++ {
		pair := v.Index(i)
		key := pair.Index(0).Interface().(string)
		if keyHasGraphQLName(key, name) {
			return pair.Index(1)
		}
	}
	return reflect.Value{}
}

// hasScalarTag reports whether struct field f has the scalar:"true" tag.
func hasScalarTag(f reflect.StructField) bool {
	return reflectutil.IsTrue(f.Tag.Get(types.ScalarTag))
}

// hasGraphQLName reports whether struct field f has GraphQL name.
func hasGraphQLName(f reflect.StructField, v reflect.Value, name string) bool {
	value := ""
	ok := false
	if reflectutil.ImplementsGraphQLType(f.Type) {
		typeName, typeok := reflectutil.GetGraphQLType(v, f.Type)
		if typeok {
			value = typeName
			ok = true
		}
	}
	if !ok {
		value, ok = f.Tag.Lookup(types.GraphQLTag)
	}
	if !ok {
		// Fall back to case-insensitive comparison when no graphql tag is present.
		// Using strings.EqualFold instead of caseconv.MixedCapsToLowerCamelCase
		// for better performance. This is slightly less precise (doesn't handle
		// camelCase conversion) but works well in practice since most GraphQL
		// schemas use standard naming conventions.
		return strings.EqualFold(f.Name, name)
	}
	return keyHasGraphQLName(value, name)
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
