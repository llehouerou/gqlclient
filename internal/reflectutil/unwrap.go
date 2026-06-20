package reflectutil

import (
	"reflect"
	"sync"

	"github.com/llehouerou/gqlclient/types"
)

// Unwrap.go consolidates all value unwrapping logic in one place.
//
// There are two distinct unwrapping patterns in this codebase:
//
// 1. POINTER/INTERFACE UNWRAPPING: Following indirection to get concrete values
//    - UnwrapToConcreteValue() - recursively unwraps pointers/interfaces
//
// 2. WRAPPER TYPE UNWRAPPING: GraphQL-specific wrapper pattern
//    - IsWrapperType()     - reports whether a value is a wrapper struct
//    - UnwrapValueField()  - returns the wrapped field (query AND unmarshaling)
//
// WRAPPER TYPE CONVENTION:
// A wrapper is a struct with exactly one field tagged `wrapped:"true"`. That
// field is the single source of truth: it holds the wrapped value, supplies the
// wrapped type during query construction, and is the writable target during
// unmarshaling. No method is involved — detection and access both ride on the
// tag, so there is nothing to keep in sync.
//
// CHOOSING THE RIGHT UNWRAP FUNCTION - Decision Tree:
//
// 1. GETTING PAST POINTER/INTERFACE INDIRECTION
//    Need to get to the concrete value beneath pointers or interfaces?
//    Example: **int -> int, or interface{} containing string -> string
//    -> Use UnwrapToConcreteValue()
//
// 2. UNWRAPPING A GRAPHQL WRAPPER TYPE
//    Need the wrapped field (its type for query construction, or a writable
//    reference for unmarshaling)?
//    -> Use UnwrapValueField() (guard with IsWrapperType() when needed)
//
// IMPORTANT: These functions are orthogonal:
// - UnwrapToConcreteValue() handles language-level indirection (pointers/interfaces)
// - IsWrapperType/UnwrapValueField() handle GraphQL wrapper types
// - You may need to use BOTH in sequence: first unwrap pointers, then unwrap wrapper

// ============================================================================
// POINTER/INTERFACE UNWRAPPING
// ============================================================================

// UnwrapToConcreteValue unwraps pointers and interfaces to get to the concrete
// value. This is a common pattern used throughout the codebase to get past
// indirection layers.
//
// Returns the concrete value, or an invalid reflect.Value if unwrapping fails
// (e.g., encounters a nil pointer or interface).
//
// Example:
//
//	var x **int
//	val := 42
//	ptr := &val
//	x = &ptr
//	v := reflect.ValueOf(x)
//	concrete := UnwrapToConcreteValue(v) // returns reflect.Value of 42
func UnwrapToConcreteValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// ============================================================================
// WRAPPER TYPE UNWRAPPING
// ============================================================================

// wrappedFieldIndexCache memoizes, per reflect.Type, the index of the field
// tagged `wrapped:"true"` (or -1 when the type is not a wrapper). The answer is
// constant per type, so caching turns the field scan into a sync.Map load on
// the hot path. It also unifies detection and access: both IsWrapperType and
// UnwrapValueField read the same cached index.
var wrappedFieldIndexCache sync.Map // map[reflect.Type]int

// wrappedFieldIndex returns the index of t's `wrapped:"true"` field, or -1 if t
// is not a struct or has no such field. The first matching field wins.
func wrappedFieldIndex(t reflect.Type) int {
	if cached, ok := wrappedFieldIndexCache.Load(t); ok {
		return cached.(int) //nolint:errcheck // wrappedFieldIndexCache only ever stores int
	}
	idx := -1
	if t.Kind() == reflect.Struct {
		for i := range t.NumField() {
			if IsTrue(t.Field(i).Tag.Get(types.WrappedTag)) {
				idx = i
				break
			}
		}
	}
	wrappedFieldIndexCache.Store(t, idx)
	return idx
}

