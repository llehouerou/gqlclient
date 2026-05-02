package graphql_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	graphql "github.com/llehouerou/gqlclient"
)

// TestErrors_Is_localJSONDecode verifies that a malformed server response
// surfaces as ErrJSONDecode via errors.Is.
func TestErrors_Is_localJSONDecode(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	}))
	t.Cleanup(srv.Close)

	c := graphql.NewClient(srv.URL, nil)
	var q struct {
		Viewer struct{ Login string }
	}
	err := c.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, graphql.ErrJSONDecode) {
		t.Errorf("errors.Is(err, ErrJSONDecode) = false; want true (err = %v)", err)
	}
}

// TestErrors_Is_serverCodeMatchesSentinel verifies that an Error decoded
// from a server response with a recognized code in Extensions matches the
// matching sentinel via errors.Is.
func TestErrors_Is_serverCodeMatchesSentinel(t *testing.T) {
	t.Parallel()

	// Server returns a GraphQL error with a code our sentinels recognize.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{
				"message":    "boom",
				"extensions": map[string]any{"code": graphql.ErrCodeRequest},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c := graphql.NewClient(srv.URL, nil)
	var q struct {
		Viewer struct{ Login string }
	}
	err := c.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, graphql.ErrRequest) {
		t.Errorf("server error with code=%q did not match ErrRequest sentinel (err = %v)",
			graphql.ErrCodeRequest, err)
	}
}

// TestErrors_Is_unrelatedSentinelDoesNotMatch verifies negative path: a
// JSON-decode failure must not satisfy errors.Is(_, ErrGraphQLEncode).
func TestErrors_Is_unrelatedSentinelDoesNotMatch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	t.Cleanup(srv.Close)

	c := graphql.NewClient(srv.URL, nil)
	var q struct {
		Viewer struct{ Login string }
	}
	err := c.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, graphql.ErrGraphQLEncode) {
		t.Errorf("a json decode error should not match ErrGraphQLEncode (err = %v)", err)
	}
}

// TestErrors_As_recoversErrors verifies errors.As pulls the Errors slice
// back out, so callers can iterate over server-side error details.
func TestErrors_As_recoversErrors(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"message": "first", "extensions": map[string]any{"code": "X1"}},
				{"message": "second", "extensions": map[string]any{"code": "X2"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := graphql.NewClient(srv.URL, nil)
	var q struct {
		Viewer struct{ Login string }
	}
	err := c.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var gqlErrs graphql.Errors
	if !errors.As(err, &gqlErrs) {
		t.Fatalf("errors.As did not recover graphql.Errors (err = %v)", err)
	}
	if len(gqlErrs) != 2 {
		t.Fatalf("expected 2 errors in slice, got %d", len(gqlErrs))
	}
	if gqlErrs[0].Message != "first" || gqlErrs[1].Message != "second" {
		t.Errorf("unexpected error messages: %+v", gqlErrs)
	}
}

// TestError_As_recoversSingleError verifies errors.As with *Error finds
// the first matching Error inside an Errors slice via Unwrap()[]error.
func TestError_As_recoversSingleError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"message": "boom", "extensions": map[string]any{"code": "X"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := graphql.NewClient(srv.URL, nil)
	var q struct {
		Viewer struct{ Login string }
	}
	err := c.Query(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var single graphql.Error
	if !errors.As(err, &single) {
		t.Fatalf("errors.As did not recover graphql.Error from the chain (err = %v)", err)
	}
	if single.Message != "boom" {
		t.Errorf("Error.Message = %q, want %q", single.Message, "boom")
	}
}
