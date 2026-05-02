package graphql_test

import (
	"testing"

	graphql "github.com/llehouerou/gqlclient"
)

// TestNewID tests the NewID and ToID helper functions for the ID type.
// ID is the only actively supported custom scalar type.
func TestNewID(t *testing.T) {
	t.Parallel()

	t.Run("string input", func(t *testing.T) {
		t.Parallel()

		got := graphql.NewID("")
		if got == nil {
			t.Fatal("NewID returned nil for empty string")
		}

		got = graphql.NewID("user-123")
		if got == nil {
			t.Fatal("NewID returned nil for non-empty string")
		}
		if *got != "user-123" {
			t.Errorf("NewID(\"user-123\") = %q, want \"user-123\"", *got)
		}
	})

	t.Run("integer input", func(t *testing.T) {
		t.Parallel()

		got := graphql.NewID(0)
		if got == nil {
			t.Fatal("NewID returned nil for integer 0")
		}
		if *got != "" {
			t.Errorf("NewID(0) = %q, want empty string", *got)
		}

		got = graphql.NewID(42)
		if got == nil {
			t.Fatal("NewID returned nil for integer 42")
		}
		if *got != "42" {
			t.Errorf("NewID(42) = %q, want \"42\"", *got)
		}
	})
}

// TestToID tests the ToID conversion function.
func TestToID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  graphql.ID
	}{
		{"empty string", "", graphql.ID("")},
		{"non-empty string", "user-123", graphql.ID("user-123")},
		{"zero integer", 0, graphql.ID("")},
		{"positive integer", 42, graphql.ID("42")},
		{"int32", int32(100), graphql.ID("100")},
		{"int64", int64(200), graphql.ID("200")},
		{"uint", uint(300), graphql.ID("300")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := graphql.ToID(tt.input)
			if got != tt.want {
				t.Errorf("ToID(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
