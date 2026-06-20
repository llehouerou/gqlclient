package decode

import (
	"fmt"
	"reflect"

	"github.com/llehouerou/gqlclient/internal/reflectutil"
)

// decodeArrayValue handles processing an array value by appending a new element
// to slices in the decoder's value stack.
func (d *decoder) decodeArrayValue() error {
	someSliceExist := false
	for i := range d.vs.len() {
		v := d.vs.top(i)
		v = reflectutil.UnwrapToConcreteValue(v)

		// Check if this is a wrapper type (has a `wrapped:"true"` field).
		// If so, unwrap to get the actual slice field.
		if v.IsValid() {
			unwrapped := reflectutil.UnwrapValueField(v)
			if unwrapped.IsValid() {
				v = unwrapped
			}
		}

		var f reflect.Value
		if v.Kind() == reflect.Slice {
			// we want to append the template item copy
			// so that all the inner structure gets preserved
			copied, err := copyTemplate(v.Index(0))
			if err != nil {
				return fmt.Errorf("failed to copy template: %w", err)
			}
			v.Set(reflect.Append(v, copied)) // v = append(v, T).
			f = v.Index(v.Len() - 1)
			someSliceExist = true
		}
		d.vs.push(i, f)
	}
	if !someSliceExist {
		return fmt.Errorf(
			"slice doesn't exist in any of %v places to unmarshal",
			d.vs.len(),
		)
	}
	return nil
}

// decodeArrayStart handles the start of a JSON array ('[' token).
// It initializes slices and ensures they have a template element.
func (d *decoder) decodeArrayStart() error {
	d.pushState('[')

	for i := range d.vs.len() {
		v := d.vs.top(i)
		// Initialize nil pointers before unwrapping.
		// This handles cases like *[]string where the pointer is nil.
		// Test coverage: TestUnmarshalGraphQL_pointerToSlice
		if v.Kind() == reflect.Ptr && v.IsNil() {
			v.Set(reflect.New(v.Type().Elem())) // v = new(T).
		}

		// Reset slice to empty (in case it had non-zero initial value).
		v = reflectutil.UnwrapToConcreteValue(v)

		if v.Kind() != reflect.Slice {
			continue
		}
		newSlice := reflect.MakeSlice(v.Type(), 0, 0) // v = make(T, 0, 0).
		switch v.Len() {
		case 0:
			// if there is no template we need to create one so that we can
			// handle both cases (with or without a template) in the same way
			newSlice = reflect.Append(newSlice, reflect.Zero(v.Type().Elem()))
		case maxTemplateSliceSize:
			// if there is a template, we need to keep it at index 0
			newSlice = reflect.Append(newSlice, v.Index(0))
		default:
			if v.Len() > maxTemplateSliceSize {
				return fmt.Errorf(
					"template slice can only have %d item, got %d",
					maxTemplateSliceSize,
					v.Len(),
				)
			}
		}
		v.Set(newSlice)
	}
	return nil
}

// handleArrayEnd handles the end of a JSON array (']' delimiter).
// It removes array templates and pops all accumulated values and the array state.
func (d *decoder) handleArrayEnd() {
	d.vs.popLeftArrayTemplates()
	d.vs.popAll()
	d.popState()
}

// copyTemplate creates a copy of a template value for array elements.
// For ordered maps, it makes a deep copy. For other types, returns the template as-is.
func copyTemplate(template reflect.Value) (reflect.Value, error) {
	if isOrderedMap(template) {
		// copy slice if it's actually an ordered map
		return copyOrderedMap(template), nil
	}
	if template.Kind() == reflect.Map {
		return reflect.Value{}, fmt.Errorf(
			"unsupported template type `%v`, use [][2]any for ordered map instead",
			template.Type(),
		)
	}
	// don't need to copy regular slice
	return template, nil
}

// isOrderedMap checks if a value is an ordered map (slice of 2-element arrays).
func isOrderedMap(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	t := v.Type()
	return t.Kind() == reflect.Slice &&
		t.Elem().Kind() == reflect.Array &&
		t.Elem().Len() == 2
}

// copyOrderedMap creates a shallow copy of an ordered map.
func copyOrderedMap(m reflect.Value) reflect.Value {
	newMap := reflect.MakeSlice(m.Type(), 0, m.Len())
	for i := range m.Len() {
		pair := m.Index(i)
		newMap = reflect.Append(newMap, pair)
	}
	return newMap
}
