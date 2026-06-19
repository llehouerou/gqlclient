package fragments

import (
	"reflect"

	"github.com/llehouerou/gqlclient/internal/tagparser"
	"github.com/llehouerou/gqlclient/types"
)

// FromField reports whether struct field f carries a GraphQL inline-fragment
// tag (graphql:"... on TypeName") and, if so, its typename. It performs a
// single tag lookup and a single parse.
//
// ok is true exactly when f is a fragment. typename may be "" even when ok is
// true — that is a typename-less "..." fragment.
//
// Example:
//
//	type Response struct {
//	    Droid `graphql:"... on Droid"`  // FromField -> ("Droid", true)
//	}
func FromField(f reflect.StructField) (typename string, ok bool) {
	value, found := f.Tag.Lookup(types.GraphQLTag)
	if !found {
		return "", false
	}
	return FromTag(value)
}

// FromTag reports whether a tag value is a GraphQL inline fragment and, if so,
// its typename. It works for both struct-field tags and ordered-map keys, with
// a single parse.
//
// ok is true exactly when tagValue is a fragment. typename may be "" even when
// ok is true — that is a typename-less "..." fragment.
//
// Example:
//
//	FromTag("... on Droid")  // ("Droid", true)
//	FromTag("...")           // ("", true)
//	FromTag("name")          // ("", false)
func FromTag(tagValue string) (typename string, ok bool) {
	parsed, err := tagparser.ParseGraphQLTag(tagValue)
	if err != nil || !parsed.IsFragment {
		return "", false
	}
	return parsed.TypeName, true
}
