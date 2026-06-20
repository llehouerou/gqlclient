# Changelog

All notable changes to this project are documented in this file.

## Unreleased

Internal cleanup of the request/response path: two divergent HTTP
execution paths are collapsed into one (fixing two latent bugs), and
error decoration is consolidated behind the error module. See
`docs/adr/0001-public-api-surface.md` for the API-shape rationale. The
wrapper-type convention also moves from a generic `GetGraphQLWrapped()`
method to a single `wrapped:"true"` struct tag — see
`docs/adr/0003-wrapper-marker-is-a-tag.md`.

### Breaking changes

- Removed the exported `Client.ExecuteRequest` method. The HTTP round
  trip (send, gzip-decompress, reject non-200) is now an internal seam
  used by `Query`/`Mutate` and the pre-built `Execute*` methods. To
  drive transport manually, build the request with `Client.BuildRequest`
  and send it with your own `*http.Client`; to swap the transport,
  inject one via `Client.WithHTTPClient`. `BuildRequest` stays public.

- A corrupt or invalid gzip response stream now classifies as
  `ErrRequest` (`ErrCodeRequest`) instead of `ErrJSONDecode`
  (`ErrCodeJSONDecode`) — a broken transport encoding is a transport
  failure, not a JSON-decode failure. Update any
  `errors.Is(err, ErrJSONDecode)` check that targeted gzip failures to
  `errors.Is(err, ErrRequest)`.

- Removed the exported `Client.NewRequestError` and `Client.DecorateError`
  methods. Error decoration is now internal: the request/response context
  is passed as one bundle and the debug-decoration policy lives in a
  single place. Per ADR-0001, error *construction* is machinery — consume
  errors via `errors.Is`, `Error.GetCode`, and `Error.GetInternalExtensions`,
  which stay public.

- The wrapper-type marker is now the `wrapped:"true"` struct tag instead of
  a `GetGraphQLWrapped()` method. A wrapper is a struct with exactly one field
  tagged `wrapped:"true"`; that tag both marks the type and locates the wrapped
  field, which need no longer be named `Value`. The exported
  `types.GraphQLWrapper` interface (and `types.GraphqlWrapperInterface`) is
  removed. To migrate, tag the wrapped field and delete the method:

  ```go
  // before
  type Wrapper[T any] struct{ Value T }
  func (w Wrapper[T]) GetGraphQLWrapped() T { return w.Value }

  // after
  type Wrapper[T any] struct {
      Value T `wrapped:"true"`
  }
  ```

  As a side effect, anonymously embedding a wrapper no longer drops the
  embedding struct's sibling fields (method promotion previously made the
  parent transparent). See `docs/adr/0003-wrapper-marker-is-a-tag.md`.

### Fixes

- A non-200 response carrying a gzip-encoded body now reports the
  decompressed body in the error message instead of the raw gzip magic
  bytes.
- The gzip reader is now closed after use (previously skipped through
  the transport helper, leaving the trailing CRC unverified).

## v0.16.0 (2026-05-02)

This release adds typed sentinel errors and ergonomic transport helpers
on top of the existing immutable-Client pattern. There are breaking
changes in the error-code constants — see the migration table below.

### Breaking changes

- Renamed the exported error-code string constants to use an `ErrCode`
  prefix, freeing up the short names for typed sentinel error values:

  | v0.15.x             | v0.16.0                  |
  | ------------------- | ------------------------ |
  | `ErrRequestError`   | `ErrCodeRequest`         |
  | `ErrJsonEncode`     | `ErrCodeJSONEncode`      |
  | `ErrJsonDecode`     | `ErrCodeJSONDecode`      |
  | `ErrGraphQLEncode`  | `ErrCodeGraphQLEncode`   |
  | `ErrGraphQLDecode`  | `ErrCodeGraphQLDecode`   |

  Callers comparing `err.GetCode()` against the old constants should
  update to the `ErrCode*` form, or — preferably — switch to
  `errors.Is(err, gqlclient.ErrJSONDecode)` and friends (see
  Additions). The string values stored in `Error.Extensions["code"]`
  are unchanged.

- Removed the exported `ConstructSubscription` function. It built a
  GraphQL subscription string but had no transport behind it (the
  fork is HTTP-only by design). No replacement is needed.

### Additions

- **Typed sentinel errors and `errors.Is`/`errors.As` support.**
  `ErrRequest`, `ErrJSONEncode`, `ErrJSONDecode`, `ErrGraphQLEncode`,
  `ErrGraphQLDecode` are now sentinel `error` values. `Errors` and
  `Error` implement `Is` / `Unwrap` (both single and multi), so the
  standard error-inspection patterns just work:

  ```go
  if err := client.Query(ctx, &q, nil); err != nil {
      switch {
      case errors.Is(err, gqlclient.ErrJSONDecode):
          // server returned non-JSON or malformed JSON
      case errors.Is(err, gqlclient.ErrRequest):
          // HTTP transport / non-200 status
      default:
          var gqlErrs gqlclient.Errors
          _ = errors.As(err, &gqlErrs) // server-level errors
      }
  }
  ```

  Locally generated errors also expose their cause via `Unwrap()` —
  e.g. recovering a `*json.SyntaxError` behind an `ErrJSONDecode`.

- **Ergonomic transport helpers:** `Client.WithHTTPClient`,
  `WithHeader`, `WithHeaders`, `WithUserAgent`. All follow the
  existing immutable pattern and are composable. `WithHeader(s)`
  apply before any `RequestModifier`, so a modifier can still
  override.

- **Runnable godoc examples** in `example_test.go` (visible on
  pkg.go.dev) for `NewClient`, `Query`, and each `With*` method.

- **`FuzzUnmarshalGraphQL`** in `internal/decode` — coverage-guided
  fuzz test for the custom JSON decoder, exercising struct,
  fragment, and ordered-map targets. Run on demand with
  `go test ./internal/decode -fuzz=FuzzUnmarshalGraphQL -fuzztime=30s`.

### Internal

- All test functions and subtests now call `t.Parallel()`. The test
  suite was audited for shared mutable state (no `t.Setenv`, no
  package-level test mutation, all fixtures read-only) and confirmed
  safe to parallelize. Wall-clock test time drops accordingly.
- Strict `golangci-lint` v2 baseline introduced earlier in the cycle
  is now load-bearing across the codebase: every error suppression
  is either an annotated `//nolint` with reason or a function on the
  exclude list (`*bytes.Buffer.Write*`, `*strings.Builder.Write*`)
  documented to never error.
- GitHub Actions CI (Linux via Nix dev shell, macOS, Windows) plus
  `dependabot` on `gomod` and `github-actions`. `master` is now a
  protected branch (no force-push, no deletion, linear history).
- Governance: `CONTRIBUTING.md`, `SECURITY.md`, package-level
  `doc.go` for pkg.go.dev rendering.

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
