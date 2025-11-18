package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/llehouerou/gqlclient/ident"
	"github.com/llehouerou/gqlclient/internal/reflectutil"
	"github.com/llehouerou/gqlclient/types"
)

// fieldOutput contains the processed information for a struct field
// used during GraphQL query construction
type fieldOutput struct {
	shouldSkip bool
	name       string
	isInline   bool
	value      reflect.Value
	isScalar   bool
}

// processStructField processes a single struct field and returns
// information needed for query construction
func processStructField(
	f reflect.StructField,
	fieldValue reflect.Value,
) fieldOutput {
	value := ""
	ok := false

	// Check if the field type implements GraphQLType
	if reflectutil.ImplementsGraphQLType(f.Type) {
		// Only skip nil pointers and nil interfaces (not nil slices/maps)
		kind := fieldValue.Kind()
		if !fieldValue.IsValid() ||
			((kind == reflect.Ptr || kind == reflect.Interface) &&
				fieldValue.IsNil()) {
			// Skip this field if it's a nil pointer or nil interface
			return fieldOutput{shouldSkip: true}
		}
		typeName, typeok := reflectutil.GetGraphQLType(fieldValue, f.Type)
		if typeok {
			value = typeName
			ok = true
		} else {
			// Skip this field if the concrete value is a nil pointer
			return fieldOutput{shouldSkip: true}
		}
	} else if f.Type.Kind() == reflect.Slice &&
		reflectutil.ImplementsGraphQLType(f.Type.Elem()) {
		// For slices, check if the element type implements GraphQLType
		typeName, typeok := reflectutil.GetGraphQLTypeFromType(f.Type.Elem())
		if typeok {
			value = typeName
			ok = true
		}
	}

	if !ok {
		value, ok = f.Tag.Lookup(types.GraphQLTag)
	}
	// Skip this field if the tag value is hyphen
	if value == "-" {
		return fieldOutput{shouldSkip: true}
	}

	inlineField := f.Anonymous && !ok
	var fieldName string
	if !inlineField {
		if ok {
			fieldName = value
		} else {
			fieldName = ident.ParseMixedCaps(f.Name).ToLowerCamelCase()
		}
	}

	isScalar := reflectutil.IsTrue(f.Tag.Get(types.ScalarTag))

	return fieldOutput{
		shouldSkip: false,
		name:       fieldName,
		isInline:   inlineField,
		value:      fieldValue,
		isScalar:   isScalar,
	}
}

// isScalarType checks if a type should be treated as a GraphQL scalar
// and not expanded during query construction.
// Returns true for types implementing json.Unmarshaler or ID type.
func isScalarType(t reflect.Type) bool {
	// If the type implements json.Unmarshaler, it's a scalar. Don't expand it.
	if reflect.PointerTo(t).Implements(jsonUnmarshaler) {
		return true
	}
	// ID type is also a scalar
	if t.AssignableTo(idType) {
		return true
	}
	return false
}

// writeStructFields iterates over struct fields and writes them to w.
// Returns an error if field processing fails.
func writeStructFields(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
) error {
	iter := 0
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fieldVal := reflectutil.FieldSafe(v, i)
		output := processStructField(f, fieldVal)

		// Skip this field if indicated by processStructField
		if output.shouldSkip {
			continue
		}

		if iter != 0 {
			_, _ = io.WriteString(w, ",")
		}
		iter++

		if !output.isInline {
			_, _ = io.WriteString(w, output.name)
		}
		// Skip writeQuery if the GraphQL type associated with the field is scalar
		if output.isScalar {
			continue
		}

		err := writeQuery(w, f.Type, fieldVal, output.isInline)
		if err != nil {
			return fmt.Errorf(
				"failed to write query for struct field `%v`: %w",
				f.Name,
				err,
			)
		}
	}
	return nil
}

// writeStructQuery writes a minified query for a struct type to w.
// If inline is true, the struct fields are inlined into parent struct.
func writeStructQuery(
	w io.Writer,
	t reflect.Type,
	v reflect.Value,
	inline bool,
) error {
	if v.IsValid() && reflectutil.IsWrapperType(v) {
		wrapped := reflectutil.UnwrapValue(v)
		if wrapped.IsValid() {
			err := writeQuery(
				w,
				wrapped.Type(),
				wrapped,
				inline,
			)
			if err != nil {
				return fmt.Errorf(
					"failed to write query for wrapped struct `%v`: %w",
					t,
					err,
				)
			}
			return nil
		}
	}

	// If the type is a scalar, don't expand it
	if isScalarType(t) {
		return nil
	}
	if !inline {
		_, _ = io.WriteString(w, "{")
	}

	err := writeStructFields(w, t, v)
	if err != nil {
		return err
	}

	if !inline {
		_, _ = io.WriteString(w, "}")
	}
	return nil
}

var jsonUnmarshaler = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
var idType = reflect.TypeOf(ID(""))
