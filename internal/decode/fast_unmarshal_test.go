package decode

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Direct unit tests for fastUnmarshal, exercising each branch plus the
// error and bypass paths. These are easy to regress in a refactor and
// the end-to-end tests do not reach the overflow / strict-typing edges.

type customUnmarshaler struct {
	bytes []byte
}

func (c *customUnmarshaler) UnmarshalJSON(b []byte) error {
	c.bytes = append([]byte(nil), b...)
	return nil
}

type namedString string
type namedInt64 int64
type namedUint8 uint8
type namedFloat64 float64

func addressableOf[T any](v T) reflect.Value {
	return reflect.ValueOf(&v).Elem()
}

func TestFastUnmarshal_StringIntoString(t *testing.T) {
	v := addressableOf("")
	handled, err := fastUnmarshal("hello", v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if got := v.String(); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestFastUnmarshal_StringIntoNamedString(t *testing.T) {
	var s namedString
	v := reflect.ValueOf(&s).Elem()
	handled, err := fastUnmarshal("OPEN", v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if s != "OPEN" {
		t.Errorf("got %q, want OPEN", s)
	}
}

func TestFastUnmarshal_StringIntoPointer(t *testing.T) {
	var p *string
	v := reflect.ValueOf(&p).Elem()
	handled, err := fastUnmarshal("hi", v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if p == nil || *p != "hi" {
		t.Errorf("got %v, want pointer to \"hi\"", p)
	}
}

func TestFastUnmarshal_StringIntoNonStringDeclines(t *testing.T) {
	var i int
	v := reflect.ValueOf(&i).Elem()
	handled, _ := fastUnmarshal("hi", v)
	if handled {
		t.Error("expected fast path to decline string -> int, got handled=true")
	}
}

func TestFastUnmarshal_BoolIntoBool(t *testing.T) {
	v := addressableOf(false)
	handled, err := fastUnmarshal(true, v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if !v.Bool() {
		t.Error("expected true")
	}
}

func TestFastUnmarshal_BoolIntoPointer(t *testing.T) {
	var p *bool
	v := reflect.ValueOf(&p).Elem()
	handled, err := fastUnmarshal(true, v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if p == nil || *p != true {
		t.Errorf("got %v, want pointer to true", p)
	}
}

func TestFastUnmarshal_NumberIntoIntKinds(t *testing.T) {
	var i64 int64
	v := reflect.ValueOf(&i64).Elem()
	handled, err := fastUnmarshal(json.Number("42"), v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if i64 != 42 {
		t.Errorf("got %d, want 42", i64)
	}

	var ni namedInt64
	v = reflect.ValueOf(&ni).Elem()
	handled, err = fastUnmarshal(json.Number("-7"), v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if ni != -7 {
		t.Errorf("got %d, want -7", ni)
	}
}

func TestFastUnmarshal_NumberIntoUintKinds(t *testing.T) {
	var u8 namedUint8
	v := reflect.ValueOf(&u8).Elem()
	handled, err := fastUnmarshal(json.Number("200"), v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if u8 != 200 {
		t.Errorf("got %d, want 200", u8)
	}
}

func TestFastUnmarshal_NumberIntoFloat(t *testing.T) {
	var f namedFloat64
	v := reflect.ValueOf(&f).Elem()
	handled, err := fastUnmarshal(json.Number("3.14"), v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if f != 3.14 {
		t.Errorf("got %v, want 3.14", f)
	}
}

func TestFastUnmarshal_NumberIntoJSONNumberPreservesString(t *testing.T) {
	var n json.Number
	v := reflect.ValueOf(&n).Elem()
	handled, err := fastUnmarshal(
		json.Number("9999999999999999999"), v,
	)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if string(n) != "9999999999999999999" {
		t.Errorf("got %q, want raw 19-digit string", n)
	}
}

func TestFastUnmarshal_IntOverflowReportsErrorNotFallthrough(t *testing.T) {
	// A 999 token cannot fit in int8. fastUnmarshal must surface the
	// error rather than declining (which would silently re-error from
	// the slow path with a more cryptic message).
	var i int8
	v := reflect.ValueOf(&i).Elem()
	handled, err := fastUnmarshal(json.Number("999"), v)
	if !handled {
		t.Fatal("expected handled=true on overflow, got false")
	}
	if err == nil {
		t.Fatal("expected overflow error, got nil")
	}
}

func TestFastUnmarshal_UintOverflow(t *testing.T) {
	var u uint8
	v := reflect.ValueOf(&u).Elem()
	handled, err := fastUnmarshal(json.Number("256"), v)
	if !handled || err == nil {
		t.Fatalf("expected handled overflow error, got handled=%v err=%v",
			handled, err)
	}
}

func TestFastUnmarshal_NegativeIntoUintDeclines(t *testing.T) {
	// strconv.ParseUint("-1") fails; fast path must decline so the
	// slow path can produce the canonical encoding/json error.
	var u uint64
	v := reflect.ValueOf(&u).Elem()
	handled, _ := fastUnmarshal(json.Number("-1"), v)
	if handled {
		t.Error("expected decline for -1 into uint64, got handled=true")
	}
}

func TestFastUnmarshal_FractionalIntoIntDeclines(t *testing.T) {
	// 1.5 cannot ParseInt; defer to the slow path so the user sees
	// the encoding/json error verbatim.
	var i int64
	v := reflect.ValueOf(&i).Elem()
	handled, _ := fastUnmarshal(json.Number("1.5"), v)
	if handled {
		t.Error("expected decline for 1.5 into int64, got handled=true")
	}
}

func TestFastUnmarshal_RawMessageDeclinesSoSlowPathCompacts(t *testing.T) {
	// RawMessage is itself a json.Unmarshaler implementation and the
	// existing semantics depend on json.Marshal compacting whitespace
	// inside the value (see TestUnmarshalGraphQL_jsonRawTag). Fast
	// path must decline so the slow path preserves that contract.
	var rm json.RawMessage
	v := reflect.ValueOf(&rm).Elem()
	handled, _ := fastUnmarshal(json.RawMessage(`{"x":1}`), v)
	if handled {
		t.Error("expected decline for RawMessage -> RawMessage")
	}

	var i int
	vi := reflect.ValueOf(&i).Elem()
	handled, _ = fastUnmarshal(json.RawMessage(`42`), vi)
	if handled {
		t.Error("expected decline for RawMessage -> int")
	}
}

func TestFastUnmarshal_NullZeroesScalar(t *testing.T) {
	v := addressableOf("nonempty")
	handled, err := fastUnmarshal(nil, v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if v.String() != "" {
		t.Errorf("got %q, want zero string", v.String())
	}
}

func TestFastUnmarshal_NullZeroesPointer(t *testing.T) {
	s := "x"
	p := &s
	v := reflect.ValueOf(&p).Elem()
	handled, err := fastUnmarshal(nil, v)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if p != nil {
		t.Errorf("got %v, want nil pointer", p)
	}
}

func TestFastUnmarshal_NullIntoInterfaceDeclines(t *testing.T) {
	var iface any
	v := reflect.ValueOf(&iface).Elem()
	handled, _ := fastUnmarshal(nil, v)
	if handled {
		t.Error("expected decline for null -> interface, got handled=true")
	}
}

func TestFastUnmarshal_InterfaceTargetDeclines(t *testing.T) {
	var iface any
	v := reflect.ValueOf(&iface).Elem()
	handled, _ := fastUnmarshal("hi", v)
	if handled {
		t.Error("expected decline for any -> interface, got handled=true")
	}
}

func TestFastUnmarshal_JSONUnmarshalerDeclines(t *testing.T) {
	// Custom UnmarshalJSON must win over the kind-based fast path.
	// fastUnmarshal must decline so the slow path runs the user's
	// UnmarshalJSON method.
	var c customUnmarshaler
	v := reflect.ValueOf(&c).Elem()
	handled, _ := fastUnmarshal("hi", v)
	if handled {
		t.Error(
			"expected decline for json.Unmarshaler target, got handled=true",
		)
	}
}

func TestFastUnmarshal_PointerToJSONUnmarshalerDeclines(t *testing.T) {
	var p *customUnmarshaler
	v := reflect.ValueOf(&p).Elem()
	handled, _ := fastUnmarshal("hi", v)
	if handled {
		t.Error(
			"expected decline for *json.Unmarshaler target, got handled=true",
		)
	}
}

func TestImplementsJSONUnmarshalerCacheStable(t *testing.T) {
	t1 := reflect.TypeOf(customUnmarshaler{})
	t2 := reflect.TypeOf("")
	if !implementsJSONUnmarshaler(t1) {
		t.Error("customUnmarshaler should be detected as Unmarshaler")
	}
	if implementsJSONUnmarshaler(t2) {
		t.Error("string should not be detected as Unmarshaler")
	}
	// Hit the cache a few more times and ensure the answer is stable.
	for range 4 {
		if !implementsJSONUnmarshaler(t1) || implementsJSONUnmarshaler(t2) {
			t.Fatal("cache flipped on repeat lookup")
		}
	}
}
