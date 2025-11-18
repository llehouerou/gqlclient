// Package decode provides GraphQL-specific JSON unmarshaling.
//
// The decoder handles GraphQL-specific patterns that standard JSON
// unmarshaling doesn't support:
//   - Inline fragments with __typename discrimination
//   - Template-based array unmarshaling
//   - Ordered maps ([][2]interface{})
//   - Wrapper types with GetGraphQLWrapped()
//
// # Architecture
//
// The package is organized into focused modules:
//   - graphql.go: Main UnmarshalGraphQL entry point and decoder struct
//   - decode_object.go: Object key/field processing, object start/end handling
//   - decode_array.go: Array value processing, template copying
//   - field_lookup.go: GraphQL field name matching
//   - value_stack.go: Fragment type tracking during unmarshaling
//
// # Inline Fragment Handling
//
// GraphQL unions and interfaces are represented using inline fragments with
// __typename discrimination. During unmarshaling, the decoder:
//  1. Captures the __typename value from the response
//  2. Filters inline fragments to populate only the matching type
//  3. Supports both struct fields and ordered map keys as fragments
//
// Example:
//
//	type SearchResult struct {
//	    User  User  `graphql:"... on User"`
//	    Repo  Repo  `graphql:"... on Repository"`
//	}
//
// When __typename is "User", only the User field is populated.
//
// # Template-Based Arrays
//
// When unmarshaling arrays, the first element in the target slice acts as
// a template that gets copied for each array item. This enables proper
// initialization of complex nested structures.
//
// Example:
//
//	// Target slice with one template element
//	result := []struct{ Name string }{{ Name: "template" }}
//	// After unmarshaling: template is copied for each array element
//
// # Ordered Maps
//
// GraphQL requires fields in specific order for mutations. The decoder
// supports ordered maps represented as [][2]interface{}:
//
//	m := [][2]interface{}{
//	    {"createUser(login: $login1)", &CreateUser{}},
//	    {"createUser(login: $login2)", &CreateUser{}},
//	}
//
// # Wrapper Types
//
// The decoder transparently unwraps container types that implement
// GetGraphQLWrapped(). During unmarshaling, JSON data is written directly
// into the wrapper's Value field.
//
// Convention: Wrapper types must have an exported field named "Value".
//
// Example:
//
//	type Wrapper[T any] struct {
//	    Value T  // REQUIRED: Must be named "Value"
//	}
//	func (w Wrapper[T]) GetGraphQLWrapped() T { return w.Value }
package decode
