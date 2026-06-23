package graphql

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/llehouerou/gqlclient/internal/decode"
)

// handleGzipResponse wraps the response body reader with a gzip decompressor
// if the Content-Encoding header indicates gzip compression.
// Returns the potentially-wrapped reader and any error encountered.
func handleGzipResponse(
	resp *http.Response,
	bodyReader io.Reader,
) (io.ReadCloser, error) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(bodyReader)
		if err != nil {
			return nil, fmt.Errorf("problem trying to create gzip reader: %w", err)
		}
		return gr, nil
	}
	return io.NopCloser(bodyReader), nil
}

// copyResponseForDebug reads the entire response body into memory
// and returns both the bytes and a reader positioned at the start.
// This allows the response to be decoded while preserving a copy for debug logging.
func copyResponseForDebug(r io.Reader) ([]byte, *bytes.Reader, error) {
	respBody, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	return respBody, bytes.NewReader(respBody), nil
}

// Response is the decoded top-level GraphQL response envelope, per the spec's
// {data, errors, extensions} shape.
//
//   - Data holds the raw "data" JSON for later decoding; nil when absent.
//   - Errors holds the "errors" array; nil/empty when the operation produced
//     no GraphQL errors.
//   - Extensions holds the top-level "extensions" map of server-provided
//     metadata (tracing, query cost, rate limits, request IDs, ...); nil when
//     the server omits it.
type Response struct {
	Data       json.RawMessage
	Errors     Errors
	Extensions map[string]any
}

// DecodeResponse decodes a GraphQL JSON response envelope into a *Response.
//
// The returned Errors is non-nil only on a local decode failure (the body is
// not valid JSON); in that case the *Response is nil. A well-formed envelope
// returns (resp, nil) even when it carries GraphQL errors — those live in
// resp.Errors, and any top-level metadata in resp.Extensions.
func (c *Client) DecodeResponse(reader io.Reader) (*Response, Errors) {
	var out struct {
		Data       json.RawMessage `json:"data"`
		Errors     Errors          `json:"errors"`
		Extensions map[string]any  `json:"extensions"`
	}

	if err := json.NewDecoder(reader).Decode(&out); err != nil {
		return nil, Errors{newJSONDecodeError(err)}
	}

	return &Response{
		Data:       out.Data,
		Errors:     out.Errors,
		Extensions: out.Extensions,
	}, nil
}

// processResponse unmarshals env.Data into v and combines the server's GraphQL
// errors with any local unmarshal failure into a single returned error.
//
// transportErrs carries build/round-trip/decode failures and short-circuits:
// when non-empty, env is nil and those errors are returned as-is. Otherwise the
// returned error is env.Errors plus a decode error if v could not be populated;
// env.Errors itself is left as the pure server-side error set.
func (c *Client) processResponse(
	v any,
	env *Response,
	resp *http.Response,
	respBody []byte,
	transportErrs Errors,
) error {
	if len(transportErrs) > 0 {
		return transportErrs
	}
	if env == nil {
		return nil
	}

	errs := env.Errors
	if len(env.Data) > 0 {
		if err := decode.UnmarshalGraphQL(env.Data, v); err != nil {
			we := c.decorate(
				newGraphQLDecodeError(err),
				errorContext{resp: resp, respBody: respBody},
			)
			errs = append(errs, we)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
