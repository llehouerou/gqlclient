package reflectutil

import (
	"reflect"

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
//    - UnwrapValue() - unwraps via GetGraphQLWrapped() method (query construction)
//    - UnwrapValueField() - unwraps via Value field (unmarshaling)
//    - UnwrapValueOrOriginal() - unwraps or returns original
//
// WRAPPER TYPE CONVENTION:
// Types implementing the wrapper pattern must:
// - Implement GetGraphQLWrapped() method that returns the wrapped value
// - Have an exported field named "Value" that holds the wrapped data
// - The "Value" field is used during unmarshaling (needs to be writable)
// - The GetGraphQLWrapped() method is used during query construction
//
// CHOOSING THE RIGHT UNWRAP FUNCTION - Decision Tree:
//
// 1. GETTING PAST POINTER/INTERFACE INDIRECTION
//    Need to get to the concrete value beneath pointers or interfaces?
//    Example: **int -> int, or interface{} containing string -> string
//    -> Use UnwrapToConcreteValue()
//
//    Common scenarios:
//    - Examining the actual type of a value for reflection operations
//    - Accessing struct fields through pointer indirection
//    - Getting the element type from a slice or array
//
// 2. UNWRAPPING GRAPHQL WRAPPER TYPES FOR QUERY CONSTRUCTION
//    Building a GraphQL query and need the wrapped value via method call?
//    Example: MyWrapper{Value: "hello"}.GetGraphQLWrapped() -> "hello"
//    -> Use UnwrapValue()
//
//    Common scenarios:
//    - Query construction where you need the actual GraphQL value
//    - Reading wrapper types that implement GetGraphQLWrapped()
//
// 3. UNWRAPPING GRAPHQL WRAPPER TYPES FOR UNMARSHALING
//    Unmarshaling JSON into a wrapper type and need writable field access?
//    Example: Need to set MyWrapper.Value field during JSON decode
//    -> Use UnwrapValueField()
//
//    Common scenarios:
//    - JSON unmarshaling into wrapper types
//    - Setting values in wrapper type fields during decoding
//    - Any operation requiring write access to the wrapped value
//
// 4. SAFE UNWRAPPING WITH FALLBACK
//    Not sure if the value is a wrapper type? Want original if not?
//    Example: Try to unwrap, but use the value as-is if it's not a wrapper
//    -> Use UnwrapValueOrOriginal()
//
//    Common scenarios:
//    - Defensive unwrapping where the type may or may not be a wrapper
//    - Generic code that handles both wrapper and non-wrapper types
//
// IMPORTANT: These functions are orthogonal:
// - UnwrapToConcreteValue() handles language-level indirection (pointers/interfaces)
// - UnwrapValue/UnwrapValueField/UnwrapValueOrOriginal() handle GraphQL wrapper types
// - You may need to use BOTH in sequence: first unwrap pointers, then unwrap wrapper

const (
	// WrapperMethodName is the name of the method that unwraps container types.
	// Types implementing this method should follow the wrapper convention:
	// they must have an exported field named "Value" that holds the wrapped data.
	WrapperMethodName = "GetGraphQLWrapped"

	// WrapperFieldName is the required name of the field holding wrapped data
	// in types that implement the wrapper pattern (GetGraphQLWrapped method).
	WrapperFieldName = "Value"
)

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

// IsWrapperType reports whether the given reflect.Value is a wrapper type.
// A wrapper type is one that implements the GetGraphQLWrapped() method.
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

	method := v.MethodByName(WrapperMethodName)
	return method.IsValid()
}

// UnwrapValue unwraps a wrapper type by calling its GetGraphQLWrapped() method.
// If the value is not a wrapper type, returns an invalid reflect.Value.
// If the value is a wrapper type, calls GetGraphQLWrapped() and returns the
// result.
//
// Note: This is used for query construction. For unmarshaling, the wrapped data
// must be stored in a field named "Value" per the wrapper convention - use
// UnwrapValueField() instead.
//
// Example:
//
//	type MyWrapper struct {
//	    Value string
//	}
//	func (w MyWrapper) GetGraphQLWrapped() interface{} {
//	    return w.Value
//	}
//
//	w := MyWrapper{Value: "hello"}
//	v := reflect.ValueOf(w)
//	unwrapped := UnwrapValue(v) // returns reflect.Value of "hello"
func UnwrapValue(v reflect.Value) reflect.Value {
	if !IsWrapperType(v) {
		return reflect.Value{}
	}

	// Unwrap pointers and interfaces to get to the struct
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	if !v.IsValid() {
		return reflect.Value{}
	}

	// Call GetGraphQLWrapped() method
	method := v.MethodByName(WrapperMethodName)
	if !method.IsValid() {
		return reflect.Value{}
	}

	// Call the method with no arguments and get the first return value
	results := method.Call(nil)
	if len(results) == 0 {
		return reflect.Value{}
	}

	return results[0]
}

// UnwrapValueField unwraps a wrapper type by accessing its Value field.
// This is specifically for unmarshaling where we need a writable field
// reference.
//
// If the value is not a wrapper type or doesn't have a Value field, returns
// an invalid reflect.Value.
//
// Convention: Wrapper types MUST have an exported field named "Value" for
// unmarshaling.
//
// Example:
//
//	type MyWrapper struct {
//	    Value string
//	}
//	func (w MyWrapper) GetGraphQLWrapped() interface{} {
//	    return w.Value
//	}
//
//	w := MyWrapper{}
//	v := reflect.ValueOf(&w).Elem() // need addressable value
//	field := UnwrapValueField(v) // returns writable reference to Value field
//	field.SetString("hello") // now w.Value == "hello"
func UnwrapValueField(v reflect.Value) reflect.Value {
	if !IsWrapperType(v) {
		return reflect.Value{}
	}

	// Unwrap pointers and interfaces to get to the struct
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	if !v.IsValid() {
		return reflect.Value{}
	}

	// Access the Value field per wrapper convention
	valueField := v.FieldByName(WrapperFieldName)
	if !valueField.IsValid() {
		return reflect.Value{}
	}

	return valueField
}

// UnwrapValueOrOriginal unwraps a wrapper type if possible, otherwise returns
// the original value. This is a convenience function that always returns a
// valid value for further processing.
//
// Example:
//
//	type MyWrapper struct {
//	    Value string
//	}
//	func (w MyWrapper) GetGraphQLWrapped() interface{} {
//	    return w.Value
//	}
//
//	// With wrapper type
//	w := MyWrapper{Value: "hello"}
//	v := reflect.ValueOf(w)
//	result := UnwrapValueOrOriginal(v) // returns reflect.Value of "hello"
//
//	// With non-wrapper type
//	s := "world"
//	v := reflect.ValueOf(s)
//	result := UnwrapValueOrOriginal(v) // returns reflect.Value of "world"
func UnwrapValueOrOriginal(v reflect.Value) reflect.Value {
	unwrapped := UnwrapValue(v)
	if unwrapped.IsValid() {
		return unwrapped
	}
	return v
}

// ============================================================================
// GRAPHQL TYPE HELPERS
// ============================================================================

// ImplementsGraphQLType reports whether the given type implements the
// GraphQLType interface. This checks if the type provides a custom GraphQL
// type name via GetGraphQLType().
func ImplementsGraphQLType(t reflect.Type) bool {
	return t.Implements(types.GraphqlTypeInterface)
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
