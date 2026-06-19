# The query-body and argument walkers stay separate

---
Status: accepted
---

Query construction has two reflection walkers: one builds the **selection set**
from a result struct (`query_builder*.go` — recurses into fields, emits
`{a b c{d}}`), the other builds the **variable type expression** from a
variables struct/map (`query_arguments.go` — never recurses into fields, emits
`Int!`, `[String!]`). They look like candidates for a shared "type classifier,"
but we deliberately keep them separate.

The only classification both genuinely share — detecting a `GraphQLType` and
extracting its name — already lives as a deep module in `internal/reflectutil`
(`ImplementsGraphQLType`, `GetGraphQLType`, `GetGraphQLTypeFromType`), and both
walkers call it. Everything else is walker-specific and does **not** overlap:
the query side owns expand-or-not (`isScalarType`, the `scalar:"true"` tag),
wrapper unwrapping, and field naming; the argument side owns kind→type-name
mapping (`Int`/`Float`/`Boolean`/`[…]`) and pointer→nullable (`!`). "Nullable"
is meaningless to the query side; "scalar/expand" is meaningless to the
argument side.

## Considered options

- **Keep them separate (chosen):** the shared core is already extracted into
  `reflectutil`; each walker keeps its own output-specific logic.
- **Extract a shared `classify(type) → {kind, nullable, scalar}` seam:**
  rejected. Each field of that result would be consumed by only one walker —
  a hypothetical seam (one adapter), not a real one (two adapters). It would
  add a shallow module and couple two fundamentally different emitters (a
  selection set vs. a type expression) for no leverage.

## Consequences

- A future architecture review will see "two reflection walkers" and may
  propose unifying them. It is not an oversight — read this ADR first. The
  shared classification is already as DRY as it should be.
- Revisit only if a **third** consumer of Go-type → GraphQL classification
  appears, or if the two walkers start needing the *same* output shape. Until
  then, two adapters do not exist, so the seam should not.
