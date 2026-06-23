package graphql_test

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/llehouerou/gqlclient"
)

// TestError_Path_decode verifies that the GraphQL spec "path" array on an
// error entry decodes into Error.Path. Path segments are field names
// (strings) or 0-indexed list positions (integers, decoded as float64 in
// an any).
func TestError_Path_decode(t *testing.T) {
	t.Parallel()

	body := `[{"message":"boom","path":["hero","friends",2,"name"]}]`

	var errs graphql.Errors
	if err := json.Unmarshal([]byte(body), &errs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if got := len(errs[0].Path); got != 4 {
		t.Fatalf("expected path length 4, got %d (%v)", got, errs[0].Path)
	}
	if errs[0].Path[0] != "hero" {
		t.Errorf("path[0] = %v, want hero", errs[0].Path[0])
	}
}

// TestError_PathString renders the path with integer indices formatted as
// integers (no ".0") and an empty string when no path is present.
func TestError_PathString(t *testing.T) {
	t.Parallel()

	t.Run("renders dotted path with int indices", func(t *testing.T) {
		t.Parallel()
		var errs graphql.Errors
		body := `[{"message":"boom","path":["hero","friends",2,"name"]}]`
		if err := json.Unmarshal([]byte(body), &errs); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got := errs[0].PathString(); got != "hero.friends.2.name" {
			t.Errorf("PathString() = %q, want %q", got, "hero.friends.2.name")
		}
	})

	t.Run("empty when no path", func(t *testing.T) {
		t.Parallel()
		e := graphql.Error{Message: "boom"}
		if got := e.PathString(); got != "" {
			t.Errorf("PathString() = %q, want empty", got)
		}
	})
}

// TestError_Error_includesPath checks that Error() surfaces the path when one
// is present (so logs say which field failed) and preserves the prior format
// when no path is present.
func TestError_Error_includesPath(t *testing.T) {
	t.Parallel()

	t.Run("includes path when present", func(t *testing.T) {
		t.Parallel()
		var errs graphql.Errors
		body := `[{"message":"boom","path":["hero","friends",2,"name"]}]`
		if err := json.Unmarshal([]byte(body), &errs); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got := errs[0].Error(); !strings.Contains(got, "hero.friends.2.name") {
			t.Errorf("Error() = %q, want it to contain the path", got)
		}
	})

	t.Run("omits path segment when absent", func(t *testing.T) {
		t.Parallel()
		e := graphql.Error{Message: "boom"}
		if got := e.Error(); strings.Contains(got, "Path") {
			t.Errorf("Error() = %q, should not mention Path when absent", got)
		}
	})
}
