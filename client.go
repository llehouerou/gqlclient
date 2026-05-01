package graphql

import (
	"context"
	"io"
	"net/http"

	"github.com/llehouerou/gqlclient/internal/decode"
)

// RequestModifier allows you to tweak the HTTP request. It might be useful to set authentication
// headers amongst other things
type RequestModifier func(*http.Request)

// Client is a GraphQL client.
//
// # Immutable Pattern
//
// The Client's With* methods (WithDebug, WithRequestModifier) follow an
// immutable pattern: they return a new Client instance rather than modifying
// the receiver. This allows for safe concurrent use and makes it clear when
// configuration changes take effect.
//
// Always use the returned Client:
//
//	client = client.WithDebug(true)  // Correct
//	client.WithDebug(true)            // Wrong - original client unchanged
//
// Methods can be chained since each returns a new Client:
//
//	client = client.WithDebug(true).WithRequestModifier(modifier)
//
// Note: This differs from SubscriptionClient, whose With* methods modify
// the receiver and return self (mutable/builder pattern).
type Client struct {
	url             string // GraphQL server URL.
	httpClient      *http.Client
	requestModifier RequestModifier
	debug           bool
}

// NewClient creates a GraphQL client targeting the specified GraphQL server URL.
// If httpClient is nil, then http.DefaultClient is used.
func NewClient(url string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		url:             url,
		httpClient:      httpClient,
		requestModifier: nil,
	}
}

// Query executes a single GraphQL query request,
// with a query derived from q, populating the response into it.
// q should be a pointer to struct that corresponds to the GraphQL schema.
//
// The variables parameter must be either nil, a map[string]any, or a struct/pointer to struct
// with json tags. Passing any other type will cause a panic (programming error).
func (c *Client) Query(
	ctx context.Context,
	q any,
	variables any,
	options ...Option,
) error {
	return c.executeAndUnmarshal(ctx, queryOperation, q, variables, options...)
}

// Mutate executes a single GraphQL mutation request,
// with a mutation derived from m, populating the response into it.
// m should be a pointer to struct that corresponds to the GraphQL schema.
//
// The variables parameter must be either nil, a map[string]any, or a struct/pointer to struct
// with json tags. Passing any other type will cause a panic (programming error).
func (c *Client) Mutate(
	ctx context.Context,
	m any,
	variables any,
	options ...Option,
) error {
	return c.executeAndUnmarshal(ctx, mutationOperation, m, variables, options...)
}

// QueryRaw executes a single GraphQL query request,
// with a query derived from q, populating the response into it.
// q should be a pointer to struct that corresponds to the GraphQL schema.
// Returns raw bytes message.
//
// The variables parameter must be either nil, a map[string]any, or a struct/pointer to struct
// with json tags. Passing any other type will cause a panic (programming error).
func (c *Client) QueryRaw(
	ctx context.Context,
	q any,
	variables any,
	options ...Option,
) ([]byte, error) {
	return c.executeForRawJSON(ctx, queryOperation, q, variables, options...)
}

// MutateRaw executes a single GraphQL mutation request,
// with a mutation derived from m, populating the response into it.
// m should be a pointer to struct that corresponds to the GraphQL schema.
// Returns raw bytes message.
//
// The variables parameter must be either nil, a map[string]any, or a struct/pointer to struct
// with json tags. Passing any other type will cause a panic (programming error).
func (c *Client) MutateRaw(
	ctx context.Context,
	m any,
	variables any,
	options ...Option,
) ([]byte, error) {
	return c.executeForRawJSON(ctx, mutationOperation, m, variables, options...)
}

// ExecuteQuery executes a pre-built GraphQL query string and unmarshals the
// response into v. Unlike Query, you must manually specify all fields in the
// query string as they are not inferred from v. This is useful when you need
// to build queries dynamically.
func (c *Client) ExecuteQuery(
	ctx context.Context,
	query string,
	v any,
	variables map[string]any,
	options ...Option,
) error {
	data, resp, respBuf, errs := c.request(ctx, query, variables) //nolint:bodyclose // closed inside c.request
	return c.processResponse(v, data, resp, respBuf, errs)
}

