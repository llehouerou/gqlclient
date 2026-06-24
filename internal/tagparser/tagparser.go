package tagparser

import "strings"

// ParsedTag represents a parsed GraphQL struct tag.
type ParsedTag struct {
	// FieldName is the GraphQL field name (after alias if present).
	FieldName string
	// Arguments contains the content inside parentheses, if any.
	Arguments string
	// Alias is the field alias (before the colon), if any.
	Alias string
	// Directives holds the field directives verbatim ("@include(if: $x)"),
	// including the leading "@", if any. The query builder emits them; the
	// decoder ignores them and matches on FieldName/Alias.
	Directives string
	// IsFragment indicates whether this is a GraphQL fragment ("...").
	IsFragment bool
	// TypeName is the typename for fragments ("... on TypeName").
	TypeName string
}

// ParseGraphQLTag parses a GraphQL struct tag value and returns structured information.
// Examples:
//   - "name" -> {FieldName: "name"}
//   - "height(unit: METER)" -> {FieldName: "height", Arguments: "unit: METER"}
//   - "node1: node(id: $id)" -> {FieldName: "node", Alias: "node1", Arguments: "id: $id"}
//   - "email @include(if: $x)" -> {FieldName: "email", Directives: "@include(if: $x)"}
//   - "... on Droid" -> {IsFragment: true, TypeName: "Droid"}
func ParseGraphQLTag(tag string) (ParsedTag, error) {
	tag = strings.TrimSpace(tag)

	var parsed ParsedTag

	// Handle empty string
	if tag == "" {
		return parsed, nil
	}

	// Handle skip field
	if tag == "-" {
		parsed.FieldName = "-"
		return parsed, nil
	}

	// Handle fragments
	if strings.HasPrefix(tag, "...") {
		parsed.IsFragment = true
		// Remove "..." prefix
		remaining := strings.TrimSpace(tag[3:])
		// Check for "on TypeName"
		if strings.HasPrefix(remaining, "on ") {
			parsed.TypeName = strings.TrimSpace(remaining[3:])
		}
		return parsed, nil
	}

	// Split off field directives before parsing the selection, so the
	// directive's own parentheses are never mistaken for field arguments.
	// A directive starts at the first "@" that sits outside any parentheses;
	// an "@" inside an argument value (e.g. handle: "@alice") is ignored.
	if dirIdx := directiveIndex(tag); dirIdx != -1 {
		parsed.Directives = strings.TrimSpace(tag[dirIdx:])
		tag = strings.TrimSpace(tag[:dirIdx])
	}

	// Find arguments first (content in parentheses)
	var fieldPart string
	parenIdx := strings.Index(tag, "(")
	if parenIdx != -1 {
		// Extract arguments
		closeIdx := strings.LastIndex(tag, ")")
		if closeIdx > parenIdx {
			parsed.Arguments = tag[parenIdx+1 : closeIdx]
		}
		fieldPart = strings.TrimSpace(tag[:parenIdx])
	} else {
		fieldPart = tag
	}

	// Handle alias in the field part (before arguments)
	if colonIdx := strings.Index(fieldPart, ":"); colonIdx != -1 {
		parsed.Alias = strings.TrimSpace(fieldPart[:colonIdx])
		parsed.FieldName = strings.TrimSpace(fieldPart[colonIdx+1:])
	} else {
		parsed.FieldName = strings.TrimSpace(fieldPart)
	}

	return parsed, nil
}

// directiveIndex returns the index of the first "@" that begins a field
// directive — that is, the first "@" found outside any parentheses — or -1 if
// the tag carries no directive. Tracking parenthesis depth keeps an "@" inside
// an argument value (e.g. handle: "@alice") from being treated as a directive.
func directiveIndex(tag string) int {
	depth := 0
	for i := range len(tag) {
		switch tag[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '@':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
