# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

A simplified GraphQL client library for Go that focuses on HTTP-based query and mutation operations. The library uses reflection to construct GraphQL queries from Go structs and unmarshal responses back into those structs.

## Key Commands

### Using Makefile (Recommended)
```bash
# Build the project
make build

# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report (generates coverage.html)
make test-coverage

# Run linters (golangci-lint with custom config)
make lint

# Format code with gofmt
make fmt

# Format code with golines (80 char lines, uses goimports-reviser)
make format

# Run go vet
make vet

# Clean build artifacts
make clean
```

### Direct Go Commands
```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test .
go test ./pkg/jsonutil
go test ./ident

# Run a single test
go test -run TestName

# Run tests with verbose output
go test -v ./...

# Build (library, no main package)
go build ./...

# Install dependencies
go mod download
go mod tidy
```

## Code Quality & Modernization

### Go Version
- **Minimum**: Go 1.25
- Uses modern Go features: `any` type alias, `io.ReadAll`, `reflect.PointerTo`

### Code Formatting
- **golines**: Enforces 80-character line limits with automatic wrapping
  - Command: `make format`
  - Config: Uses `goimports-reviser` as base formatter, preserves struct tags

### Linting
- **golangci-lint v2.6**: Comprehensive static analysis
  - Config: `.golangci.yml` (version 2 format)
  - Enabled linters: errcheck, govet, ineffassign, staticcheck, unused, misspell, unconvert

### Dependencies
- `google/uuid`: v1.6.0 (only external dependency)

### Error Handling
- Explicit error handling using blank identifier (`_ =`) where appropriate
- Proper error bubbling in cleanup paths
- Uses `//nolint` directives with explanations for intentional suppressions

## Architecture

### Core Components

**1. Client (`graphql.go`)**: The main GraphQL HTTP client
- Handles query and mutation operations via POST requests
- Supports debug mode with detailed request/response logging
- Uses reflection to construct queries from Go structs via `query.go`
- Returns structured `Errors` type for GraphQL errors

**2. Query Construction (`query.go`, `query_*.go`)**: Reflection-based query builder
- Main functions: `ConstructQuery`, `ConstructMutation`
- Uses Go struct tags (`graphql:"..."`) to specify GraphQL field names and arguments
- Supports variables with type inference from Go types
- Handles inline fragments with `... on TypeName` syntax
- Supports operation names and directives via `Option` interface

**3. JSON Unmarshaling (`pkg/jsonutil/graphql.go`)**: Custom JSON decoder
- `UnmarshalGraphQL()` decodes GraphQL responses into Go structs
- Handles GraphQL-specific patterns: fragments, embedded structs, ordered maps
- Uses `__typename` discrimination for union/interface type resolution
- Supports both struct fields and ordered maps (`[][2]interface{}`)
- Includes fragment matching logic to populate only the correct union/interface variant

### Key Patterns

**Struct Tags**: The `graphql` struct tag drives both query construction and unmarshaling:
- Field arguments: `graphql:"height(unit: METER)"`
- Variables: `graphql:"human(id: $id)"`
- Inline fragments: `graphql:"... on Droid"`
- Skip field: `graphql:"-"`
- Custom scalars: `scalar:"true"` (prevents expansion during query generation)

**Type System**:
- `ID` type for GraphQL IDs
- `GraphQLType` interface (`types/types.go`) for custom type name specification
- Reflection-based type mapping: Go types → GraphQL types
- Special handling for slices (lists), pointers (nullable), interfaces

**Error Handling**:
- GraphQL errors are returned as `Errors` (slice of `Error`)
- Partial data can exist alongside errors
- Debug mode adds request/response details to error extensions

### Fragment Matching

The library handles GraphQL unions and interfaces using inline fragments with `__typename` discrimination:
- During unmarshaling, captures `__typename` from response
- Filters inline fragments to populate only the matching type
- Supports both struct fields and ordered map keys as fragments
- See `pkg/jsonutil/graphql.go` for fragment filtering logic

### Ordered Maps

GraphQL requires fields in specific order for mutations. Use `[][2]interface{}` instead of regular maps:
```go
// [][2]interface{} is treated as an ordered map
m := [][2]interface{}{
    {"createUser(login: $login1)", &CreateUser{}},
    {"createUser(login: $login2)", &CreateUser{}},
}
```

## Important Implementation Details

1. **Variable Requirements**: When constructing queries with variables, both the struct tag must reference the variable (e.g., `$id`) AND the variable must be passed in the variables map.

2. **Scalar Types**: Types implementing `json.Unmarshaler` or `ID` are treated as scalars and not recursively expanded during query construction.

3. **Template Slices**: When unmarshaling arrays, the first element in the target slice acts as a template that gets copied for each array item.

4. **Pointer vs Value**: Pointer types in structs indicate optional/nullable GraphQL fields. Value types are required (append `!` in GraphQL schema).

5. **Pre-built Queries**: `Exec()` and `ExecRaw()` methods allow executing dynamically-constructed query strings (useful for CLI tools or dynamic filtering).