// ExecuteQueryRaw executes a pre-built GraphQL query string and returns the
// raw JSON response without unmarshaling. Unlike Query, you must manually
// specify all fields in the query string. This is useful when you need to
// build queries dynamically or process the raw JSON response yourself.
func (c *Client) ExecuteQueryRaw(
	ctx context.Context,
	query string,
	variables map[string]any,
	options ...Option,
) ([]byte, error) {
	data, _, _, errs := c.request(ctx, query, variables) //nolint:bodyclose // closed inside c.request
	if len(errs) > 0 {
		return data, errs
	}
	return data, nil
}

// clone creates a copy of the Client with all fields preserved.
// This helper prevents field-copying bugs when adding new fields to Client.
func (c *Client) clone() *Client {
	return &Client{
		url:             c.url,
		httpClient:      c.httpClient,
		requestModifier: c.requestModifier,
		debug:           c.debug,
	}
}

// WithRequestModifier returns a new Client with the request modifier set.
// This allows you to reuse the same TCP connection for multiple slightly
// different requests to the same server (e.g., different authentication
// headers for multitenant applications).
//
// This method follows an immutable pattern: it returns a NEW Client instance
// without modifying the original. You must use the returned Client:
//
//	client = client.WithRequestModifier(modifier)  // Correct
//	client.WithRequestModifier(modifier)            // Wrong - has no effect
//
// The method can be chained with other With* methods:
//
//	client = client.WithRequestModifier(modifier).WithDebug(true)
func (c *Client) WithRequestModifier(f RequestModifier) *Client {
	clone := c.clone()
	clone.requestModifier = f
	return clone
}

// WithDebug returns a new Client with debug mode enabled or disabled.
// When enabled, debug mode adds detailed request/response information to
// error extensions, which is useful for troubleshooting GraphQL API issues.
//
// This method follows an immutable pattern: it returns a NEW Client instance
// without modifying the original. You must use the returned Client:
//
//	client = client.WithDebug(true)  // Correct
//	client.WithDebug(true)            // Wrong - has no effect
//
// The method can be chained with other With* methods:
//
//	client = client.WithDebug(true).WithRequestModifier(modifier)
func (c *Client) WithDebug(debug bool) *Client {
	clone := c.clone()
	clone.debug = debug
	return clone
}

// UnmarshalGraphQL parses the JSON-encoded GraphQL response data and stores
// the result in the GraphQL query data structure pointed to by v.
//
// The implementation is created on top of the JSON tokenizer available
// in "encoding/json".Decoder.
// This function is re-exported from the internal package
func UnmarshalGraphQL(data []byte, v any) error {
	return decode.UnmarshalGraphQL(data, v)
}

// executeAndUnmarshal executes a single GraphQL operation and unmarshals the
// response into v.
func (c *Client) executeAndUnmarshal(
	ctx context.Context,
	op operationType,
	v any,
	variables any,
	options ...Option,
) error {
	//nolint:bodyclose // closed inside c.request, called by c.constructQueryAndExecute
	data, resp, respBuf, errs := c.constructQueryAndExecute(
		ctx,
		op,
		v,
		variables,
		options...)
	return c.processResponse(v, data, resp, respBuf, errs)
}

// executeForRawJSON executes a single GraphQL operation and returns the raw
// JSON response without unmarshaling.
func (c *Client) executeForRawJSON(
	ctx context.Context,
	op operationType,
	v any,
	variables any,
	options ...Option,
) ([]byte, error) {
	//nolint:bodyclose // closed inside c.request, called by c.constructQueryAndExecute
	data, _, _, err := c.constructQueryAndExecute(
		ctx,
		op,
		v,
		variables,
		options...)
	if len(err) > 0 {
		return data, err
	}
	return data, nil
}

// constructQueryAndExecute constructs a GraphQL query from the provided struct
// and executes it, returning the raw response data and any errors.
func (c *Client) constructQueryAndExecute(
	ctx context.Context,
	op operationType,
	v any,
	variables any,
	options ...Option,
) ([]byte, *http.Response, io.Reader, Errors) {
	var query string
	var err error
	switch op {
	case queryOperation:
		query, err = ConstructQuery(v, variables, options...)
	case mutationOperation:
		query, err = ConstructMutation(v, variables, options...)
	}

	if err != nil {
		return nil, nil, nil, newSimpleErrors(ErrGraphQLEncode, err)
	}

	return c.request(ctx, query, variables)
}

type operationType uint8

const (
	queryOperation operationType = iota
	mutationOperation
	// subscriptionOperation // Unused.
)
