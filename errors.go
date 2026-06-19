package graphql

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Error code strings stored in Error.Extensions["code"]. These are the
// stable identifiers serialized in the GraphQL extensions map and used
// for cross-process or cross-language consumers.
//
// Most Go callers should reach for the matching sentinel error values
// (ErrRequest, ErrJSONEncode, ...) with errors.Is rather than comparing
// these strings by hand.
const (
	ErrCodeRequest       = "request_error"
	ErrCodeJSONEncode    = "json_encode_error"
	ErrCodeJSONDecode    = "json_decode_error"
	ErrCodeGraphQLEncode = "graphql_encode_error"
	ErrCodeGraphQLDecode = "graphql_decode_error"
)

// Sentinel errors usable with errors.Is to detect specific failure
// modes. errors.Is(err, ErrJSONDecode) is true whenever any Error
// in the chain has Extensions["code"] == ErrCodeJSONDecode, including
// errors generated locally by this library and errors returned by a
// GraphQL server using the same codes.
//
// Example:
//
//	if err := client.Query(ctx, &q, nil); err != nil {
//	    switch {
//	    case errors.Is(err, graphql.ErrJSONDecode):
//	        // server returned non-JSON or malformed JSON
//	    case errors.Is(err, graphql.ErrRequest):
//	        // HTTP transport failure (DNS, TCP, TLS, status != 200)
//	    default:
//	        // server-level GraphQL errors; inspect via errors.As
//	        var gqlErrs graphql.Errors
//	        if errors.As(err, &gqlErrs) { ... }
//	    }
//	}
var (
	ErrRequest       error = &codeError{code: ErrCodeRequest, msg: "request error"}
	ErrJSONEncode    error = &codeError{code: ErrCodeJSONEncode, msg: "json encode error"}
	ErrJSONDecode    error = &codeError{code: ErrCodeJSONDecode, msg: "json decode error"}
	ErrGraphQLEncode error = &codeError{code: ErrCodeGraphQLEncode, msg: "graphql encode error"}
	ErrGraphQLDecode error = &codeError{code: ErrCodeGraphQLDecode, msg: "graphql decode error"}
)

// codeError is the type used for the package-level sentinel error
// values. It is unexported because users compare with errors.Is rather
// than constructing instances directly.
type codeError struct {
	code string
	msg  string
}

func (e *codeError) Error() string { return e.msg }

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

	// underlying is the wrapped cause when this Error was produced
	// locally (e.g., a *json.SyntaxError surfaced as an ErrJSONDecode).
	// It is not serialized; Errors decoded from a server response
	// always have underlying == nil.
	underlying error
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

// Is reports whether target is one of the package-level sentinel
// errors and the receiver carries the same code in Extensions["code"].
// This makes errors.Is(err, ErrJSONDecode) work whether the error was
// generated locally or arrived in a server response with that code.
func (e Error) Is(target error) bool {
	var ce *codeError
	if errors.As(target, &ce) {
		return e.GetCode() == ce.code
	}
	return false
}

// Unwrap returns the wrapped cause when this Error was produced
// locally (e.g., a *json.SyntaxError under an ErrJSONDecode). Returns
// nil for Errors decoded from a server response.
func (e Error) Unwrap() error {
	return e.underlying
}

// Error implements error interface.
func (e Errors) Error() string {
	b := strings.Builder{}
	for _, err := range e {
		b.WriteString(err.Error())
	}
	return b.String()
}

// Is reports whether any constituent Error matches target. Together with
// Unwrap, this lets errors.Is and errors.As walk the slice transparently.
func (e Errors) Is(target error) bool {
	for i := range e {
		if errors.Is(e[i], target) {
			return true
		}
	}
	return false
}

// Unwrap exposes the constituent Error values to the standard
// errors package, so errors.As(err, &someErr) finds the first match
// inside an Errors slice.
func (e Errors) Unwrap() []error {
	out := make([]error, len(e))
	for i := range e {
		out[i] = e[i]
	}
	return out
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
// The underlying error is retained on the Error so callers can recover it
// via errors.As; its message is also placed in Error.Message and the
// code in Error.Extensions["code"] for serialization.
func newError(code string, err error) Error {
	return Error{
		Message: err.Error(),
		Extensions: map[string]any{
			"code": code,
		},
		underlying: err,
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
	return newError(ErrCodeJSONDecode, err)
}

// newGraphQLDecodeError creates an error for GraphQL response unmarshaling failures.
// Used when GraphQL JSON cannot be unmarshaled into the target Go struct.
func newGraphQLDecodeError(err error) Error {
	return newError(ErrCodeGraphQLDecode, err)
}

// errorContext carries the HTTP request/response material that debug-mode
// decoration attaches to an error. Any field may be zero; reqBody/respBody are
// the already-buffered bodies (respBody is only populated in debug mode).
type errorContext struct {
	req      *http.Request
	reqBody  []byte
	resp     *http.Response
	respBody []byte
}

// withDebugInfo adds debug information to the error's internal extensions,
// storing body alongside headers under the infoType key ("request" or
// "response").
func (e Error) withDebugInfo(
	infoType string,
	headers http.Header,
	body []byte,
) Error {
	internal := e.getInternalExtension()
	internal[infoType] = map[string]any{
		"headers": headers,
		"body":    string(body),
	}

	if e.Extensions == nil {
		e.Extensions = make(map[string]any)
	}
	e.Extensions["internal"] = internal
	return e
}

// withRequest adds HTTP request information to the error's debug extensions.
func (e Error) withRequest(req *http.Request, body []byte) Error {
	return e.withDebugInfo("request", req.Header, body)
}

// withResponse adds HTTP response information to the error's debug extensions.
func (e Error) withResponse(res *http.Response, body []byte) Error {
	return e.withDebugInfo("response", res.Header, body)
}

// decorate is the single owner of the debug-decoration policy: when debug mode
// is enabled it attaches whatever request/response context ctx carries. A
// no-op when debug is off, so callers never gate on c.debug themselves.
func (c *Client) decorate(e Error, ctx errorContext) Error {
	if !c.debug {
		return e
	}

	if ctx.req != nil && ctx.reqBody != nil {
		e = e.withRequest(ctx.req, ctx.reqBody)
	}

	if ctx.resp != nil && ctx.respBody != nil {
		e = e.withResponse(ctx.resp, ctx.respBody)
	}

	return e
}

// newRequestError creates a new error with the given code and decorates it
// with ctx when debug mode is enabled, in one step.
func (c *Client) newRequestError(
	code string,
	err error,
	ctx errorContext,
) Error {
	return c.decorate(newError(code, err), ctx)
}
