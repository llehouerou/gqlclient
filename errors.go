package graphql

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Error codes used throughout the client.
const (
	ErrRequestError  = "request_error"
	ErrJsonEncode    = "json_encode_error"
	ErrJsonDecode    = "json_decode_error"
	ErrGraphQLEncode = "graphql_encode_error"
	ErrGraphQLDecode = "graphql_decode_error"
)

// Errors represents the "errors" array in a response from a GraphQL server.
// If returned via error interface, the slice is expected to contain at least 1 element.
//
// Specification: https://facebook.github.io/graphql/#sec-Errors.
type Errors []Error

// Error represents a single error from a GraphQL response.
type Error struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
	Locations  []struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"locations"`
}

// RequestInfo contains HTTP request information stored in error extensions.
type RequestInfo struct {
	Headers http.Header
	Body    string
}

// ResponseInfo contains HTTP response information stored in error extensions.
type ResponseInfo struct {
	Headers http.Header
	Body    string
}

// InternalExtensions contains internal debugging information stored in error
// extensions. This information is added when debug mode is enabled.
type InternalExtensions struct {
	Request  *RequestInfo
	Response *ResponseInfo
	Error    error
}

// Error implements error interface.
func (e Error) Error() string {
	return fmt.Sprintf("Message: %s, Locations: %+v", e.Message, e.Locations)
}

// Error implements error interface.
func (e Errors) Error() string {
	b := strings.Builder{}
	for _, err := range e {
		b.WriteString(err.Error())
	}
	return b.String()
}

// GetCode returns the error code from the extensions, or an empty string if
// not present.
func (e Error) GetCode() string {
	if e.Extensions == nil {
		return ""
	}
	code, ok := e.Extensions["code"].(string)
	if !ok {
		return ""
	}
	return code
}

// GetInternalExtensions returns the typed internal extensions, or nil if not
// present. Internal extensions contain debugging information added when debug
// mode is enabled.
func (e Error) GetInternalExtensions() *InternalExtensions {
	if e.Extensions == nil {
		return nil
	}

	internal, ok := e.Extensions["internal"].(map[string]any)
	if !ok {
		return nil
	}

	ext := &InternalExtensions{}

	if req, ok := internal["request"].(map[string]any); ok {
		ext.Request = &RequestInfo{}
		if headers, ok := req["headers"].(http.Header); ok {
			ext.Request.Headers = headers
		}
		if body, ok := req["body"].(string); ok {
			ext.Request.Body = body
		}
	}

	if resp, ok := internal["response"].(map[string]any); ok {
		ext.Response = &ResponseInfo{}
		if headers, ok := resp["headers"].(http.Header); ok {
			ext.Response.Headers = headers
		}
		if body, ok := resp["body"].(string); ok {
			ext.Response.Body = body
		}
	}

	if err, ok := internal["error"].(error); ok {
		ext.Error = err
	}

	return ext
}

// getInternalExtension retrieves the internal extension map, creating it if needed.
func (e Error) getInternalExtension() map[string]any {
	if e.Extensions == nil {
		return make(map[string]any)
	}

	if ex, ok := e.Extensions["internal"]; ok {
		if m, ok := ex.(map[string]any); ok {
			return m
		}
	}

	return make(map[string]any)
}

// newError creates a new Error with the given code and underlying error.
// The underlying error is stored in the extensions for debugging.
func newError(code string, err error) Error {
	return Error{
		Message: err.Error(),
		Extensions: map[string]any{
			"code": code,
		},
	}
}

// newSimpleErrors creates an Errors slice with a single error, wrapping the
// given error with the specified code. This is a convenience method for simple
// error cases that don't have request/response context.
func newSimpleErrors(code string, err error) Errors {
	return Errors{newError(code, err)}
}

// Convenience factory functions for common error types with predefined codes.
// These functions standardize error creation and make the error type explicit.

// newJSONDecodeError creates an error for JSON decoding failures.
// Used when the HTTP response body cannot be parsed as valid JSON.
func newJSONDecodeError(err error) Error {
	return newError(ErrJsonDecode, err)
}

// newGraphQLDecodeError creates an error for GraphQL response unmarshaling failures.
// Used when GraphQL JSON cannot be unmarshaled into the target Go struct.
func newGraphQLDecodeError(err error) Error {
	return newError(ErrGraphQLDecode, err)
}

// withDebugInfo adds debug information to the error's internal extensions.
// It reads the body from bodyReader and stores it along with headers under the
// specified infoType key ("request" or "response").
func (e Error) withDebugInfo(
	infoType string,
	headers http.Header,
	bodyReader io.Reader,
) Error {
	internal := e.getInternalExtension()
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		internal["error"] = err
	} else {
		internal[infoType] = map[string]any{
			"headers": headers,
			"body":    string(bodyBytes),
		}
	}

	if e.Extensions == nil {
		e.Extensions = make(map[string]any)
	}
	e.Extensions["internal"] = internal
	return e
}

// withRequest adds HTTP request information to the error's debug extensions.
func (e Error) withRequest(req *http.Request, bodyReader io.Reader) Error {
	return e.withDebugInfo("request", req.Header, bodyReader)
}

// withResponse adds HTTP response information to the error's debug extensions.
func (e Error) withResponse(res *http.Response, bodyReader io.Reader) Error {
	return e.withDebugInfo("response", res.Header, bodyReader)
}

// DecorateError decorates an error with request/response information if debug
// mode is enabled. This helper method centralizes the error decoration logic
// and eliminates repetitive debug checks throughout the codebase.
func (c *Client) DecorateError(
	err Error,
	req *http.Request,
	resp *http.Response,
	reqBody,
	respBody io.Reader,
) Error {
	if !c.debug {
		return err
	}

	if req != nil && reqBody != nil {
		err = err.withRequest(req, reqBody)
	}

	if resp != nil && respBody != nil {
		err = err.withResponse(resp, respBody)
	}

	return err
}

// NewRequestError creates a new error with the given code and decorates it with
// request/response information if debug mode is enabled. This is a convenience
// method that combines error creation and decoration in one step.
func (c *Client) NewRequestError(
	code string,
	err error,
	req *http.Request,
	resp *http.Response,
	reqBody,
	respBody io.Reader,
) Error {
	e := newError(code, err)
	return c.DecorateError(e, req, resp, reqBody, respBody)
}
