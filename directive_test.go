package graphql_test

import (
	"context"
	"net/http"
	"testing"

	graphql "github.com/llehouerou/gqlclient"
)

// TestUnmarshalGraphQL_FieldWithIncludeDirective is the no-server proof that
// the decoder matches the field name, not the whole tag, when a field carries
// an @include directive. The server returns "email"; the struct field is
// tagged `email @include(if: $withEmail)`.
func TestUnmarshalGraphQL_FieldWithIncludeDirective(t *testing.T) {
	t.Parallel()

	var q struct {
		User struct {
			Email *string `graphql:"email @include(if: $withEmail)"`
		} `graphql:"user(id: $id)"`
	}

	err := graphql.UnmarshalGraphQL(
		[]byte(`{"user":{"email":"a@x.io"}}`),
		&q,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.User.Email == nil || *q.User.Email != "a@x.io" {
		t.Fatalf("expected email a@x.io, got %v", q.User.Email)
	}
}

// TestUnmarshalGraphQL_FieldWithSkipDirective covers the @skip spelling and a
// field that also carries real arguments before the directive.
func TestUnmarshalGraphQL_FieldWithSkipDirective(t *testing.T) {
	t.Parallel()

	var q struct {
		Posts []struct {
			Title string
		} `graphql:"posts(first: 10) @skip(if: $hidePosts)"`
	}

	err := graphql.UnmarshalGraphQL(
		[]byte(`{"posts":[{"title":"hello"}]}`),
		&q,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Posts) != 1 || q.Posts[0].Title != "hello" {
		t.Fatalf("expected one post titled hello, got %v", q.Posts)
	}
}

// TestQuery_IncludeDirective_RoundTrip exercises both branches end to end: the
// field is present when included and absent (nil) when skipped, with no decode
// error in either case.
func TestQuery_IncludeDirective_RoundTrip(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		body := mustRead(req.Body)
		w.Header().Set("Content-Type", "application/json")
		// Emulate a server honoring @include.
		if containsSub(body, `"withEmail":true`) {
			mustWrite(
				w,
				`{"data":{"user":{"id":"1","name":"Alice","email":"a@x.io"}}}`,
			)
		} else {
			mustWrite(w, `{"data":{"user":{"id":"1","name":"Alice"}}}`)
		}
	})
	client := graphql.NewClient(
		"/graphql",
		&http.Client{Transport: localRoundTripper{handler: mux}},
	)

	type query struct {
		User struct {
			ID    graphql.ID
			Name  string
			Email *string `graphql:"email @include(if: $withEmail)"`
		} `graphql:"user(id: $id)"`
	}

	var inc query
	if err := client.Query(context.Background(), &inc, map[string]any{
		"id": graphql.ID("1"), "withEmail": true,
	}); err != nil {
		t.Fatalf("included branch: unexpected error: %v", err)
	}
	if inc.User.Email == nil || *inc.User.Email != "a@x.io" {
		t.Errorf(
			"included branch: expected email a@x.io, got %v",
			inc.User.Email,
		)
	}

	var skip query
	if err := client.Query(context.Background(), &skip, map[string]any{
		"id": graphql.ID("1"), "withEmail": false,
	}); err != nil {
		t.Fatalf("skipped branch: unexpected error: %v", err)
	}
	if skip.User.Email != nil {
		t.Errorf("skipped branch: expected nil email, got %v", *skip.User.Email)
	}
	if skip.User.Name != "Alice" {
		t.Errorf("skipped branch: expected name Alice, got %q", skip.User.Name)
	}
}

func containsSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
