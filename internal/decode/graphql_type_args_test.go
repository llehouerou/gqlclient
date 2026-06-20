package decode_test

import (
	"testing"

	"github.com/llehouerou/gqlclient/internal/decode"
)

// Regression test for v0.15.0 -> v0.15.1.
//
// When a struct field's TYPE implements types.GraphQLType and
// GetGraphQLType returns a parameterized expression like
// "card(slug:$slug)", the JSON response key is just "card" — the
// arguments are not part of the response shape. The v0.14 decoder
// fed the GraphQLType-derived string through the tag parser, which
// extracted FieldName="card" before comparing. v0.15.0's cached
// lookup compared liveName == name directly, missing the match.
//
// This test pins the v0.14 contract and must pass.

type slugCardWrapper[T any] struct {
	Value T `wrapped:"true"`
}

func (slugCardWrapper[T]) GetGraphQLType() string {
	return "card(slug:$slug)"
}

type slugCardInner struct {
	Slug string `graphql:"slug"`
}

func TestUnmarshalGraphQL_GraphQLTypeWithArgsMatchesBareKey(t *testing.T) {
	t.Parallel()

	type payload struct {
		Card slugCardWrapper[slugCardInner]
	}
	var got payload
	err := decode.UnmarshalGraphQL(
		[]byte(`{"card":{"slug":"abc"}}`), &got,
	)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Card.Value.Slug != "abc" {
		t.Errorf("got slug %q, want %q", got.Card.Value.Slug, "abc")
	}
}

// Aliased GraphQLType return: "x: card(slug:$slug)" should match
// the response key "x" (the alias), not "card".
type aliasedCardWrapper struct {
	Value slugCardInner `wrapped:"true"`
}

func (aliasedCardWrapper) GetGraphQLType() string {
	return "x: card(slug:$slug)"
}

func TestUnmarshalGraphQL_GraphQLTypeWithAliasMatchesAlias(t *testing.T) {
	t.Parallel()

	type payload struct {
		Card aliasedCardWrapper
	}
	var got payload
	err := decode.UnmarshalGraphQL(
		[]byte(`{"x":{"slug":"def"}}`), &got,
	)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Card.Value.Slug != "def" {
		t.Errorf("got slug %q, want %q", got.Card.Value.Slug, "def")
	}
}
