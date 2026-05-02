package decode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/llehouerou/gqlclient/internal/fragments"
	"github.com/llehouerou/gqlclient/internal/reflectutil"
	"github.com/llehouerou/gqlclient/types"
)

// fieldInfo holds information about a field discovered during JSON object unmarshaling.
type fieldInfo struct {
	field         reflect.Value
	isScalar      bool
	fragmentMatch bool
}

// decodeObjectKey handles the processing of an object key and its value.
// This is called when we're inside an object and see the next key.
func (d *decoder) decodeObjectKey(
	key string,
	rawMessageValue reflect.Value,
) (any, error) {
	// Track current key for typename capture
	d.currentKey = key

	// First pass: find all fields and check if any matching fragment has it
	fields, hasMatchingFragmentWithField, rawMessage := d.findFieldsForKey(
		key,
		rawMessageValue,
	)

	// Second pass: decide which fields to use and push to value stacks
	someFieldExist, isScalar := d.selectAndPushFields(
		fields,
		hasMatchingFragmentWithField,
	)

	if !someFieldExist {
		return nil, fmt.Errorf(
			"struct field for %q doesn't exist in any of %v places to unmarshal",
			key,
			d.vs.len(),
		)
	}

	// Read the next token based on field type
	return d.readNextToken(rawMessage, isScalar)
}

// findFieldsForKey discovers fields matching the given key across all value stacks.
// It returns:
// - fields: slice of fieldInfo (one per stack)
// - hasMatchingFragmentWithField: whether any matching fragment has the field
// - rawMessage: whether any field is of json.RawMessage type
func (d *decoder) findFieldsForKey(
	key string,
	rawMessageValue reflect.Value,
) ([]fieldInfo, bool, bool) {
	fields := make([]fieldInfo, d.vs.len())
	hasMatchingFragmentWithField := false
	rawMessage := false

	for i := range d.vs.len() {
		v := d.vs.top(i)
		v = reflectutil.UnwrapToConcreteValue(v)

		var f reflect.Value
		var scalar bool

		switch v.Kind() {
		case reflect.Struct:
			f, scalar = fieldByGraphQLName(v, key)
			if f.IsValid() {
				// Check if this is a wrapper type and unwrap if needed
				unwrapped := reflectutil.UnwrapValueField(f)
				if unwrapped.IsValid() {
					// Wrapper type detected. Unmarshal directly into
					// the unwrapped Value field, bypassing the wrapper.
					f = unwrapped
				}
				// Check for special embedded json
				if f.Type() == rawMessageValue.Type() {
					rawMessage = true
				}
			}
		case reflect.Slice:
			f = orderedMapValueByGraphQLName(v, key)
			// For ordered maps, we need to be careful about unwrapping
			// Unwrap pointers, but keep interfaces as they are
			// (unmarshalValue can handle interface types)
			for f.Kind() == reflect.Ptr {
				f = f.Elem()
			}
		}

		fragmentMatch := true
		fragType := d.vs.fragmentType(i)
		if fragType != "" && d.currentTypename != "" {
			fragmentMatch = fragType == d.currentTypename
		}

		fields[i] = fieldInfo{
			field:         f,
			isScalar:      scalar,
			fragmentMatch: fragmentMatch,
		}

		if f.IsValid() && fragmentMatch {
			hasMatchingFragmentWithField = true
		}
	}

	return fields, hasMatchingFragmentWithField, rawMessage
}

// selectAndPushFields processes discovered fields, filtering by fragment matching,
// and pushes selected fields to the value stacks.
// Returns (someFieldExist, isScalar) flags.
func (d *decoder) selectAndPushFields(
	fields []fieldInfo,
	hasMatchingFragmentWithField bool,
) (someFieldExist, isScalar bool) {
	for i := range d.vs.len() {
		f := fields[i].field

		if f.IsValid() {
			someFieldExist = true
			if fields[i].isScalar {
				isScalar = true
			}
		}

		// Skip this field if:
		// 1. It's from a non-matching fragment AND
		// 2. A matching fragment also has this field
		if f.IsValid() && !fields[i].fragmentMatch &&
			hasMatchingFragmentWithField {
			f = reflect.Value{}
		}

		d.vs.push(i, f)
	}

	return someFieldExist, isScalar
}

// readNextToken reads the next JSON token based on whether the field is raw or scalar.
// For raw/scalar fields, it decodes the entire value as json.RawMessage.
// For regular fields, it returns the next token for further processing.
func (d *decoder) readNextToken(rawMessage, isScalar bool) (any, error) {
	if rawMessage || isScalar {
		// Read the next complete object from the json stream
		var data json.RawMessage
		err := d.tokenizer.Decode(&data)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	// We've just consumed the current token, which was the key.
	// Read the next token, which should be the value,
	// and let the rest of code process it.
	tok, err := d.tokenizer.Token()
	if errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected end of JSON input")
	} else if err != nil {
		return nil, err
	}

	return tok, nil
}

// decodeObjectStart handles the start of a JSON object ('{' token).
// It initializes values and discovers GraphQL fragments and embedded structs.
func (d *decoder) decodeObjectStart() {
	d.pushState('{')

	// frontier is a BFS queue: the prefix is seeded initial state,
	// appends below extend the queue.
	frontier := make([]reflect.Value, d.vs.len())
	for i := range d.vs.len() {
		v := d.vs.top(i)
		frontier[i] = v
		// Initialize only the immediate nil pointer, not recursively.
		// Deeper levels are initialized as needed during field processing.
		// Test coverage: TestUnmarshalGraphQL_nilPointerToWrapper (includes **Wrapper)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			v.Set(reflect.New(v.Type().Elem())) // v = new(T).
		}
	}
	// Find GraphQL fragments/embedded structs recursively, adding to frontier
	// as new ones are discovered and exploring them further.
	for len(frontier) > 0 {
		v := frontier[0]
		frontier = frontier[1:]
		v = reflectutil.UnwrapToConcreteValue(v)

		if v.Kind() == reflect.Struct {
			for i := range v.NumField() {
				field := v.Type().Field(i)
				if fragments.IsStructField(field) {
					// Add GraphQL fragment and track its typename
					tag, _ := field.Tag.Lookup(types.GraphQLTag)
					d.vs.addStack(v.Field(i), fragments.ExtractTypename(tag))
					frontier = append(frontier, v.Field(i)) //nolint:makezero // BFS queue extension; see frontier init above
				} else if field.Anonymous {
					// Add embedded struct (not a fragment)
					d.vs.addStack(v.Field(i), "")
					frontier = append(frontier, v.Field(i)) //nolint:makezero // BFS queue extension; see frontier init above
				}
			}
		} else if isOrderedMap(v) {
			for i := range v.Len() {
				pair := v.Index(i)
				key, val := pair.Index(0), pair.Index(1)
				keyStr, ok := key.Interface().(string)
				if !ok {
					continue
				}
				if fragments.IsTag(keyStr) {
					// Add GraphQL fragment and track its typename
					d.vs.addStack(val, fragments.ExtractTypename(keyStr))
					frontier = append(frontier, val) //nolint:makezero // BFS queue extension; see frontier init above
				}
			}
		}
	}
}

// handleObjectEnd handles the end of a JSON object ('}' delimiter).
// It pops all accumulated values and the object state from the stack.
func (d *decoder) handleObjectEnd() {
	d.vs.popAll()
	d.popState()
}
