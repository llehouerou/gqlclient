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

	// Apply client-level headers before the request modifier so that the
	// modifier can override them when it needs to.
	for k, vs := range c.headers {
		for _, v := range vs {
			request.Header.Set(k, v)
		}
	}

	if c.requestModifier != nil {
		c.requestModifier(request)
	}

	return request, reqBody, nil
}

// executeRequest performs the HTTP round trip for req: it sends the request,
// transparently decompresses a gzip-encoded body, and rejects any non-200
// response. On success it returns the response and a reader over the
// (decompressed) body; the caller owns closing BOTH resp.Body and the reader.
//
// Decompression happens before the status check so that a gzip-encoded error
// body is readable rather than raw magic bytes. Every failure here — transport,
// non-200 status, or a corrupt gzip stream — is returned as a plain error
// describing the cause; classifying it into the library's error model is the
// orchestrator's job (see request). This is the single internal transport seam;
// see docs/adr/0001-public-api-surface.md for why it is not exported.
func (c *Client) executeRequest(
	req *http.Request,
) (*http.Response, io.ReadCloser, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	body, err := handleGzipResponse(resp, resp.Body)
	if err != nil {
		_ = resp.Body.Close() //nolint:errcheck // close on error path; nothing to do with the close error
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(body) //nolint:errcheck // partial body is best-effort context for the status error
		_ = body.Close()         //nolint:errcheck // close on error path; nothing to do with the close error
		_ = resp.Body.Close()    //nolint:errcheck // close on error path; nothing to do with the close error
		return nil, nil, fmt.Errorf("%v; body: %q", resp.Status, b)
	}

	return resp, body, nil
}

// request is the common method that sends a graphql request. It is the
// orchestrator: build the request, hand the round trip to executeRequest, then
// decode the response. Transport failures (anything executeRequest reports) are
// classified under ErrCodeRequest; GraphQL and decode errors flow out of
// DecodeResponse.
func (c *Client) request(
	ctx context.Context,
	query string,
	variables any,
) ([]byte, *http.Response, io.Reader, Errors) {
	// Build HTTP request with JSON body
	request, reqBody, err := c.BuildRequest(ctx, query, variables)
	if err != nil {
		e := c.NewRequestError(
			ErrCodeJSONEncode,
			fmt.Errorf("problem constructing request: %w", err),
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{e}
	}

	// Execute the HTTP round trip through the single transport seam.
	resp, body, err := c.executeRequest(request)
	if err != nil {
		e := c.NewRequestError(
			ErrCodeRequest,
			err,
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{e}
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // deferred close; nothing to do with the close error
	defer func() { _ = body.Close() }()      //nolint:errcheck // deferred close; nothing to do with the close error

	// Copy response body for debugging if needed
	var respBody []byte
	var respReader *bytes.Reader
	var decodeFrom io.Reader = body
	if c.debug {
		respBody, respReader, err = copyResponseForDebug(body)
		if err != nil {
			return nil, nil, nil, Errors{newJSONDecodeError(err)}
		}
		decodeFrom = respReader
	}

	// Decode GraphQL response
	rawData, gqlErrors := c.DecodeResponse(decodeFrom)

	if c.debug && respReader != nil {
		_, _ = respReader.Seek(0, io.SeekStart) //nolint:errcheck // *bytes.Reader.Seek(0, io.SeekStart) cannot fail
	}

	// Handle JSON decode errors
	if len(gqlErrors) > 0 {
		// Check if it's a decode error (has ErrCodeJSONDecode code)
		if code, ok := gqlErrors[0].Extensions["code"].(string); ok &&
			code == ErrCodeJSONDecode {
			we := c.NewRequestError(
				ErrCodeJSONDecode,
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
