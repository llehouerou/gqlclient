// Package fragments provides utilities for detecting and working with
// GraphQL inline fragments.
//
// GraphQL inline fragments use the "... on TypeName" syntax to handle
// unions and interfaces. This package centralizes the logic for:
//   - Detecting whether a tag represents a fragment
//   - Extracting typename from fragment tags
//   - Supporting both struct field tags and ordered map keys
//
// # Fragment Syntax
//
// Inline fragments allow querying union and interface types:
//
//	Basic fragment:
//	  `graphql:"... on Droid"`
//
//	Anonymous fragment (rare):
//	  `graphql:"..."`
//
// # Usage Example
//
// Struct with inline fragments for a union type:
//
//	type SearchResult struct {
//	    User  User  `graphql:"... on User"`
//	    Repo  Repo  `graphql:"... on Repository"`
//	}
//
// The decoder uses this package to:
//  1. Detect which fields are fragments and read their typename in one call
//     (FromField, FromTag)
//  2. Match that typename against the __typename in the response
//  3. Populate only the matching fragment field
//
// # Ordered Map Support
//
// Fragments can also appear in ordered map keys:
//
//	m := [][2]interface{}{
//	    {"... on User", &User{}},
//	    {"... on Repository", &Repository{}},
//	}
//
// # Architecture
//
// This package acts as a focused facade over the tagparser package.
// It provides two functions, each doing a single parse and returning the
// typename alongside the match:
//   - FromField(f reflect.StructField) (typename string, ok bool)
//   - FromTag(tagValue string) (typename string, ok bool)
//
// All fragment detection is based on the tagparser package, which handles
// the low-level tag parsing. This package adds GraphQL-specific semantics
// for fragment handling.
package fragments
