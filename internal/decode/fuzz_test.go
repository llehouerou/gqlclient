package decode_test

import (
	"testing"

	"github.com/llehouerou/gqlclient/internal/decode"
)

// FuzzUnmarshalGraphQL exercises UnmarshalGraphQL with mutated JSON inputs to
// flush out panics, infinite loops, or other crash modes in the custom JSON
// decoder. The test target is intentionally varied (struct fields, embedded
// fragment, nested slice, ordered map) so the fuzzer reaches as many decode
// branches as possible.
//
// Run as:
//
//	go test ./internal/decode -fuzz=FuzzUnmarshalGraphQL -fuzztime=30s
func FuzzUnmarshalGraphQL(f *testing.F) {
	// Seed corpus: representative shapes the decoder must handle.
	seeds := [][]byte{
		[]byte(`null`),
		[]byte(`{}`),
		[]byte(`[]`),
		[]byte(`{"viewer":{"login":"alice"}}`),
		[]byte(`{"viewer":null}`),
		[]byte(`{"viewer":{"login":"alice","createdAt":"2024-01-02T03:04:05Z"}}`),
		[]byte(`{"items":[{"name":"a"},{"name":"b"}]}`),
		[]byte(`{"items":[]}`),
		[]byte(`{"node":{"__typename":"User","login":"alice"}}`),
		[]byte(`{"node":{"__typename":"Org","slug":"acme"}}`),
		[]byte(`{"a":{"b":{"c":{"d":"deep"}}}}`),
		[]byte(`{"viewer":{"login":42}}`), // type mismatch
		[]byte(`{"viewer":{"login":"x"`),  // truncated
		[]byte("{\"viewer\":{\"login\":\"\x00\"}}"),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	type fragmentNode struct {
		Typename string `graphql:"__typename" json:"__typename"`
		User     struct {
			Login string
		} `graphql:"... on User"`
		Org struct {
			Slug string
		} `graphql:"... on Org"`
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// The decoder must not panic for any input. It is allowed to return
		// an error; we only care about non-crashing behavior.
		var simple struct {
			Viewer struct {
				Login     string
				CreatedAt string
			}
			Items []struct {
				Name string
			}
		}
		_ = decode.UnmarshalGraphQL(data, &simple)

		var withFragment struct {
			Node fragmentNode
		}
		_ = decode.UnmarshalGraphQL(data, &withFragment)

		var orderedMap [][2]any
		_ = decode.UnmarshalGraphQL(data, &orderedMap)
	})
}