// IsWrapperType reports whether the given reflect.Value is a wrapper type.
// A wrapper type is a struct with a field tagged `wrapped:"true"`.
// Returns false if the value is invalid or nil.
func IsWrapperType(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	// Unwrap pointers and interfaces to check the concrete type
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return false
	}

	return wrappedFieldIndex(v.Type()) >= 0
}

// UnwrapValueField returns the field tagged `wrapped:"true"` of a wrapper type.
// It is used both for query construction (where the field's type drives the
// selection set) and for unmarshaling (where the returned field is the writable
// decode target — callers pass an addressable value to get a settable field).
//
// If the value is not a wrapper type (or is a nil pointer/interface), returns
// an invalid reflect.Value.
//
// Example:
//
//	type MyWrapper struct {
//	    Value string `wrapped:"true"`
//	}
//
//	w := MyWrapper{}
//	v := reflect.ValueOf(&w).Elem() // need addressable value
//	field := UnwrapValueField(v) // returns writable reference to Value field
//	field.SetString("hello") // now w.Value == "hello"
func UnwrapValueField(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return reflect.Value{}
	}

	// Unwrap pointers and interfaces to get to the struct
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}

	if !v.IsValid() || v.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	idx := wrappedFieldIndex(v.Type())
	if idx < 0 {
		return reflect.Value{}
	}

	return v.Field(idx)
}

// ============================================================================
// GRAPHQL TYPE HELPERS
// ============================================================================

// implementsGraphQLTypeCache memoizes ImplementsGraphQLType results per
// reflect.Type. The answer is constant for a given type, so caching
// turns a reflect.(*rtype).Implements call into a sync.Map load on the
// hot path.
var implementsGraphQLTypeCache sync.Map // map[reflect.Type]bool

// ImplementsGraphQLType reports whether the given type implements the
// GraphQLType interface. This checks if the type provides a custom GraphQL
// type name via GetGraphQLType().
func ImplementsGraphQLType(t reflect.Type) bool {
	if cached, ok := implementsGraphQLTypeCache.Load(t); ok {
		return cached.(bool) //nolint:errcheck // implementsGraphQLTypeCache only ever stores bool
	}
	result := t.Implements(types.GraphqlTypeInterface)
	implementsGraphQLTypeCache.Store(t, result)
	return result
}

// GetGraphQLType extracts the GraphQL type name from a value that implements
// GraphQLType interface. Returns empty string if the value doesn't implement
// GraphQLType or if extraction fails.
func GetGraphQLType(v reflect.Value, t reflect.Type) (string, bool) {
	if !ImplementsGraphQLType(t) {
		return "", false
	}

	// Handle nil pointers and interfaces
	if !v.IsValid() {
		return "", false
	}

	kind := v.Kind()
	if (kind == reflect.Ptr || kind == reflect.Interface) && v.IsNil() {
		return "", false
	}

	// Try to get the GraphQLType from the value
	graphqlType, ok := v.Interface().(types.GraphQLType)
	if !ok {
		return "", false
	}

	// Additional check: if the interface contains a nil pointer, reject it
	graphqlTypeVal := reflect.ValueOf(graphqlType)
	if graphqlTypeVal.IsValid() &&
		graphqlTypeVal.Kind() == reflect.Ptr &&
		graphqlTypeVal.IsNil() {
		return "", false
	}

	return graphqlType.GetGraphQLType(), true
}

// GetGraphQLTypeFromType extracts the GraphQL type name from a type (not
// value). This creates a zero value or pointer to call GetGraphQLType().
// Useful when you don't have an instance but need the type name.
func GetGraphQLTypeFromType(t reflect.Type) (string, bool) {
	if !ImplementsGraphQLType(t) {
		return "", false
	}

	var graphqlType types.GraphQLType
	var ok bool

	if t.Kind() == reflect.Ptr {
		graphqlType, ok = reflect.New(t.Elem()).Interface().(types.GraphQLType)
	} else {
		graphqlType, ok = reflect.Zero(t).Interface().(types.GraphQLType)
	}

	if !ok {
		return "", false
	}

	return graphqlType.GetGraphQLType(), true
}
