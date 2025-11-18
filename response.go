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
func copyResponseForDebug(r io.Reader) ([]byte, io.Reader, error) {
	respBody, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	return respBody, bytes.NewReader(respBody), nil
}

// DecodeResponse decodes a GraphQL JSON response into raw data and errors.
// It returns the raw data bytes (if present) and any GraphQL errors.
func (c *Client) DecodeResponse(reader io.Reader) ([]byte, Errors) {
	var out struct {
		Data   *json.RawMessage
		Errors Errors
	}

	err := json.NewDecoder(reader).Decode(&out)
	if err != nil {
		return nil, newSimpleErrors(ErrJsonDecode, err)
	}

	var rawData []byte
	if out.Data != nil && len(*out.Data) > 0 {
		rawData = *out.Data
	}

	if len(out.Errors) > 0 {
		return rawData, out.Errors
	}

	return rawData, nil
}

// processResponse handles the unmarshaling of response data and error aggregation.
func (c *Client) processResponse(
	v any,
	data []byte,
	resp *http.Response,
	respBuf io.Reader,
	errs Errors,
) error {
	if len(data) > 0 {
		err := decode.UnmarshalGraphQL(data, v)
		if err != nil {
			we := c.DecorateError(
				newError(ErrGraphQLDecode, err),
				nil,
				resp,
				nil,
				respBuf,
			)
			errs = append(errs, we)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
