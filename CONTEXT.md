# gqlclient

A Go client that builds GraphQL operations from Go structs, sends them over HTTP, and unmarshals the response back into those structs. This glossary fixes the vocabulary the library and its users share — especially the error taxonomy, where precise terms matter because a single response can succeed and fail at the same time.

## Language

### Error taxonomy

A response travels through three stages — reach the server, get a GraphQL answer, fit that answer into your struct — and a failure at each stage is a different kind of error. The library keeps them distinct because callers handle them differently, and surfaces each as a typed sentinel they can match with `errors.Is`.

**Transport error**:
The request never came back as a usable GraphQL response — the HTTP call failed, the server answered non-200, or the response body could not be decompressed. Surfaced as `ErrRequest`.
_Avoid_: network error, HTTP error (too narrow — a non-200 with a body is also a transport error here).

**GraphQL error**:
The server returned a well-formed response carrying an `errors[]` array, per the GraphQL spec. May arrive *alongside* partial `data` — a response is not all-or-nothing. This is the server reporting a problem with the operation, not with the transport.
_Avoid_: server error, query error.

**Decode error**:
The transport succeeded and `data` is present, but it could not be unmarshalled into the caller's target struct (shape mismatch, bad scalar). A client-side failure to interpret a successful response. Surfaced as `ErrGraphQLDecode`.
_Avoid_: parse error, unmarshal error (reserve those for the lower-level JSON layer).

### Example dialogue

> **Dev:** The call returned an error — is the server down?
> **Maintainer:** Which kind? If it's a **transport error**, the request never landed a usable response — connection refused, a 500, a broken gzip stream. If it's a **GraphQL error**, the server answered fine but reported a problem in `errors[]`; you might still have partial `data`.
> **Dev:** And if the server answered 200 with clean data but my call still failed?
> **Maintainer:** That's a **decode error** — transport and GraphQL both succeeded, but the `data` didn't fit your struct. The fix is on your side, not the server's.
