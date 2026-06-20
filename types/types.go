package types

import "reflect"

// GraphQLType interface is used to specify the GraphQL type associated
// with a particular type. If a type implements this interface, the name of
// the variable used while creating the GraphQL query will be the output of
// the function defined below.
//
// In the current implementation, the GetGraphQLType function is applied to
// the zero value of the type to get the GraphQL type. So those who are
// implementing the function should avoid referencing the value of the type
// inside the function. Further, by this design, the output of the GetGraphQLType
// function will be a constant.
type GraphQLType interface {
	GetGraphQLType() string
}

// GraphqlTypeInterface is the reflect.Type of GraphQLType, precomputed so
// callers can perform reflect.Type.Implements checks on hot paths without
// allocating per call.
var GraphqlTypeInterface = reflect.TypeOf((*GraphQLType)(nil)).Elem()

// The wrapper pattern (a struct made transparent so its single wrapped field is
// spliced in place of the wrapper) is driven by the `wrapped:"true"` struct tag
// — see WrappedTag in constants.go — not by an interface. A tag is the single
// source of truth: it both marks the type as a wrapper and identifies the
// writable field, so detection and access can never disagree.
