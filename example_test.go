package graphql_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	graphql "github.com/llehouerou/gqlclient"
)

// ExampleNewClient shows the most common setup: a Client targeting a GraphQL
// endpoint, using http.DefaultClient.
func ExampleNewClient() {
	client := graphql.NewClient("https://api.example.com/graphql", nil)
	_ = client
}

// ExampleClient_Query shows a typical query: a struct with `graphql:"…"`
// tags, populated in place by Query.
func ExampleClient_Query() {
	client := graphql.NewClient("https://api.example.com/graphql", nil)

	var q struct {
		Viewer struct {
			Login     string
			CreatedAt time.Time
		}
	}
	if err := client.Query(context.Background(), &q, nil); err != nil {
		fmt.Println("query failed:", err)
		return
	}
	fmt.Println("viewer:", q.Viewer.Login)
}

// ExampleClient_QueryWithResponse shows reading top-level response extensions
// (tracing, query cost, rate limits, request IDs, …) while still populating
// the struct in place.
func ExampleClient_QueryWithResponse() {
	client := graphql.NewClient("https://api.example.com/graphql", nil)

	var q struct {
		Viewer struct{ Login string }
	}
	resp, err := client.QueryWithResponse(context.Background(), &q, nil)
	if err != nil {
		fmt.Println("query failed:", err)
		return
	}
	fmt.Println("viewer:", q.Viewer.Login)
	if cost, ok := resp.Extensions["cost"]; ok {
		fmt.Println("query cost:", cost)
	}
}

// ExampleError_PathString shows locating which field failed in a partial
// response by inspecting the GraphQL error path.
func ExampleError_PathString() {
	client := graphql.NewClient("https://api.example.com/graphql", nil)

	var q struct {
		Hero struct{ Name string }
	}
	err := client.Query(context.Background(), &q, nil)

	var gqlErrs graphql.Errors
	if errors.As(err, &gqlErrs) {
		for _, e := range gqlErrs {
			fmt.Printf("%s failed: %s\n", e.PathString(), e.Message)
		}
	}
}

// ExampleClient_WithHeader shows the typical pattern for setting an
// Authorization header on every request issued by a Client.
func ExampleClient_WithHeader() {
	client := graphql.NewClient("https://api.example.com/graphql", nil).
		WithHeader("Authorization", "Bearer "+token())

	var q struct {
		Viewer struct{ Login string }
	}
	_ = client.Query(context.Background(), &q, nil)
}

// ExampleClient_WithHeaders shows setting multiple default headers at once,
// for instance for tenant routing or distributed tracing.
func ExampleClient_WithHeaders() {
	client := graphql.NewClient("https://api.example.com/graphql", nil).
		WithHeaders(http.Header{
			"Authorization": []string{"Bearer " + token()},
			"X-Tenant":      []string{"acme"},
		})
	_ = client
}

// ExampleClient_WithHTTPClient shows injecting a pre-configured *http.Client,
// for instance to set an overall request timeout, custom TLS, or an
// instrumented round tripper.
func ExampleClient_WithHTTPClient() {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	client := graphql.NewClient("https://api.example.com/graphql", nil).
		WithHTTPClient(httpClient)
	_ = client
}

// ExampleClient_WithRequestModifier shows the escape hatch for cases that
// don't fit the WithHeader / WithHeaders helpers — for example, a header
// value derived from per-request context.
func ExampleClient_WithRequestModifier() {
	client := graphql.NewClient("https://api.example.com/graphql", nil).
		WithRequestModifier(func(r *http.Request) {
			r.Header.Set("X-Request-ID", newRequestID())
		})
	_ = client
}

// ExampleClient_WithDebug shows enabling debug mode, which enriches errors
// with the originating HTTP request and response payloads.
func ExampleClient_WithDebug() {
	client := graphql.NewClient("https://api.example.com/graphql", nil).
		WithDebug(true)

	var q struct {
		Viewer struct{ Login string }
	}
	if err := client.Query(context.Background(), &q, nil); err != nil {
		// err carries request/response details under err.Extensions["internal"].
		fmt.Println(err)
	}
}

// token returns the bearer token to use for requests. In real code this
// would come from configuration, environment, or a secret store.
func token() string { return "" }

// newRequestID returns a unique identifier for a request. In real code this
// would typically be a UUID or trace-id propagated from the calling context.
func newRequestID() string { return "" }
