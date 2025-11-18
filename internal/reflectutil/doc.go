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
// Types implementing the wrapper pattern must:
//  1. Implement GetGraphQLWrapped() method returning the wrapped value
//  2. Have an exported field named "Value" holding the wrapped data
//
// The GetGraphQLWrapped() method is used during query construction (read-only),
// while the Value field is used during unmarshaling (needs to be writable).
//
// Example:
//
//	type Wrapper[T any] struct {
//	    Value T  // REQUIRED: Must be named "Value"
//	}
//	func (w Wrapper[T]) GetGraphQLWrapped() T { return w.Value }
//
// # Unwrapping Decision Tree
//
// Choose the right unwrap function based on your use case:
//
//	Need to follow pointer/interface indirection?
//	  -> Use UnwrapToConcreteValue()
//
//	Need to unwrap for query construction (calling methods)?
//	  -> Use UnwrapValue() (calls GetGraphQLWrapped method)
//
//	Need to unwrap for unmarshaling (writing data)?
//	  -> Use UnwrapValueField() (accesses Value field)
//
//	Not sure if it's a wrapper type?
//	  -> Use UnwrapValueOrOriginal() (safe fallback)
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
