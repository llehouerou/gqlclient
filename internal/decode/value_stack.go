package decode

import (
	"reflect"

	"github.com/llehouerou/gqlclient/internal/reflectutil"
)

type stack []reflect.Value

func (s stack) Top() reflect.Value {
	return s[len(s)-1]
}

func (s stack) Pop() stack {
	return s[:len(s)-1]
}

// valueStack manages multiple parallel stacks of values to unmarshal into,
// along with their associated fragment type names. This encapsulation prevents
// sync bugs between the parallel slices.
type valueStack struct {
	// values holds multiple stacks of reflect.Values.
	// Multiple stacks exist because we might unmarshal a single JSON value
	// into multiple GraphQL fragments or embedded structs simultaneously.
	values []stack

	// fragmentTypes tracks the typename for each stack in values.
	// Empty string means not a fragment or typename not applicable.
	// This is used to filter inline fragments during unmarshaling.
	fragmentTypes []string
}

// len returns the number of value stacks.
func (vs *valueStack) len() int {
	return len(vs.values)
}

// top returns the top value from the i-th stack.
func (vs *valueStack) top(i int) reflect.Value {
	return vs.values[i].Top()
}

// push appends a value to the i-th stack.
func (vs *valueStack) push(i int, v reflect.Value) {
	vs.values[i] = append(vs.values[i], v)
}

// addStack appends a new stack with the given initial value and fragment type.
func (vs *valueStack) addStack(v reflect.Value, fragmentType string) {
	vs.values = append(vs.values, []reflect.Value{v})
	vs.fragmentTypes = append(vs.fragmentTypes, fragmentType)
}

// popAll pops from all stacks, keeping only non-empty ones.
func (vs *valueStack) popAll() {
	var nonEmpty []stack
	var nonEmptyTypes []string
	for i := range vs.values {
		vs.values[i] = vs.values[i].Pop()
		if len(vs.values[i]) > 0 {
			nonEmpty = append(nonEmpty, vs.values[i])
			// Keep fragment type in sync
			if i < len(vs.fragmentTypes) {
				nonEmptyTypes = append(nonEmptyTypes, vs.fragmentTypes[i])
			} else {
				nonEmptyTypes = append(nonEmptyTypes, "")
			}
		}
	}
	vs.values = nonEmpty
	vs.fragmentTypes = nonEmptyTypes
}

// popLeftArrayTemplates removes the template element from all slice stacks.
func (vs *valueStack) popLeftArrayTemplates() {
	for i := range vs.values {
		v := vs.values[i].Top()
		// Unwrap pointers and interfaces to get to the actual slice
		v = reflectutil.UnwrapToConcreteValue(v)

		// Only call Slice if it's actually a slice type
		if v.IsValid() && v.Kind() == reflect.Slice {
			v.Set(v.Slice(1, v.Len()))
		}
	}
}

// fragmentType returns the fragment type for the i-th stack.
func (vs *valueStack) fragmentType(i int) string {
	if i < len(vs.fragmentTypes) {
		return vs.fragmentTypes[i]
	}
	return ""
}
