// Package graphql provides a reflection-based GraphQL client for Go,
// focused on query and mutation operations over HTTP.
//
// Queries are constructed from Go structs whose fields carry `graphql:"..."`
// tags. Reflection walks the struct to build the GraphQL query string and to
// unmarshal the response into the same struct, including support for inline
// fragments on unions and interfaces, custom scalars, and ordered maps for
// mutations that require deterministic field ordering.
//
// The Client is immutable: With* methods return a new Client without
// modifying the receiver, so a single base Client can be safely shared across
// goroutines and customized per-call.
//
// This package does not provide WebSocket subscriptions, file uploads, or
// persisted-query support; it is intentionally scoped to HTTP query and
// mutation traffic.
package graphql
