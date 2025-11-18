// Package fragments provides utilities for detecting and working with
// GraphQL inline fragments in struct tags and ordered map keys.
//
// GraphQL inline fragments use the "... on TypeName" syntax to handle
// unions and interfaces. This package centralizes the logic for:
//   - Detecting whether a tag represents a fragment
//   - Extracting typename from fragment tags
//   - Supporting both struct field tags and ordered map keys
//
// All fragment detection is based on the tagparser package, which handles
// the low-level tag parsing.
package fragments

import (
	"reflect"

	"github.com/llehouerou/gqlclient/internal/tagparser"
	"github.com/llehouerou/gqlclient/types"
)

// IsStructField reports whether struct field f is a GraphQL inline fragment.
// It checks the graphql:"..." tag for the "... on TypeName" pattern.
//
// Example:
//
//	type Response struct {
//	    Droid `graphql:"... on Droid"`  // IsStructField returns true
//	}
func IsStructField(f reflect.StructField) bool {
	value, ok := f.Tag.Lookup(types.GraphQLTag)
	if !ok {
		return false
	}
	return IsTag(value)
}

// IsTag reports whether a tag value represents a GraphQL inline fragment.
// This works for both struct field tags and ordered map keys.
//
// Example:
//
//	IsTag("... on Droid")  // true
//	IsTag("name")          // false
func IsTag(tagValue string) bool {
	parsed, err := tagparser.ParseGraphQLTag(tagValue)
	if err != nil {
		return false
	}
	return parsed.IsFragment
}

// ExtractTypename extracts the typename from a GraphQL fragment tag.
// For example, "... on Droid" returns "Droid".
// Returns empty string if not a valid fragment tag or if there's no typename.
//
// Example:
//
//	ExtractTypename("... on Droid")  // "Droid"
//	ExtractTypename("...")           // ""
//	ExtractTypename("name")          // ""
func ExtractTypename(tagValue string) string {
	parsed, err := tagparser.ParseGraphQLTag(tagValue)
	if err != nil {
		return ""
	}
	if !parsed.IsFragment {
		return ""
	}
	return parsed.TypeName
}
