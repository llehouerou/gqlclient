package graphql

import (
	"errors"
	"net/http"
	"testing"
)

// These exercise the unexported error-construction seam (decorate /
// newRequestError) directly. They live in-package because that surface is
// internal — see docs/adr/0001-public-api-surface.md. Behavior is observed
// end-to-end through the public Query path elsewhere in graphql_test.go.

func TestClient_decorate(t *testing.T) {
	t.Parallel()

	t.Run("debug mode enabled with request and response", func(t *testing.T) {
		t.Parallel()

		client := NewClient("http://example.com", nil).WithDebug(true)

		req, err := http.NewRequest(
			http.MethodPost,
			"http://example.com",
			http.NoBody,
		)
		if err != nil {
			t.Fatal(err)
		}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}

		baseErr := Error{
			Message:    "test error",
			Extensions: map[string]any{"code": ErrCodeRequest},
		}

		decorated := client.decorate(baseErr, errorContext{
			req:      req,
			reqBody:  []byte(`{"query":"{test}"}`),
			resp:     resp,
			respBody: []byte(`{"data":null}`),
		})

		internal, ok := decorated.Extensions["internal"].(map[string]any)
		if !ok {
			t.Fatal("expected internal extensions to exist")
		}
		if _, ok := internal["request"]; !ok {
			t.Error("expected request information in internal extensions")
		}
		if _, ok := internal["response"]; !ok {
			t.Error("expected response information in internal extensions")
		}
	})

	t.Run("debug mode disabled", func(t *testing.T) {
		t.Parallel()

		client := NewClient("http://example.com", nil).WithDebug(false)

		req, err := http.NewRequest(
			http.MethodPost,
			"http://example.com",
			http.NoBody,
		)
		if err != nil {
			t.Fatal(err)
		}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}

		baseErr := Error{
			Message:    "test error",
			Extensions: map[string]any{"code": ErrCodeRequest},
		}

		decorated := client.decorate(baseErr, errorContext{
			req:      req,
			reqBody:  []byte(`{"query":"{test}"}`),
			resp:     resp,
			respBody: []byte(`{"data":null}`),
		})

		if decorated.Extensions != nil {
			if internal, ok := decorated.Extensions["internal"].(map[string]any); ok {
				if len(internal) > 0 {
					t.Error("expected no internal extensions in non-debug mode")
				}
			}
		}
		if code, ok := decorated.Extensions["code"].(string); !ok ||
			code != ErrCodeRequest {
			t.Error("expected code to be preserved")
		}
	})
}

func TestClient_newRequestError(t *testing.T) {
	t.Parallel()

	t.Run("creates error with code and message", func(t *testing.T) {
		t.Parallel()

		client := NewClient("http://example.com", nil)

		err := client.newRequestError(
			ErrCodeJSONDecode,
			errors.New("json decode failed"),
			errorContext{},
		)

		if err.Message != "json decode failed" {
			t.Errorf("expected message 'json decode failed', got %q", err.Message)
		}
		if code, ok := err.Extensions["code"].(string); !ok ||
			code != ErrCodeJSONDecode {
			t.Errorf("expected code %q, got %v", ErrCodeJSONDecode, code)
		}
	})

	t.Run("decorates with debug info when enabled", func(t *testing.T) {
		t.Parallel()

		client := NewClient("http://example.com", nil).WithDebug(true)

		req, err := http.NewRequest(
			http.MethodPost,
			"http://example.com",
			http.NoBody,
		)
		if err != nil {
			t.Fatal(err)
		}
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}

		decoratedErr := client.newRequestError(
			ErrCodeRequest,
			errors.New("server error"),
			errorContext{
				req:      req,
				reqBody:  []byte(`{"query":"{test}"}`),
				resp:     resp,
				respBody: []byte(`{"errors":[]}`),
			},
		)

		if decoratedErr.Message != "server error" {
			t.Errorf("expected message 'server error', got %q", decoratedErr.Message)
		}
		internal, ok := decoratedErr.Extensions["internal"].(map[string]any)
		if !ok {
			t.Fatal("expected internal extensions in debug mode")
		}
		if _, ok := internal["request"]; !ok {
			t.Error("expected request information in debug mode")
		}
		if _, ok := internal["response"]; !ok {
			t.Error("expected response information in debug mode")
		}
	})
}

// TestClient_decorate_internalExtensionsRoundTrip checks that an error
// decorated via decorate can be read back through the public
// GetInternalExtensions API.
func TestClient_decorate_internalExtensionsRoundTrip(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com", nil).WithDebug(true)

	reqBody := `{"query":"{test}"}`
	respBody := `{"data":null,"errors":[{"message":"error"}]}`
	req, err := http.NewRequest(
		http.MethodPost,
		"http://example.com",
		http.NoBody,
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	baseErr := Error{
		Message:    "test error",
		Extensions: map[string]any{"code": ErrCodeRequest},
	}

	decorated := client.decorate(baseErr, errorContext{
		req:      req,
		reqBody:  []byte(reqBody),
		resp:     resp,
		respBody: []byte(respBody),
	})

	if code := decorated.GetCode(); code != ErrCodeRequest {
		t.Errorf("expected code %q, got %q", ErrCodeRequest, code)
	}

	internal := decorated.GetInternalExtensions()
	if internal == nil {
		t.Fatal("expected non-nil internal extensions")
	}
	if internal.Request == nil {
		t.Fatal("expected non-nil request info")
	}
	if internal.Request.Body != reqBody {
		t.Errorf("expected request body %q, got %q", reqBody, internal.Request.Body)
	}
	if len(internal.Request.Headers) == 0 {
		t.Error("expected non-empty request headers")
	}
	if internal.Response == nil {
		t.Fatal("expected non-nil response info")
	}
	if internal.Response.Body != respBody {
		t.Errorf("expected response body %q, got %q", respBody, internal.Response.Body)
	}
	if len(internal.Response.Headers) == 0 {
		t.Error("expected non-empty response headers")
	}
}
