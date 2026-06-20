package decode_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/llehouerou/gqlclient/internal/decode"
)

// Concurrent decode test exercises the per-type sync.Map caches added in
// v0.15.0 (field lookup, ImplementsGraphQLType, wrappedFieldIndex,
// implementsJSONUnmarshaler) under contention. Run with `go test -race`
// to catch any unsafe access.
//
// Multiple types are mixed so first-encounter races (LoadOrStore) and
// repeat lookups happen in parallel.

type concA struct {
	ID    string `graphql:"id"`
	Count int64  `graphql:"count"`
}

type concB struct {
	Name  string  `graphql:"name"`
	Score float64 `graphql:"score"`
	Items []struct {
		Tag string `graphql:"tag"`
	} `graphql:"items"`
}

type concC struct {
	Active bool   `graphql:"active"`
	Note   string `graphql:"note"`
}

var (
	payloadA = []byte(`{"id":"x","count":42}`)
	payloadB = []byte(
		`{"name":"abc","score":1.5,"items":[{"tag":"t1"},{"tag":"t2"}]}`,
	)
	payloadC = []byte(`{"active":true,"note":"ok"}`)
)

func TestConcurrentDecode_RaceFreeAcrossTypes(t *testing.T) {
	t.Parallel()

	const goroutines = 32
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := range goroutines {
		go func(seed int) {
			defer wg.Done()
			for i := range iterations {
				switch (seed + i) % 3 {
				case 0:
					var a concA
					if err := decode.UnmarshalGraphQL(payloadA, &a); err != nil {
						t.Errorf("A: %v", err)
						return
					}
					if a.ID != "x" || a.Count != 42 {
						t.Errorf("A wrong: %+v", a)
						return
					}
				case 1:
					var b concB
					if err := decode.UnmarshalGraphQL(payloadB, &b); err != nil {
						t.Errorf("B: %v", err)
						return
					}
					if b.Name != "abc" || b.Score != 1.5 ||
						len(b.Items) != 2 {
						t.Errorf("B wrong: %+v", b)
						return
					}
				case 2:
					var c concC
					if err := decode.UnmarshalGraphQL(payloadC, &c); err != nil {
						t.Errorf("C: %v", err)
						return
					}
					if !c.Active || c.Note != "ok" {
						t.Errorf("C wrong: %+v", c)
						return
					}
				}
			}
		}(g)
	}
	wg.Wait()
}

// recordingUnmarshaler is an end-to-end witness: if implementsJSONUnmarshaler
// caches incorrectly or fastUnmarshal stops gating on it, this type's
// UnmarshalJSON would be skipped and the recorded marker would be empty.

type recordingUnmarshaler struct {
	raw string
}

func (r *recordingUnmarshaler) UnmarshalJSON(b []byte) error {
	// Trim wrapping quotes for ergonomic assertions.
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		r.raw = string(b[1 : len(b)-1])
		return nil
	}
	r.raw = string(b)
	return nil
}

func TestUnmarshalGraphQL_RoutesCustomUnmarshalerThroughSlowPath(t *testing.T) {
	t.Parallel()

	type query struct {
		Custom recordingUnmarshaler `graphql:"custom"`
	}
	var got query
	err := decode.UnmarshalGraphQL(
		[]byte(`{"custom":"hello"}`), &got,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Custom.raw != "hello" {
		t.Errorf(
			"UnmarshalJSON not invoked: got raw=%q, want \"hello\"",
			got.Custom.raw,
		)
	}
}

func TestUnmarshalGraphQL_PointerCustomUnmarshalerStillSlowPath(t *testing.T) {
	t.Parallel()

	type query struct {
		Custom *recordingUnmarshaler `graphql:"custom"`
	}
	var got query
	err := decode.UnmarshalGraphQL(
		[]byte(`{"custom":"world"}`), &got,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Custom == nil || got.Custom.raw != "world" {
		t.Errorf(
			"UnmarshalJSON not invoked on pointer: got %+v",
			got.Custom,
		)
	}
}

// Sanity check that json.Number stays raw when targeted at a
// json.Number field, end-to-end.
func TestUnmarshalGraphQL_PreservesJSONNumberPrecision(t *testing.T) {
	t.Parallel()

	type query struct {
		N json.Number `graphql:"n"`
	}
	var got query
	err := decode.UnmarshalGraphQL(
		[]byte(`{"n":9999999999999999999}`), &got,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.N) != "9999999999999999999" {
		t.Errorf(
			"json.Number lost precision: got %q, want raw 19-digit string",
			got.N,
		)
	}
}
