package graphql_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	graphql "github.com/llehouerou/gqlclient"
)

// TestDecodeResponse_envelope verifies the primitive returns a *Response
// carrying raw data, GraphQL errors, and top-level extensions. The second
// return is reserved for a local decode failure and must be nil for a
// well-formed envelope, even one that contains GraphQL errors.
func TestDecodeResponse_envelope(t *testing.T) {
	t.Parallel()

	body := `{
		"data": {"user": {"name": "Alice"}},
		"errors": [{"message": "deprecated field", "path": ["user", "name"]}],
		"extensions": {"cost": {"actual": 3}, "requestId": "abc123"}
	}`

	client := graphql.NewClient("http://example.com/graphql", nil)
	resp, decErrs := client.DecodeResponse(strings.NewReader(body))
	if decErrs != nil {
		t.Fatalf("unexpected local decode errors: %v", decErrs)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Data) == 0 {
		t.Error("expected raw data to be present")
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Message != "deprecated field" {
		t.Errorf("expected GraphQL error in resp.Errors, got %v", resp.Errors)
	}
	if resp.Extensions == nil {
		t.Fatal("expected top-level extensions to be surfaced")
	}
	if _, ok := resp.Extensions["cost"]; !ok {
		t.Errorf("expected extensions[cost], got %v", resp.Extensions)
	}
	if id, _ := resp.Extensions["requestId"].(string); id != "abc123" {
		t.Errorf("expected extensions[requestId]=abc123, got %v", resp.Extensions["requestId"])
	}
}

// TestDecodeResponse_invalidJSON checks that an unparseable envelope flows out
// of the second return (local decode failure), with a nil *Response.
func TestDecodeResponse_invalidJSON(t *testing.T) {
	t.Parallel()

	client := graphql.NewClient("http://example.com/graphql", nil)
	resp, decErrs := client.DecodeResponse(strings.NewReader(`{invalid}`))
	if decErrs == nil {
		t.Fatal("expected a local decode failure, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response on decode failure, got %v", resp)
	}
}

// TestQueryWithResponse_extensions verifies the typed struct is still
// populated AND the top-level extensions are returned to the caller.
func TestQueryWithResponse_extensions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{
			"data": {"user": {"name": "Alice"}},
			"extensions": {"tracing": {"version": 1}}
		}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct{ Name string }
	}
	resp, err := client.QueryWithResponse(context.Background(), &q, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.User.Name != "Alice" {
		t.Errorf("expected struct populated, got name=%q", q.User.Name)
	}
	if resp == nil || resp.Extensions == nil {
		t.Fatalf("expected extensions on response, got %v", resp)
	}
	if _, ok := resp.Extensions["tracing"]; !ok {
		t.Errorf("expected extensions[tracing], got %v", resp.Extensions)
	}
}

// TestMutateWithResponse_extensions mirrors the query path for mutations.
func TestMutateWithResponse_extensions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{
			"data": {"createUser": {"id": "1"}},
			"extensions": {"requestId": "m-1"}
		}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var m struct {
		CreateUser struct{ ID string } `graphql:"createUser"`
	}
	resp, err := client.MutateWithResponse(context.Background(), &m, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CreateUser.ID != "1" {
		t.Errorf("expected struct populated, got id=%q", m.CreateUser.ID)
	}
	if resp == nil || resp.Extensions["requestId"] != "m-1" {
		t.Fatalf("expected extensions[requestId]=m-1, got %v", resp)
	}
}

// TestExecuteQueryWithResponse_extensions verifies the prebuilt-query path
// also surfaces top-level extensions while populating v in place.
func TestExecuteQueryWithResponse_extensions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{
			"data": {"user": {"name": "Alice"}},
			"extensions": {"cost": {"actual": 7}}
		}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var v struct {
		User struct {
			Name string `json:"name"`
		} `json:"user"`
	}
	resp, err := client.ExecuteQueryWithResponse(
		context.Background(),
		`query { user { name } }`,
		&v,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.User.Name != "Alice" {
		t.Errorf("expected struct populated, got name=%q", v.User.Name)
	}
	if resp == nil || resp.Extensions["cost"] == nil {
		t.Fatalf("expected extensions[cost] on response, got %v", resp)
	}
}

// TestQueryWithResponse_partialDataCarriesExtensions verifies that when the
// server returns partial data alongside GraphQL errors, the caller still gets
// the error AND the response envelope (with extensions and partial data).
func TestQueryWithResponse_partialDataCarriesExtensions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, `{
			"data": {"user": {"name": "Alice"}},
			"errors": [{"message": "boom", "path": ["other"]}],
			"extensions": {"requestId": "p-1"}
		}`)
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	var q struct {
		User struct{ Name string }
	}
	resp, err := client.QueryWithResponse(context.Background(), &q, nil)
	if err == nil {
		t.Fatal("expected GraphQL error, got nil")
	}
	if resp == nil || resp.Extensions["requestId"] != "p-1" {
		t.Fatalf("expected extensions on partial response, got %v", resp)
	}
	if q.User.Name != "Alice" {
		t.Errorf("expected partial data populated, got name=%q", q.User.Name)
	}
}
