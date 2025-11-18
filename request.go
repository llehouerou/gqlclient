package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// BuildRequest constructs an HTTP request with JSON body for a GraphQL operation.
// It returns the HTTP request and the request body bytes (useful for error decoration).
func (c *Client) BuildRequest(
	ctx context.Context,
	query string,
	variables any,
) (*http.Request, []byte, error) {
	// Normalize empty variable maps to nil
	if !hasVariables(variables) {
		variables = nil
	}
	in := struct {
		Query     string `json:"query"`
		Variables any    `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: variables,
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(in)
	if err != nil {
		return nil, nil, err
	}

	reqBody := buf.Bytes()
	reqReader := bytes.NewReader(reqBody)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.url,
		reqReader,
	)
	if err != nil {
		return nil, reqBody, err
	}
	request.Header.Add("Content-Type", "application/json")

	if c.requestModifier != nil {
		c.requestModifier(request)
	}

	return request, reqBody, nil
}

// ExecuteRequest executes an HTTP request and handles gzip decompression.
// It returns the HTTP response and a reader for the (possibly decompressed) body.
func (c *Client) ExecuteRequest(
	req *http.Request,
) (*http.Response, io.Reader, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	r := resp.Body

	// Handle gzip decompression
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := handleGzipResponse(resp, r)
		if err != nil {
			_ = resp.Body.Close()
			return nil, nil, err
		}
		// Note: caller is responsible for closing both gr and resp.Body
		r = gr
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(r)
		_ = resp.Body.Close()
		return nil, nil, fmt.Errorf("%v; body: %q", resp.Status, body)
	}

	return resp, r, nil
}

// request is the common method that sends a graphql request
func (c *Client) request(
	ctx context.Context,
	query string,
	variables any,
) ([]byte, *http.Response, io.Reader, Errors) {
	// Build HTTP request with JSON body
	request, reqBody, err := c.BuildRequest(ctx, query, variables)
	if err != nil {
		e := c.NewRequestError(
			ErrRequestError,
			fmt.Errorf("problem constructing request: %w", err),
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{e}
	}

	// Execute HTTP request
	resp, err := c.httpClient.Do(request)
	if err != nil {
		e := c.NewRequestError(
			ErrRequestError,
			err,
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{e}
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := c.NewRequestError(
			ErrRequestError,
			fmt.Errorf("%v; body: %q", resp.Status, body),
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{err}
	}

	// Handle gzip decompression
	r, err := handleGzipResponse(resp, resp.Body)
	if err != nil {
		return nil, nil, nil, newSimpleErrors(ErrJsonDecode, err)
	}
	defer func() { _ = r.Close() }()

	// Copy response body for debugging if needed
	var respBody []byte
	var respReader *bytes.Reader
	if c.debug {
		var debugReader io.Reader
		respBody, debugReader, err = copyResponseForDebug(r)
		if err != nil {
			return nil, nil, nil, newSimpleErrors(ErrJsonDecode, err)
		}
		respReader = debugReader.(*bytes.Reader)
		r = io.NopCloser(respReader)
	}

	// Decode GraphQL response
	rawData, gqlErrors := c.DecodeResponse(r)

	if c.debug {
		if respReader != nil {
			_, _ = respReader.Seek(
				0,
				io.SeekStart,
			) // Ignore seek errors for debug logging
		}
	}

	// Handle JSON decode errors
	if len(gqlErrors) > 0 {
		// Check if it's a decode error (has ErrJsonDecode code)
		if code, ok := gqlErrors[0].Extensions["code"].(string); ok &&
			code == ErrJsonDecode {
			we := c.NewRequestError(
				ErrJsonDecode,
				fmt.Errorf("%s", gqlErrors[0].Message),
				request,
				resp,
				bytes.NewReader(reqBody),
				bytes.NewReader(respBody),
			)
			return nil, nil, nil, Errors{we}
		}

		// Handle GraphQL errors - decorate first error if debug mode
		if c.debug &&
			(gqlErrors[0].Extensions == nil || gqlErrors[0].Extensions["request"] == nil) {
			gqlErrors[0] = c.DecorateError(
				gqlErrors[0],
				request,
				resp,
				bytes.NewReader(reqBody),
				bytes.NewReader(respBody),
			)
		}

		return rawData, resp, respReader, gqlErrors
	}

	return rawData, resp, respReader, nil
}
