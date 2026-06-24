package tagparser

import (
	"testing"
)

func TestParseGraphQLTag_SimpleFieldName(t *testing.T) {
	t.Parallel()

	tag := "name"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "name" {
		t.Errorf("expected FieldName 'name', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "" {
		t.Errorf("expected empty Arguments, got '%s'", parsed.Arguments)
	}
	if parsed.Alias != "" {
		t.Errorf("expected empty Alias, got '%s'", parsed.Alias)
	}
	if parsed.IsFragment {
		t.Error("expected IsFragment to be false")
	}
}

func TestParseGraphQLTag_FieldWithArguments(t *testing.T) {
	t.Parallel()

	tag := "height(unit: METER)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "height" {
		t.Errorf("expected FieldName 'height', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "unit: METER" {
		t.Errorf("expected Arguments 'unit: METER', got '%s'", parsed.Arguments)
	}
	if parsed.Alias != "" {
		t.Errorf("expected empty Alias, got '%s'", parsed.Alias)
	}
}

func TestParseGraphQLTag_Alias(t *testing.T) {
	t.Parallel()

	tag := "node1: node"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "node" {
		t.Errorf("expected FieldName 'node', got '%s'", parsed.FieldName)
	}
	if parsed.Alias != "node1" {
		t.Errorf("expected Alias 'node1', got '%s'", parsed.Alias)
	}
	if parsed.Arguments != "" {
		t.Errorf("expected empty Arguments, got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_AliasWithArguments(t *testing.T) {
	t.Parallel()

	tag := `node1: node(id: "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng==")`

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "node" {
		t.Errorf("expected FieldName 'node', got '%s'", parsed.FieldName)
	}
	if parsed.Alias != "node1" {
		t.Errorf("expected Alias 'node1', got '%s'", parsed.Alias)
	}
	expectedArgs := `id: "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng=="`
	if parsed.Arguments != expectedArgs {
		t.Errorf(
			"expected Arguments '%s', got '%s'",
			expectedArgs,
			parsed.Arguments,
		)
	}
}

func TestParseGraphQLTag_Fragment(t *testing.T) {
	t.Parallel()

	tag := "... on Droid"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "Droid" {
		t.Errorf("expected TypeName 'Droid', got '%s'", parsed.TypeName)
	}
}

func TestParseGraphQLTag_FragmentNoTypename(t *testing.T) {
	t.Parallel()

	tag := "..."

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "" {
		t.Errorf("expected empty TypeName, got '%s'", parsed.TypeName)
	}
}

func TestParseGraphQLTag_SkipField(t *testing.T) {
	t.Parallel()

	tag := "-"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "-" {
		t.Errorf("expected FieldName '-', got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_WithWhitespace(t *testing.T) {
	t.Parallel()

	tag := "  height(unit: METER)  "

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "height" {
		t.Errorf("expected FieldName 'height', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "unit: METER" {
		t.Errorf("expected Arguments 'unit: METER', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_EmptyString(t *testing.T) {
	t.Parallel()

	tag := ""

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "" {
		t.Errorf("expected empty FieldName, got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_VariableInArguments(t *testing.T) {
	t.Parallel()

	tag := "human(id: $id)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "human" {
		t.Errorf("expected FieldName 'human', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "id: $id" {
		t.Errorf("expected Arguments 'id: $id', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_ComplexRealWorldExample(t *testing.T) {
	t.Parallel()

	tag := `node1: node(id: "MDEyOklzc3VlQ29tbWVudDE2OTQwNzk0Ng==")`

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Alias != "node1" {
		t.Errorf("expected Alias 'node1', got '%s'", parsed.Alias)
	}
	if parsed.FieldName != "node" {
		t.Errorf("expected FieldName 'node', got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_MultipleColonsInArguments(t *testing.T) {
	t.Parallel()

	tag := "field(a: 1, b: 2, c: 3)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "a: 1, b: 2, c: 3" {
		t.Errorf(
			"expected Arguments 'a: 1, b: 2, c: 3', got '%s'",
			parsed.Arguments,
		)
	}
}

func TestParseGraphQLTag_FragmentWithExtraWhitespace(t *testing.T) {
	t.Parallel()

	tag := "  ...   on   Droid  "

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "Droid" {
		t.Errorf("expected TypeName 'Droid', got '%s'", parsed.TypeName)
	}
}

func TestParseGraphQLTag_NoArguments(t *testing.T) {
	t.Parallel()

	tag := "field()"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "" {
		t.Errorf("expected empty Arguments, got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_UnbalancedParentheses(t *testing.T) {
	t.Parallel()

	// Missing closing paren - should handle gracefully
	tag := "field(arg: value"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still extract field name
	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
}

func TestParseGraphQLTag_NestedParentheses(t *testing.T) {
	t.Parallel()

	tag := "field(arg: func(nested))"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "field" {
		t.Errorf("expected FieldName 'field', got '%s'", parsed.FieldName)
	}
	// Should capture everything inside outer parentheses
	if parsed.Arguments != "arg: func(nested)" {
		t.Errorf(
			"expected Arguments 'arg: func(nested)', got '%s'",
			parsed.Arguments,
		)
	}
}

func TestParseGraphQLTag_AliasWithColonInFieldName(t *testing.T) {
	t.Parallel()

	// Field name can contain colons (only first colon is alias separator)
	tag := "alias: http://example.com"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Alias != "alias" {
		t.Errorf("expected Alias 'alias', got '%s'", parsed.Alias)
	}
	// The rest after the FIRST colon is the field name
	// (including subsequent colons which are part of the field name)
	if parsed.FieldName != "http://example.com" {
		t.Errorf(
			"expected FieldName 'http://example.com', got '%s'",
			parsed.FieldName,
		)
	}
}

func TestParseGraphQLTag_AliasWithArgumentsRealWorld(t *testing.T) {
	t.Parallel()

	// Real-world pattern from sorare: alias with arguments
	tag := "shortDisplayName:displayName(short:true)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Alias != "shortDisplayName" {
		t.Errorf("expected Alias 'shortDisplayName', got '%s'", parsed.Alias)
	}
	if parsed.FieldName != "displayName" {
		t.Errorf("expected FieldName 'displayName', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "short:true" {
		t.Errorf("expected Arguments 'short:true', got '%s'", parsed.Arguments)
	}
}

func TestParseGraphQLTag_EscapedQuotesInArguments(t *testing.T) {
	t.Parallel()

	// Real-world pattern from sorare: escaped quotes inside arguments
	tag := `videoUrl(derivative:\"low_res\")`

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "videoUrl" {
		t.Errorf("expected FieldName 'videoUrl', got '%s'", parsed.FieldName)
	}
	expectedArgs := `derivative:\"low_res\"`
	if parsed.Arguments != expectedArgs {
		t.Errorf(
			"expected Arguments '%s', got '%s'",
			expectedArgs,
			parsed.Arguments,
		)
	}
}

func TestParseGraphQLTag_DirectiveNoArguments(t *testing.T) {
	t.Parallel()

	// A field directive with no field arguments: the directive's parentheses
	// must not be mistaken for field arguments.
	tag := "email @include(if: $withEmail)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "email" {
		t.Errorf("expected FieldName 'email', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "" {
		t.Errorf("expected empty Arguments, got '%s'", parsed.Arguments)
	}
	if parsed.Directives != "@include(if: $withEmail)" {
		t.Errorf(
			"expected Directives '@include(if: $withEmail)', got '%s'",
			parsed.Directives,
		)
	}
}

func TestParseGraphQLTag_DirectiveWithFieldArguments(t *testing.T) {
	t.Parallel()

	tag := "posts(first: 10) @skip(if: $hidePosts)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "posts" {
		t.Errorf("expected FieldName 'posts', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != "first: 10" {
		t.Errorf("expected Arguments 'first: 10', got '%s'", parsed.Arguments)
	}
	if parsed.Directives != "@skip(if: $hidePosts)" {
		t.Errorf(
			"expected Directives '@skip(if: $hidePosts)', got '%s'",
			parsed.Directives,
		)
	}
}

func TestParseGraphQLTag_AliasWithDirective(t *testing.T) {
	t.Parallel()

	tag := "e: email @include(if: $x)"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Alias != "e" {
		t.Errorf("expected Alias 'e', got '%s'", parsed.Alias)
	}
	if parsed.FieldName != "email" {
		t.Errorf("expected FieldName 'email', got '%s'", parsed.FieldName)
	}
	if parsed.Directives != "@include(if: $x)" {
		t.Errorf("expected Directives '@include(if: $x)', got '%s'", parsed.Directives)
	}
}

func TestParseGraphQLTag_DirectiveWithAtInArgumentValue(t *testing.T) {
	t.Parallel()

	// An '@' inside an argument value must not be taken for the directive
	// boundary; only an '@' outside parentheses starts a directive.
	tag := `user(handle: "@alice") @include(if: $x)`

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "user" {
		t.Errorf("expected FieldName 'user', got '%s'", parsed.FieldName)
	}
	if parsed.Arguments != `handle: "@alice"` {
		t.Errorf("expected Arguments 'handle: \"@alice\"', got '%s'", parsed.Arguments)
	}
	if parsed.Directives != "@include(if: $x)" {
		t.Errorf("expected Directives '@include(if: $x)', got '%s'", parsed.Directives)
	}
}

func TestParseGraphQLTag_DirectiveWithoutParentheses(t *testing.T) {
	t.Parallel()

	tag := "name @deprecated"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FieldName != "name" {
		t.Errorf("expected FieldName 'name', got '%s'", parsed.FieldName)
	}
	if parsed.Directives != "@deprecated" {
		t.Errorf("expected Directives '@deprecated', got '%s'", parsed.Directives)
	}
}

func TestParseGraphQLTag_LongFragmentTypename(t *testing.T) {
	t.Parallel()

	// Real-world pattern from sorare: long typename in fragment
	tag := "... on SolanaTokenTransferAuthorizationRequest"

	parsed, err := ParseGraphQLTag(tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !parsed.IsFragment {
		t.Error("expected IsFragment to be true")
	}
	if parsed.TypeName != "SolanaTokenTransferAuthorizationRequest" {
		t.Errorf(
			"expected TypeName 'SolanaTokenTransferAuthorizationRequest', got '%s'",
			parsed.TypeName,
		)
	}
}
