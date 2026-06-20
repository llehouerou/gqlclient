// Package types defines the GraphQL-related interfaces, struct-tag names, and
// reflection constants shared between the gqlclient packages.
//
// User types participate in query construction and response decoding through:
//
//   - The GraphQLType interface: returns the GraphQL type name to use when the
//     type appears as a query variable.
//   - The WrappedTag (`wrapped:"true"`) struct tag: marks the single field of a
//     wrapper struct, so query construction and decoding transparently splice
//     that field in place of the wrapper.
//
// The package-level reflect.Type value GraphqlTypeInterface is precomputed for
// use with reflect.Type.Implements on hot paths.
package types
