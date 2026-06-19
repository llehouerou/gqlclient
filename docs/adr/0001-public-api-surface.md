# Public API exposes composable primitives, but not the HTTP round-trip

---
Status: accepted
---

The library is a ladder of composable primitives, not just a high-level client: `ConstructQuery`/`ConstructMutation` (build the query string), `BuildRequest` (build the `*http.Request` + body bytes), and `UnmarshalGraphQL` (decode a response) are all public so callers can enter at any rung. We deliberately keep the HTTP round-trip step (`executeRequest`: `Do` + gzip + status handling) **unexported**, even though it is the natural symmetric counterpart to `BuildRequest`.

The asymmetry is intentional: the valuable customization points are *before* send (sign the body, hash for persisted queries, dump the wire payload, hand the request to a custom executor) and *after* receive (decode it yourself), and those are exactly the rungs we expose. The round-trip in the middle has little standalone value and is the part most entangled with internal policy (gzip handling, the non-200 rejection rule); exporting it would commit us to that policy as a public contract. The transport substitution point callers actually need is already public — inject a `*http.Client` via `WithHTTPClient`.

## Considered options

- **Composable ladder (chosen):** export `BuildRequest` + the existing `ConstructQuery`/`UnmarshalGraphQL`; keep `executeRequest` internal.
- **Full ladder:** also export `ExecuteRequest`. Rejected — locks internal gzip/status policy into the public surface for a rung with weak independent value.
- **Minimal client:** unexport the build/decode primitives too. Rejected — `ConstructQuery` and `UnmarshalGraphQL` already have real external consumers.

## Consequences

- A future reader seeing `BuildRequest` public but no `ExecuteRequest` may assume the omission is an oversight and "fix" it for symmetry. It is not an oversight — read this ADR before exporting it.
- `executeRequest` is the single internal transport seam; production (`request`) and tests both exercise it, so its second adapter lives in an in-package test (`package graphql`), not the external `graphql_test` package.
- Exporting `executeRequest` later, if a concrete need appears, is a non-breaking change; that asymmetry of reversibility is part of why we default to keeping it internal now.
