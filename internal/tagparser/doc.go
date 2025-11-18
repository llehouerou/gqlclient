// Package tagparser provides parsing for GraphQL struct tags.
//
// This package handles the low-level parsing of `graphql:"..."` struct tag
// values, extracting structured information needed for both query construction
// and response unmarshaling.
//
// # Tag Syntax
//
// The parser supports the full GraphQL field syntax:
//
//	Simple field:
//	  `graphql:"name"`
//
//	Field with arguments:
//	  `graphql:"height(unit: METER)"`
//
//	Field with variables:
//	  `graphql:"human(id: $id)"`
//
//	Aliased field:
//	  `graphql:"node1: node(id: $id)"`
//
//	Inline fragment:
//	  `graphql:"... on Droid"`
//
//	Skip field:
//	  `graphql:"-"`
//
// # Parsed Structure
//
// The parser returns a ParsedTag struct containing:
//   - FieldName: GraphQL field name (after alias if present)
//   - Arguments: Content inside parentheses (e.g., "unit: METER")
//   - Alias: Field alias (before the colon) if any
//   - IsFragment: Whether this is a fragment ("...")
//   - TypeName: The typename for fragments ("... on TypeName")
//
// # Usage
//
// The tagparser is used by:
//   - Query construction: To build GraphQL query strings from structs
//   - Fragment detection: To identify inline fragments (see fragments package)
//   - Field matching: To map GraphQL response fields to struct fields
//
// Example:
//
//	tag := `graphql:"node1: node(id: $id)"`
//	value, _ := reflect.TypeOf(MyStruct{}).Field(0).Tag.Lookup("graphql")
//	parsed, _ := tagparser.ParseGraphQLTag(value)
//	// parsed.FieldName == "node"
//	// parsed.Alias == "node1"
//	// parsed.Arguments == "id: $id"
package tagparser
