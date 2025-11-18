// Package jsonutil provides a function for decoding JSON
// into a GraphQL query data structure.
package decode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/llehouerou/gqlclient/types"
)

const (
	// maxTemplateSliceSize is the maximum number of items allowed in a template slice.
	// Template slices should contain either 0 items (use zero value) or 1 item (use as template).
	// Having more than 1 item is ambiguous and not supported.
	maxTemplateSliceSize = 1
)

// UnmarshalGraphQL parses the JSON-encoded GraphQL response data and stores
// the result in the GraphQL query data structure pointed to by v.
//
// The implementation is created on top of the JSON tokenizer available
// in "encoding/json".Decoder.
//
// # Wrapper Types
//
// UnmarshalGraphQL supports transparent unwrapping of container types that
// implement the GetGraphQLWrapped() method. This allows GraphQL schemas with
// wrapper/container patterns to be cleanly represented in Go.
//
// Convention: Any type implementing GetGraphQLWrapped() MUST have an exported
// field named "Value" that holds the wrapped data. During unmarshaling, the
// library will detect the GetGraphQLWrapped() method and unmarshal JSON data
// directly into the Value field, bypassing the wrapper.
//
// Rationale: The GetGraphQLWrapped() method returns a value (used during query
// construction for reflection), but unmarshaling requires a writable field
// reference. The "Value" field provides this writable target.
//
// Example:
//
//	type Wrapper[T any] struct {
//	    Value T  // REQUIRED: Must be named reflectutil.WrapperFieldName ("Value")
//	}
//	func (w Wrapper[T]) GetGraphQLWrapped() T { return w.Value }
func UnmarshalGraphQL(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := (&decoder{tokenizer: dec}).Decode(v)
	if err != nil {
		return err
	}
	tok, err := dec.Token()
	switch err {
	case io.EOF:
		// Expect to get io.EOF. There shouldn't be any more
		// tokens left after we've decoded v successfully.
		return nil
	case nil:
		return fmt.Errorf("invalid token '%v' after top-level value", tok)
	default:
		return err
	}
}

// decoder is a JSON decoder that performs custom unmarshaling behavior
// for GraphQL query data structures. It's implemented on top of a JSON tokenizer.
type decoder struct {
	tokenizer interface {
		Token() (json.Token, error)
		Decode(v any) error
	}

	// Stack of what part of input JSON we're in the middle of - objects, arrays.
	parseState []json.Delim

	// vs manages stacks of values to unmarshal into, along with their fragment types.
	vs valueStack

	// currentTypename holds the __typename value for the current object being unmarshaled.
	// This is used to filter inline fragments so only the matching fragment is populated.
	currentTypename string

	// currentKey holds the current JSON key being processed, used to capture __typename.
	currentKey string
}

// Decode decodes a single JSON value from d.tokenizer into v.
func (d *decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot decode into non-pointer %T", v)
	}
	d.vs = valueStack{
		values:        []stack{{rv.Elem()}},
		fragmentTypes: []string{""}, // Root is not a fragment
	}
	return d.decode()
}

// decode decodes a single JSON value from d.tokenizer into d.vs.
func (d *decoder) decode() error {
	rawMessageValue := reflect.ValueOf(json.RawMessage{})

	// The loop invariant is that the top of each d.vs stack
	// is where we try to unmarshal the next JSON value we see.
	for d.vs.len() > 0 {
		var tok any
		tok, err := d.tokenizer.Token()

		if err == io.EOF {
			return errors.New("unexpected end of JSON input")
		} else if err != nil {
			return err
		}

		switch {

		// Are we inside an object and seeing next key (rather than end of object)?
		case d.state() == '{' && tok != json.Delim('}'):
			key, ok := tok.(string)
			if !ok {
				return errors.New("unexpected non-key in JSON input")
			}

			tok, err = d.decodeObjectKey(key, rawMessageValue)
			if err != nil {
				return err
			}

		// Are we inside an array and seeing next value (rather than end of array)?
		case d.state() == '[' && tok != json.Delim(']'):
			err = d.decodeArrayValue()
			if err != nil {
				return err
			}
		}

		switch tok := tok.(type) {
		case string, json.Number, bool, nil, json.RawMessage:
			// Scalar value.
			err := d.decodeScalarValue(tok)
			if err != nil {
				return err
			}

		case json.Delim:
			// Delimiter (object/array start or end).
			err := d.handleDelimiter(tok)
			if err != nil {
				return err
			}

		default:
			return errors.New("unexpected token in JSON input")
		}
	}
	return nil
}

// decodeScalarValue handles decoding of scalar values
// (string, number, bool, nil, json.RawMessage).
func (d *decoder) decodeScalarValue(tok any) error {
	// Capture __typename value to filter inline fragments
	if d.currentKey == types.TypenameField {
		if typename, ok := tok.(string); ok {
			d.currentTypename = typename
		}
	}

	for i := 0; i < d.vs.len(); i++ {
		v := d.vs.top(i)
		if !v.IsValid() {
			continue
		}
		err := unmarshalValue(tok, v)
		if err != nil {
			return err
		}
	}
	d.vs.popAll()
	return nil
}

// handleDelimiter handles JSON delimiter tokens ('{', '[', '}', ']').
// It dispatches to the appropriate handler based on the delimiter type.
func (d *decoder) handleDelimiter(tok json.Delim) error {
	switch tok {
	case '{':
		// Start of object.
		d.decodeObjectStart()
		return nil
	case '[':
		// Start of array.
		return d.decodeArrayStart()
	case '}':
		// End of object.
		d.handleObjectEnd()
		return nil
	case ']':
		// End of array.
		d.handleArrayEnd()
		return nil
	default:
		return errors.New("unexpected delimiter in JSON input")
	}
}

// pushState pushes a new parse state s onto the stack.
func (d *decoder) pushState(s json.Delim) {
	d.parseState = append(d.parseState, s)
}

// popState pops a parse state (already obtained) off the stack.
// The stack must be non-empty.
func (d *decoder) popState() {
	d.parseState = d.parseState[:len(d.parseState)-1]
}

// state reports the parse state on top of stack, or 0 if empty.
func (d *decoder) state() json.Delim {
	if len(d.parseState) == 0 {
		return 0
	}
	return d.parseState[len(d.parseState)-1]
}

// unmarshalValue unmarshals JSON value into v.
// v must be addressable and not obtained by the use of unexported
// struct fields, otherwise unmarshalValue will panic.
func unmarshalValue(value any, v reflect.Value) error {
	// This function uses json.Marshal + json.Unmarshal to convert JSON tokens
	// (from the tokenizer) into Go values. While this could be optimized with
	// direct type conversion for simple cases (string, number, bool), the current
	// approach handles all edge cases correctly (custom UnmarshalJSON, etc.).
	// TODO: Profile to measure impact, then consider optimizing hot paths if needed.
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ty := v.Type()
	if ty.Kind() == reflect.Interface {
		if !v.Elem().IsValid() {
			return json.Unmarshal(b, v.Addr().Interface())
		}
		ty = v.Elem().Type()
	}
	newVal := reflect.New(ty)
	err = json.Unmarshal(b, newVal.Interface())
	if err != nil {
		return err
	}
	v.Set(newVal.Elem())
	return nil
}
