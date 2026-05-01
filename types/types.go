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

// GraphQLWrapper marks types that wrap a GraphQL value. Implementations
// return the wrapped value via GetGraphQLWrapped, allowing the decoder to
// unmarshal JSON directly into the wrapped field rather than the wrapper.
//
// Implementations must also expose an exported field named "Value" holding
// the wrapped data, since unmarshaling needs a writable target.
type GraphQLWrapper interface {
	GetGraphQLWrapped() any
}

// GraphqlWrapperInterface is the reflect.Type of GraphQLWrapper, precomputed
// for use with reflect.Type.Implements on hot paths.
var GraphqlWrapperInterface = reflect.TypeOf((*GraphQLWrapper)(nil)).Elem()
