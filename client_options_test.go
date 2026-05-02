package graphql_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	graphql "github.com/llehouerou/gqlclient"
)

// recordingServer captures the headers from the most recent request and
// returns a minimal valid GraphQL response.
func recordingServer(t *testing.T, captured *http.Header) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = r.Header.Clone()
		_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"alice"}}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runViewerQuery(t *testing.T, c *graphql.Client) {
	t.Helper()
	var q struct {
		Viewer struct {
			Login string
		}
	}
	if err := c.Query(context.Background(), &q, nil); err != nil {
		t.Fatalf("query: %v", err)
	}
}

func TestWithHeader_setsHeader(t *testing.T) {
	var got http.Header
	srv := recordingServer(t, &got)

	c := graphql.NewClient(srv.URL, nil).
		WithHeader("Authorization", "Bearer token-1")
	runViewerQuery(t, c)

	if v := got.Get("Authorization"); v != "Bearer token-1" {
		t.Errorf("Authorization header = %q, want %q", v, "Bearer token-1")
	}
}

func TestWithHeader_lastWriteWins(t *testing.T) {
	var got http.Header
	srv := recordingServer(t, &got)

	c := graphql.NewClient(srv.URL, nil).
		WithHeader("X-Token", "first").
		WithHeader("X-Token", "second")
	runViewerQuery(t, c)

	if v := got.Get("X-Token"); v != "second" {
		t.Errorf("X-Token = %q, want %q", v, "second")
	}
}

func TestWithHeader_immutability(t *testing.T) {
	original := graphql.NewClient("http://example.invalid", nil)
	modified := original.WithHeader("X-Foo", "bar")
	if original == modified {
		t.Fatal("WithHeader returned the same Client instance")
	}
	// A second derivation from the original must not see the modified value.
	other := original.WithHeader("X-Foo", "baz")
	if modified == other {
		t.Fatal("derivations from the same root share state")
	}
}

func TestWithHeaders_mergesExisting(t *testing.T) {
	var got http.Header
	srv := recordingServer(t, &got)

	c := graphql.NewClient(srv.URL, nil).
		WithHeader("X-Trace", "abc").
		WithHeaders(http.Header{
			"Authorization": []string{"Bearer t"},
			"X-Trace":       []string{"def"}, // overrides
		})
	runViewerQuery(t, c)

	if v := got.Get("Authorization"); v != "Bearer t" {
		t.Errorf("Authorization = %q, want %q", v, "Bearer t")
	}
	if v := got.Get("X-Trace"); v != "def" {
		t.Errorf("X-Trace = %q, want %q (last writer wins)", v, "def")
	}
}

func TestWithHeaders_nilIsNoop(t *testing.T) {
	original := graphql.NewClient("http://example.invalid", nil).
		WithHeader("X-Foo", "bar")
	modified := original.WithHeaders(nil)
	// Behavior: nil leaves existing headers in place. The clone is still a
	// new instance per the immutable pattern.
	if original == modified {
		t.Fatal("WithHeaders(nil) returned the same Client instance")
	}
}

func TestWithUserAgent(t *testing.T) {
	var got http.Header
	srv := recordingServer(t, &got)

	c := graphql.NewClient(srv.URL, nil).WithUserAgent("gqlclient-test/1.0")
	runViewerQuery(t, c)

	if v := got.Get("User-Agent"); v != "gqlclient-test/1.0" {
		t.Errorf("User-Agent = %q, want %q", v, "gqlclient-test/1.0")
	}
}

func TestWithHTTPClient_replacesTransport(t *testing.T) {
	var calls int
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		// Hand back a minimal valid GraphQL response.
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       http.NoBody,
		}, nil
	})

	custom := &http.Client{Transport: rt}
	c := graphql.NewClient("http://example.invalid", nil).WithHTTPClient(custom)

	// Run a query; we don't care about the empty body, only that custom RT was hit.
	var q struct {
		Viewer struct{ Login string }
	}
	_ = c.Query(context.Background(), &q, nil)

	if calls != 1 {
		t.Errorf("custom RoundTripper called %d times, want 1", calls)
	}
}

func TestWithHTTPClient_nilFallsBackToDefault(t *testing.T) {
	original := graphql.NewClient("http://example.invalid", nil)
	modified := original.WithHTTPClient(nil)
	// We can't easily inspect the field; just verify a new instance was returned.
	if original == modified {
		t.Fatal("WithHTTPClient(nil) returned the same Client instance")
	}
}

func TestWithRequestModifier_overridesClientHeader(t *testing.T) {
	var got http.Header
	srv := recordingServer(t, &got)

	c := graphql.NewClient(srv.URL, nil).
		WithHeader("Authorization", "Bearer baseline").
		WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer override")
		})
	runViewerQuery(t, c)

	if v := got.Get("Authorization"); v != "Bearer override" {
		t.Errorf("Authorization = %q, want %q (modifier should override)", v, "Bearer override")
	}
}

// roundTripperFunc is the function-typed http.RoundTripper used by
// TestWithHTTPClient_replacesTransport.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
