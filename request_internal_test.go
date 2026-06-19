package graphql

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// recordingRoundTripper serves requests through an in-memory handler, so the
// transport seam can be exercised without a real network listener. It lives in
// package graphql (not graphql_test) because executeRequest is unexported — see
// docs/adr/0001-public-api-surface.md.
type recordingRoundTripper struct{ handler http.Handler }

func (rt recordingRoundTripper) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	w := httptest.NewRecorder()
	rt.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func newSeamClient(handler http.Handler) *Client {
	return NewClient(
		"/graphql",
		&http.Client{Transport: recordingRoundTripper{handler: handler}},
	)
}

func newSeamRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/graphql",
		strings.NewReader("{}"),
	)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	return req
}

// TestClient_executeRequest covers the single internal transport seam directly:
// it sends the request, transparently decompresses gzip, and rejects non-200
// responses with a readable body.
func TestClient_executeRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns the body on success", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":{"user":{"name":"Alice"}}}`)
		})

		resp, body, err := newSeamClient(mux).executeRequest(newSeamRequest(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		defer func() { _ = body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		got, err := io.ReadAll(body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if want := `{"data":{"user":{"name":"Alice"}}}`; string(got) != want {
			t.Errorf("body = %q, want %q", got, want)
		}
	})

	t.Run("transparently decompresses gzip", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer func() { _ = gz.Close() }()
			_, _ = gz.Write([]byte(`{"data":{"user":{"name":"Bob"}}}`))
		})

		resp, body, err := newSeamClient(mux).executeRequest(newSeamRequest(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		defer func() { _ = body.Close() }()

		got, err := io.ReadAll(body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if want := `{"data":{"user":{"name":"Bob"}}}`; string(got) != want {
			t.Errorf("decompressed body = %q, want %q", got, want)
		}
	})

	t.Run("rejects non-200 with status in the error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})

		//nolint:bodyclose // executeRequest closes the body before returning a non-nil error
		_, _, err := newSeamClient(mux).executeRequest(newSeamRequest(t))
		if err == nil {
			t.Fatal("expected error for non-200 status, got nil")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected error to mention status 500, got %q", err.Error())
		}
	})

	t.Run("decompresses a gzipped non-200 body before reporting it", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/graphql", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusInternalServerError)
			gz := gzip.NewWriter(w)
			defer func() { _ = gz.Close() }()
			_, _ = gz.Write([]byte("internal boom"))
		})

		//nolint:bodyclose // executeRequest closes the body before returning a non-nil error
		_, _, err := newSeamClient(mux).executeRequest(newSeamRequest(t))
		if err == nil {
			t.Fatal("expected error for non-200 status, got nil")
		}
		if !strings.Contains(err.Error(), "internal boom") {
			t.Errorf("expected decompressed body in error, got %q", err.Error())
		}
	})
}
