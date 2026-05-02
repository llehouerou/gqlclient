# Changelog

All notable changes to this project are documented in this file.

## Unreleased

### Breaking changes

- Renamed exported error-code constants `ErrJsonEncode` and `ErrJsonDecode`
  to `ErrJSONEncode` and `ErrJSONDecode` to match Go naming conventions
  (initialisms in upper case). Callers comparing `err.GetCode()` against
  these constants must update the references. No behavior change beyond
  the rename.
- Removed the exported `ConstructSubscription` function. It built a
  GraphQL subscription string but the library has no transport to
  execute one (subscription support was deliberately removed in the fork
  — see README). Callers in the wild are not expected; the function had
  no executable counterpart on `Client`.

## v0.15.1

### Fixes

- `fix(decode):` preserve tag-parser matching for dynamic GraphQLType
  fields (regression from v0.15.0). When a struct field's type
  implements `types.GraphQLType` and `GetGraphQLType()` returns a
  parameterized expression (e.g. `"card(slug:$slug)"`) or an alias
  (e.g. `"x: card(...)"`), the JSON response key is the bare
  field/alias name. The v0.15.0 cached lookup compared the live
  GraphQLType result to the response key with raw string equality and
  missed those matches; the legacy v0.14 path ran the result through
  `tagparser.ParseGraphQLTag` to extract `FieldName`/`Alias` first.
  v0.15.1 restores the tag-parser step on the cached path.

## v0.15.0

### Performance

The typed JSON decoder (`internal/decode`) is **~2.1× faster** with **~31% fewer
bytes allocated** and **~17% fewer allocations** on a representative WebSocket
subscription payload (12 fields, 337 bytes, nested struct + slice). No public
API changes — drop-in upgrade.

Measured on a `BenchmarkUnmarshalGraphQL_WSOffer` decoding the same payload
into the same target type, `go test -bench -benchtime=5s -count=5`:

| Metric    | v0.14.0  | v0.15.0  | Δ            |
| --------- | -------- | -------- | ------------ |
| ns/op     | 57,000   | 27,240   | **2.09×**    |
| B/op      | 12,111   | 8,388    | **−31%**     |
| allocs/op | 435      | 359      | **−17%**     |
| MB/s      | 5.9      | 12.4     | **2.10×**    |

Three internal changes drive the win:

1. **Per-type field-lookup cache.** Struct field metadata (graphql tag name,
   alias, scalar flag, declaration order) is precomputed once per
   `reflect.Type` and reused for every subsequent decode, replacing the
   per-key linear scan that re-parsed tags on every JSON key. Fields whose
   type implements `types.GraphQLType` still resolve their match name
   from the live value at lookup time, preserving instance-dependent
   implementations like `ActualNodes[T].GetGraphQLType` in the test suite.

2. **Cached interface checks.** `reflectutil.ImplementsGraphQLType` and the
   `GetGraphQLWrapped` method-set check are now cached per `reflect.Type`,
   turning the hot-path reflection lookups into one `sync.Map.Load`.

3. **Direct scalar decoding.** Strings, bools, JSON numbers, and
   `json.RawMessage` are written into matching Go fields via
   `reflect.Set*` directly from the `json.Decoder.Token()` value. The old
   path round-tripped each scalar through `json.Marshal` +
   `json.Unmarshal`, which dominated allocations. Types implementing
   `json.Unmarshaler` (e.g. `time.Time`), interface targets, and
   raw-message transcoding still use the reference path.

### Notes

- All existing tests pass unchanged.
- New benchmark added: `internal/decode/ws_frame_benchmark_test.go`.
