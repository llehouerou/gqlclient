// Package types defines the GraphQL-related interfaces and reflection
// constants shared between the gqlclient packages.
//
// The two main interfaces, GraphQLType and GraphQLWrapper, let user types
// participate in query construction and response decoding:
//
//   - A type implementing GraphQLType returns the GraphQL type name to use
//     when the type appears as a query variable.
//   - A type implementing GraphQLWrapper exposes a wrapped value so that
//     decoding can transparently unmarshal directly into the wrapped field.
//
// The package-level reflect.Type values (GraphqlTypeInterface,
// GraphqlWrapperInterface) are precomputed for use with reflect.Type.Implements
// on hot paths.
package types
