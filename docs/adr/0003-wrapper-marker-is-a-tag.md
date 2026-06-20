# A wrapper type is marked by the `wrapped:"true"` tag, not a method

---
Status: accepted
---

A *wrapper* is a struct made transparent to the library: where query construction
would expand a struct into a selection set, and decoding would unmarshal into its
fields, a wrapper instead splices its single inner field in its place. This lets a
Go-side container (generics ergonomics, a scalar that needs methods attached, a
dynamic container path) map to a GraphQL shape that only sees the inner type.

The marker for "this struct is a wrapper, and *this* is its wrapped field" is a
single struct tag, `wrapped:"true"`, on exactly one field — mirroring the existing
`scalar:"true"` precedent. It is the **single source of truth**: detection and
access both read it, so they cannot disagree. Query construction takes the tagged
field's *type* (purely at the type level — no value, no user code executed);
decoding writes into the tagged field directly.

We deliberately reject the previous design — a `GetGraphQLWrapped() T` method —
because it was a hack forced by Go generics, and it split the convention in two:

- **Generic method ⇒ no interface.** A generic wrapper declares
  `GetGraphQLWrapped() T`, not `() any`, so it never satisfies an
  `interface{ GetGraphQLWrapped() any }` (Go requires an exact signature). The
  precomputed `Implements` check was therefore dead, and detection had to fall
  back to `reflect.Value.MethodByName(...)` (with a per-type cache), plus a
  reflective `method.Call(nil)` on the query path *just to read a field*.
- **Two unsynchronised sources of truth.** The method drove query construction
  while a field named `Value` drove decoding. Nothing tied them together: a
  wrapper whose method returned one field while decode looked for `Value` would
  silently misbehave. The tag collapses both into one declaration.

## Considered options

- **`wrapped:"true"` tag, no method (chosen):** one declaration marks the type
  *and* locates the writable field. Detection and access share a single cached
  per-type field index (`reflectutil.wrappedFieldIndex`). Mirrors `scalar:"true"`.
- **Keep the method, but make it `GetGraphQLWrapped() any`:** would let
  `Implements` work and kill the `MethodByName` reflection — but only de-hacks
  *detection*. It keeps the second source of truth (the `Value` field for the
  writable decode target) and still requires a value to read the wrapped data.
  Rejected: fixes the symptom, not the split.
- **"Struct with exactly one field" heuristic, no marker:** rejected. Real
  wrappers carry siblings — the sole consumer's wrapper has a `gqlType` field
  next to the wrapped value — so "one field" is both wrong and too implicit
  (it would silently capture any single-field struct).
- **Fold into the `graphql` tag as `graphql:",wrapped"`:** rejected. It entangles
  "how to name this field" with "this field is the wrapper's payload", and forces
  the `graphql` tag parser (which today carries name + args + directives) to grow
  an options syntax. A dedicated tag key, like `scalar`, stays orthogonal.

## Consequences

- **Breaking change to the public convention** (a minor-version bump). The
  `GetGraphQLWrapped` method requirement is gone, and the dead public interface
  `types.GraphQLWrapper` (+ `GraphqlWrapperInterface`) is removed. Migrating a
  wrapper is mechanical: tag the wrapped field `wrapped:"true"` and delete the
  method. `GetGraphQLType()`, which names the field, is unaffected — it is a
  separate concern and stays a method.
- **The wrapped field need not be named `Value`** anymore; the tag, not the
  name, identifies it.
- **`reflectutil` shrinks** to `IsWrapperType` + `UnwrapValueField` (used by both
  the query and decode walkers). `UnwrapValue`, `UnwrapValueOrOriginal`,
  `WrapperMethodName`, `WrapperFieldName`, and the method cache are gone.
- **Anonymous embedding of a wrapper no longer drops sibling fields.** The old
  method-promotion design promoted `GetGraphQLWrapped` to the embedding struct,
  making *it* transparent and silently swallowing siblings. Tag detection does
  not promote (the tag lives on the concrete wrapper's field, not on the
  embedding field), so siblings are preserved. Only `GetGraphQLType()` is still
  promoted (ordinary Go behaviour), which affects naming. See
  `TestWrapper_EmbeddedAnonymous`.
- **Relationship to ADR-0002:** unchanged. Wrapper unwrapping remains query-side
  (and decode-side) walker logic; only the *marker mechanism* moved from a method
  to a tag. The two walkers stay separate.
- A future reader may look for a `GetGraphQLWrapped` method or a `GraphQLWrapper`
  interface and assume it was lost. It was not — read this ADR. The tag is the
  marker.
