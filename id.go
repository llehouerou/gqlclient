package graphql

import (
	"fmt"
	"reflect"
)

// ID represents a unique identifier that is Base64 obfuscated. It
// is often used to refetch an object or as key for a cache. The ID
// type appears in a JSON response as a String; however, it is not
// intended to be human-readable. When expected as an input type,
// any string (such as "VXNlci0xMA==") or integer (such as 4) input
// value will be accepted as an ID.
//
// Unlike the deprecated scalar types (Boolean, Float, Int, String),
// ID remains actively supported because GraphQL's ID type has special
// semantics that differ from plain strings. Use native Go types
// (bool, float64, int32, string) for other scalar fields.
type ID string

// NewID is a helper to make a new *ID.
//
// It accepts any integer type or string and converts it to an ID pointer.
// Integer values of 0 are converted to empty string.
//
// Example:
//
//	id := graphql.NewID("user-123")
//	id := graphql.NewID(42)
func NewID(v any) *ID {
	rv := ToID(v)
	return &rv
}

// ToID is a helper for converting integers or strings to ID values.
//
// It converts integer types (int, int8, int16, int32, int64, uint, uint8,
// uint16, uint32, uint64) to their string representation. Integer values
// of 0 are converted to empty string. String values are passed through
// unchanged.
//
// Example:
//
//	id := graphql.ToID("user-123")  // ID("user-123")
//	id := graphql.ToID(42)          // ID("42")
//	id := graphql.ToID(0)           // ID("")
func ToID(v any) ID {
	var s string
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s = fmt.Sprintf("%d", v)
		if s == "0" {
			s = ""
		}
	case reflect.String:
		s = v.(string)
	}
	return ID(s)
}
