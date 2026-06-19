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

// entry pairs a value stack with the fragment typename that gates it. A single
// JSON object may be unmarshalled into several entries at once — one per inline
// fragment or embedded struct. fragType is "" when the stack is not a fragment
// (the root, or an embedded struct) and is fixed for the entry's whole life.
type entry struct {
	stack    stack
	fragType string
}

// valueStack manages multiple value stacks to unmarshal into. Pairing each
// stack with its fragment typename in a single entry makes the two impossible
// to desync — there are no parallel slices to keep aligned by hand.
type valueStack struct {
	// entries holds one stack per simultaneous unmarshal target. Multiple
	// entries exist because we might unmarshal a single JSON value into
	// multiple GraphQL fragments or embedded structs at once.
	entries []entry
}

// len returns the number of value stacks.
func (vs *valueStack) len() int {
	return len(vs.entries)
}

// top returns the top value from the i-th stack.
func (vs *valueStack) top(i int) reflect.Value {
	return vs.entries[i].stack.Top()
}

// push appends a value to the i-th stack.
func (vs *valueStack) push(i int, v reflect.Value) {
	vs.entries[i].stack = append(vs.entries[i].stack, v)
}

// addStack appends a new stack with the given initial value and fragment type.
func (vs *valueStack) addStack(v reflect.Value, fragmentType string) {
	vs.entries = append(vs.entries, entry{
		stack:    stack{v},
		fragType: fragmentType,
	})
}

// popAll pops from all stacks, keeping only non-empty ones. Each surviving
// entry carries its fragment type with it, so the two can never drift apart.
func (vs *valueStack) popAll() {
	var kept []entry
	for i := range vs.entries {
		vs.entries[i].stack = vs.entries[i].stack.Pop()
		if len(vs.entries[i].stack) > 0 {
			kept = append(kept, vs.entries[i])
		}
	}
	vs.entries = kept
}

// popLeftArrayTemplates removes the template element from all slice stacks.
func (vs *valueStack) popLeftArrayTemplates() {
	for i := range vs.entries {
		v := vs.entries[i].stack.Top()
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
	return vs.entries[i].fragType
}
