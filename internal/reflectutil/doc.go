// Package reflectutil provides reflection utilities for GraphQL type handling.
//
// This package centralizes reflection-based operations needed for both query
// construction and response unmarshaling. It handles two distinct unwrapping
// patterns:
//   - Pointer/interface unwrapping: Following indirection to concrete values
//   - Wrapper type unwrapping: GraphQL-specific container pattern
//
// # Package Organization
//
//   - unwrap.go: All value unwrapping logic and wrapper type support
//   - safe.go: Safe reflection accessors (Len, Index, Field, MapIndex, etc.)
//   - helpers.go: Type checking utilities (IsScalar, IsBasicType, etc.)
//
// # Wrapper Type Convention
//
// This package defines and enforces the wrapper type convention used
// throughout gqlclient:
//
// A wrapper is a struct with exactly one field tagged `wrapped:"true"` (see
// types.WrappedTag). That field is the single source of truth: it holds the
// wrapped value, supplies the wrapped type during query construction, and is the
// writable target during unmarshaling. No method is involved, so detection and
// access can never disagree.
//
// Example:
//
//	type Wrapper[T any] struct {
//	    Value T `wrapped:"true"` // the wrapped field; any name works
//	}
//
// # Unwrapping Decision Tree
//
// Choose the right unwrap function based on your use case:
//
//  1. GETTING PAST POINTER/INTERFACE INDIRECTION
//     Need to get to the concrete value beneath pointers or interfaces?
//     Example: **int -> int, or interface{} containing string -> string
//     -> Use UnwrapToConcreteValue()
//
//  2. UNWRAPPING A WRAPPER TYPE (query construction OR unmarshaling)
//     Need the wrapped field — its type for the selection set, or a writable
//     reference for decoding?
//     -> Use UnwrapValueField() (guard with IsWrapperType() when needed)
//
// IMPORTANT: These are orthogonal - you may need to unwrap pointers first,
// then unwrap wrapper types. See unwrap.go for detailed examples and scenarios.
//
// # Safe Reflection Accessors
//
// The safe.go utilities provide nil-safe wrappers around common reflection
// operations. They handle edge cases and invalid values gracefully:
//   - SafeLen: Returns length of arrays, slices, maps, strings, channels
//   - SafeIndex: Safely accesses slice/array elements by index
//   - SafeField: Safely accesses struct fields by index
//   - SafeMapIndex: Safely accesses map values by key
//
// # Type Checking Helpers
//
// The helpers.go utilities provide convenient type checking:
//   - IsScalar: Checks if a type should be treated as a GraphQL scalar
//   - IsBasicType: Checks if a type is a Go basic type (int, string, etc.)
//
// These helpers consider both Go's built-in types and GraphQL-specific
// patterns like json.Unmarshaler implementations.
package reflectutil
